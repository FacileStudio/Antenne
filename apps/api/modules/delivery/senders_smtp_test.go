package delivery

import (
	"bufio"
	"fmt"
	"github.com/FacileStudio/Antenne/apps/api/modules/settings"
	"log/slog"
	"net"
	"strings"
	"testing"
)

/*
fakeSMTP speaks just enough of RFC 5321 for a submission attempt to complete,
over plaintext. It deliberately does *not* advertise STARTTLS: the ordering bug
this file exists to pin fires either way, because net/smtp's Extension() greets
the server as a side effect, and that is what a later Hello() refuses to follow.

It records the verbs it saw so a test can assert the conversation, not just the
absence of an error.
*/
func fakeSMTP(t *testing.T, advertiseSTARTTLS bool) (addr string, verbs *[]string) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	seen := make([]string, 0, 8)

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		reader := bufio.NewReader(conn)
		write := func(format string, args ...any) {
			fmt.Fprintf(conn, format+"\r\n", args...)
		}

		write("220 fake ESMTP")
		inData := false
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			line = strings.TrimRight(line, "\r\n")

			if inData {
				if line == "." {
					inData = false
					write("250 2.0.0 queued")
				}
				continue
			}

			verb := strings.ToUpper(strings.Fields(line + " ")[0])
			seen = append(seen, verb)

			switch verb {
			case "EHLO":
				if advertiseSTARTTLS {
					write("250-fake")
					write("250 STARTTLS")
				} else {
					write("250 fake")
				}
			case "MAIL", "RCPT":
				write("250 2.1.0 ok")
			case "DATA":
				inData = true
				write("354 send it")
			case "QUIT":
				write("221 2.0.0 bye")
				return
			default:
				write("250 2.0.0 ok")
			}
		}
	}()

	return listener.Addr().String(), &seen
}

/*
Submission over STARTTLS — the whole of port 587 — was broken from the start, and
nothing caught it because implicit TLS on 465 takes a different path: NewClient
does not greet, so Hello() there is the first command and succeeds.

The failure was not a bad password or an unreachable relay. net/smtp's Extension
and StartTLS both call the package-private hello() for you, and Hello() then
returns "smtp: Hello called after other methods" — so sendSMTP aborted before it
ever presented a credential. Any test that only asserted "sending fails against a
misconfigured target" would have passed happily.
*/
func TestSendSMTPGreetsBeforeAskingAboutExtensions(t *testing.T) {
	for _, advertise := range []bool{false, true} {
		name := "server without STARTTLS"
		if advertise {
			name = "server advertising STARTTLS"
		}
		t.Run(name, func(t *testing.T) {
			addr, verbs := fakeSMTP(t, advertise)
			host, port := splitHostPort(t, addr)

			target := smtpTarget(host, port)
			// A server that advertises STARTTLS would have us upgrade, and this
			// fake cannot complete a handshake — so only the plaintext case runs
			// to completion. The greeting order is what both cases prove.
			sender := NewSender(slog.New(slog.DiscardHandler))
			err := sender.sendSMTP(target, "<p>hello</p>")

			if err != nil && strings.Contains(err.Error(), "Hello called after other methods") {
				t.Fatalf("sendSMTP greeted the server too late: %v", err)
			}
			if !advertise && err != nil {
				t.Fatalf("sendSMTP over plaintext: %v", err)
			}

			if len(*verbs) == 0 || (*verbs)[0] != "EHLO" {
				t.Fatalf("first verb was %v, want EHLO first", *verbs)
			}
		})
	}
}

func smtpTarget(host string, port int) settings.DeliveryTarget {
	return settings.DeliveryTarget{
		ID:            "smtp-1",
		Type:          settings.TargetSMTP,
		Name:          "Relay",
		Enabled:       true,
		Host:          host,
		Port:          port,
		FromAddress:   "antenne@example.test",
		ToAddresses:   []string{"ops@example.test"},
		Secure:        false,
		SubjectPrefix: "",
	}
}

func splitHostPort(t *testing.T, addr string) (string, int) {
	t.Helper()
	host, portText, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("split %q: %v", addr, err)
	}
	var port int
	if _, err := fmt.Sscanf(portText, "%d", &port); err != nil {
		t.Fatalf("port %q: %v", portText, err)
	}
	return host, port
}
