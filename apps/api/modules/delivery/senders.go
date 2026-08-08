package delivery

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/smtp"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/FacileStudio/Antenne/apps/api/internal/envelope"
	"github.com/FacileStudio/Antenne/apps/api/modules/settings"
)

// smtpEHLOName is what Antenne calls itself in EHLO. RFC 5321 wants a resolvable
// FQDN or an address literal here, and a bare word is neither — strict receivers
// reject it. Left as-is for now because changing it changes how every existing
// relay sees us, but it belongs in configuration.
const smtpEHLOName = "antenne"

func (s *Sender) sendDiscord(ctx context.Context, target settings.DeliveryTarget, message string) error {
	body, err := json.Marshal(map[string]string{"content": envelope.HTMLToPlainText(message)})
	if err != nil {
		return err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, target.WebhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", "Antenne/1.0 uptime-bot")

	return s.expectOK(request, "Discord webhook")
}

func (s *Sender) sendNtfy(ctx context.Context, target settings.DeliveryTarget, message string) error {
	base := target.ServerURL
	if !strings.HasSuffix(base, "/") {
		base += "/"
	}
	endpoint, err := url.Parse(base)
	if err != nil {
		return err
	}
	endpoint = endpoint.JoinPath(target.Topic)

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(),
		strings.NewReader(envelope.HTMLToPlainText(message)))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "text/plain; charset=utf-8")
	request.Header.Set("User-Agent", "Antenne/1.0 uptime-bot")
	request.Header.Set("Title", "Antenne Alert")
	if target.AccessToken != "" {
		request.Header.Set("Authorization", "Bearer "+target.AccessToken)
	}
	if target.Priority != nil {
		request.Header.Set("Priority", strconv.Itoa(*target.Priority))
	}
	if len(target.Tags) > 0 {
		request.Header.Set("Tags", strings.Join(target.Tags, ","))
	}
	if target.ClickURL != nil && *target.ClickURL != "" {
		request.Header.Set("Click", *target.ClickURL)
	}

	return s.expectOK(request, "ntfy")
}

func (s *Sender) sendHTTP(ctx context.Context, target settings.DeliveryTarget, message Message) error {
	body, err := encodeEnvelope(message, target.IncludeRawPayload)
	if err != nil {
		return err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, target.URL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header = envelope.SignedHeaders(deref(target.SecretValue), message.Envelope.ID, body)

	return s.expectOK(request, "HTTP delivery webhook")
}

func (s *Sender) sendPerception(ctx context.Context, target settings.DeliveryTarget, message Message) error {
	body, err := encodeEnvelope(message, target.IncludeRawPayload)
	if err != nil {
		return err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, target.URL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header = envelope.SignedHeaders(deref(target.SecretValue), message.Envelope.ID, body)
	request.Header.Set("x-perception-workspace", target.Workspace)

	return s.expectOK(request, "Perception")
}

// sendSMTP speaks SMTP directly rather than through a mailer dependency: Antenne
// sends one short multipart message per alert and needs no queue, no templating
// and no attachments.
func (s *Sender) sendSMTP(target settings.DeliveryTarget, message string) error {
	address := net.JoinHostPort(target.Host, strconv.Itoa(target.Port))

	var client *smtp.Client
	var err error

	if target.Secure {
		// Implicit TLS, the submission-over-465 shape.
		connection, dialErr := tls.Dial("tcp", address, &tls.Config{ServerName: target.Host})
		if dialErr != nil {
			return dialErr
		}
		client, err = smtp.NewClient(connection, target.Host)
		if err != nil {
			connection.Close()
			return err
		}
	} else {
		client, err = smtp.Dial(address)
		if err != nil {
			return err
		}
	}
	defer client.Close()

	/*
	 * Hello has to come first, and this ordering is the whole reason submission
	 * over 587 never worked: Extension and StartTLS both call the package's
	 * private hello() internally, and Hello then refuses with "Hello called
	 * after other methods" — so every STARTTLS send failed before it reached the
	 * password. Implicit TLS on 465 was unaffected, because NewClient does not
	 * greet, which is why the bug hid.
	 *
	 * StartTLS re-issues EHLO itself once the handshake is up, which is what
	 * RFC 3207 requires (the pre-TLS capability list must be discarded), so
	 * greeting before it is correct rather than merely earlier.
	 */
	if err := client.Hello(smtpEHLOName); err != nil {
		return err
	}

	if !target.Secure {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(&tls.Config{ServerName: target.Host}); err != nil {
				return err
			}
		}
	}

	if target.Username != "" && target.Password != nil && *target.Password != "" {
		auth := smtp.PlainAuth("", target.Username, *target.Password, target.Host)
		if err := client.Auth(auth); err != nil {
			return err
		}
	}

	if err := client.Mail(sanitizeAddress(target.FromAddress)); err != nil {
		return err
	}
	for _, recipient := range target.ToAddresses {
		if err := client.Rcpt(sanitizeAddress(recipient)); err != nil {
			return err
		}
	}

	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := writer.Write(buildMIME(target, message)); err != nil {
		writer.Close()
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}

	return client.Quit()
}

func buildMIME(target settings.DeliveryTarget, message string) []byte {
	subject := "Antenne Alert"
	if prefix := sanitizeHeader(target.SubjectPrefix); prefix != "" {
		subject = prefix + " Antenne Alert"
	}

	recipients := make([]string, 0, len(target.ToAddresses))
	for _, recipient := range target.ToAddresses {
		recipients = append(recipients, sanitizeHeader(recipient))
	}

	boundary := "antenne-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	plain := envelope.HTMLToPlainText(message)

	var body bytes.Buffer
	fmt.Fprintf(&body, "From: %s\r\n", sanitizeHeader(target.FromAddress))
	fmt.Fprintf(&body, "To: %s\r\n", strings.Join(recipients, ", "))
	fmt.Fprintf(&body, "Subject: %s\r\n", subject)
	body.WriteString("MIME-Version: 1.0\r\n")
	fmt.Fprintf(&body, "Content-Type: multipart/alternative; boundary=\"%s\"\r\n\r\n", boundary)
	fmt.Fprintf(&body, "--%s\r\n", boundary)
	body.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
	body.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
	body.WriteString(plain)
	fmt.Fprintf(&body, "\r\n--%s\r\n", boundary)
	body.WriteString("Content-Type: text/html; charset=utf-8\r\n")
	body.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
	body.WriteString(message)
	fmt.Fprintf(&body, "\r\n--%s--\r\n", boundary)

	return body.Bytes()
}

func encodeEnvelope(message Message, includeRaw bool) ([]byte, error) {
	payload := message.Envelope
	if !includeRaw {
		payload = envelope.RedactRaw(payload)
	}
	return json.Marshal(payload)
}

// expectOK sends the request and turns a non-2xx answer into an error carrying
// the response body, which is the only thing that makes a misconfigured webhook
// debuggable from the activity log.
func (s *Sender) expectOK(request *http.Request, label string) error {
	response, err := s.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode >= 200 && response.StatusCode < 300 {
		io.Copy(io.Discard, io.LimitReader(response.Body, 1<<16))
		return nil
	}

	body, _ := io.ReadAll(io.LimitReader(response.Body, 1<<14))
	if len(body) > 0 {
		return fmt.Errorf("%s -> HTTP %d %s", label, response.StatusCode, strings.TrimSpace(string(body)))
	}
	return fmt.Errorf("%s -> HTTP %d", label, response.StatusCode)
}

// sanitizeAddress and sanitizeHeader strip the characters that would let a
// configured address inject extra SMTP headers or commands.
func sanitizeAddress(address string) string {
	return strings.NewReplacer("\r", "", "\n", "", "<", "", ">", "").Replace(address)
}

func sanitizeHeader(value string) string {
	return strings.NewReplacer("\r", "", "\n", "").Replace(value)
}

func deref(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
