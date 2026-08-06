package providers

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"

	"github.com/FacileStudio/Nook/apps/api/internal/envelope"
	"github.com/FacileStudio/Nook/apps/api/modules/settings"
)

const imapMailbox = "INBOX"

type imapState struct {
	lastSeenUID uint32
	initialized bool
}

// IMAP polls inbox providers and reports new messages.
type IMAP struct {
	log  *slog.Logger
	emit Emit

	mu     sync.Mutex
	states map[string]*imapState
	cancel context.CancelFunc
	group  sync.WaitGroup
}

// NewIMAP builds the inbox poller.
func NewIMAP(log *slog.Logger, emit Emit) *IMAP {
	return &IMAP{log: log, emit: emit, states: map[string]*imapState{}}
}

// Apply restarts the pollers to match the current settings.
func (i *IMAP) Apply(ctx context.Context, current settings.AppSettings) {
	i.Stop()

	loopCtx, cancel := context.WithCancel(ctx)
	i.mu.Lock()
	i.cancel = cancel
	i.states = map[string]*imapState{}
	i.mu.Unlock()

	for _, provider := range current.Providers.Items {
		if provider.Type != settings.ProviderIMAP || !provider.Enabled {
			continue
		}
		if provider.Host == "" || provider.Username == "" || provider.Password == nil || *provider.Password == "" {
			continue
		}

		i.mu.Lock()
		i.states[provider.ID] = &imapState{}
		i.mu.Unlock()

		i.log.Info("polling inbox",
			slog.String("provider", provider.Name),
			slog.String("host", provider.Host),
			slog.Int("port", provider.Port),
			slog.Int("interval_seconds", provider.PollInterval))

		i.group.Add(1)
		go i.watch(loopCtx, provider)
	}
}

// Stop halts every poller.
func (i *IMAP) Stop() {
	i.mu.Lock()
	cancel := i.cancel
	i.cancel = nil
	i.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	i.group.Wait()
}

func (i *IMAP) watch(ctx context.Context, provider settings.Provider) {
	defer i.group.Done()

	interval := time.Duration(provider.PollInterval) * time.Second
	if interval <= 0 {
		interval = time.Minute
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	i.poll(ctx, provider)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			i.poll(ctx, provider)
		}
	}
}

func (i *IMAP) poll(ctx context.Context, provider settings.Provider) {
	client, err := dialIMAP(provider)
	if err != nil {
		i.log.Error("inbox poll failed", slog.String("provider", provider.Name), slog.Any("error", err))
		return
	}
	defer client.Close()

	if err := client.Login(provider.Username, *provider.Password).Wait(); err != nil {
		i.log.Error("inbox login failed", slog.String("provider", provider.Name), slog.Any("error", err))
		return
	}
	defer client.Logout().Wait()

	mailbox, err := client.Select(imapMailbox, nil).Wait()
	if err != nil {
		i.log.Error("inbox select failed", slog.String("provider", provider.Name), slog.Any("error", err))
		return
	}
	if mailbox.NumMessages == 0 {
		return
	}

	i.mu.Lock()
	state, ok := i.states[provider.ID]
	if !ok {
		i.mu.Unlock()
		return
	}
	initialized := state.initialized
	lastSeen := state.lastSeenUID
	i.mu.Unlock()

	// The first poll only records where the inbox is. Without it a restart
	// would re-alert on every message the mailbox still holds.
	if !initialized {
		highest, err := highestUID(client)
		if err != nil {
			i.log.Error("inbox scan failed", slog.String("provider", provider.Name), slog.Any("error", err))
			return
		}
		i.mu.Lock()
		state.lastSeenUID = highest
		state.initialized = true
		i.mu.Unlock()
		i.log.Info("inbox initialized",
			slog.String("provider", provider.Name),
			slog.Uint64("last_uid", uint64(highest)))
		return
	}

	messages, err := fetchSince(client, lastSeen, provider.IncludeEmailBody)
	if err != nil {
		i.log.Error("inbox fetch failed", slog.String("provider", provider.Name), slog.Any("error", err))
		return
	}

	for _, message := range messages {
		if message.uid <= lastSeen {
			continue
		}
		lastSeen = message.uid
		i.emit(buildIMAPAlert(provider, message))
	}

	i.mu.Lock()
	if lastSeen > state.lastSeenUID {
		state.lastSeenUID = lastSeen
	}
	i.mu.Unlock()
}

func dialIMAP(provider settings.Provider) (*imapclient.Client, error) {
	address := net.JoinHostPort(provider.Host, strconv.Itoa(provider.Port))
	options := &imapclient.Options{TLSConfig: &tls.Config{ServerName: provider.Host}}

	if provider.Secure {
		return imapclient.DialTLS(address, options)
	}

	// STARTTLS rather than a plain connection: a "not secure" port setting
	// means implicit TLS is off, not that credentials should cross the wire in
	// clear. A server that does not advertise it falls back to plaintext.
	client, err := imapclient.DialStartTLS(address, options)
	if err == nil {
		return client, nil
	}
	return imapclient.DialInsecure(address, options)
}

func highestUID(client *imapclient.Client) (uint32, error) {
	criteria := &imap.SearchCriteria{}
	data, err := client.UIDSearch(criteria, nil).Wait()
	if err != nil {
		return 0, err
	}

	var highest uint32
	for _, uid := range data.AllUIDs() {
		if uint32(uid) > highest {
			highest = uint32(uid)
		}
	}
	return highest, nil
}

type imapMessage struct {
	uid       uint32
	subject   string
	from      string
	to        string
	date      string
	messageID string
	body      string
}

func fetchSince(client *imapclient.Client, lastSeen uint32, includeBody bool) ([]imapMessage, error) {
	uidSet := imap.UIDSetNum()
	uidSet = append(uidSet, imap.UIDRange{Start: imap.UID(lastSeen + 1), Stop: 0})

	options := &imap.FetchOptions{
		UID:      true,
		Envelope: true,
	}
	if includeBody {
		options.BodySection = []*imap.FetchItemBodySection{{Specifier: imap.PartSpecifierText, Peek: true}}
	}

	buffers, err := client.Fetch(uidSet, options).Collect()
	if err != nil {
		return nil, err
	}

	messages := make([]imapMessage, 0, len(buffers))
	for _, buffer := range buffers {
		message := imapMessage{uid: uint32(buffer.UID)}
		if buffer.Envelope != nil {
			message.subject = buffer.Envelope.Subject
			message.from = formatAddresses(buffer.Envelope.From)
			message.to = formatAddresses(buffer.Envelope.To)
			message.messageID = buffer.Envelope.MessageID
			if !buffer.Envelope.Date.IsZero() {
				message.date = buffer.Envelope.Date.Format(time.RFC3339)
			}
		}
		for _, section := range buffer.BodySection {
			message.body = strings.TrimSpace(string(section.Bytes))
			break
		}
		messages = append(messages, message)
	}

	return messages, nil
}

func formatAddresses(addresses []imap.Address) string {
	parts := make([]string, 0, len(addresses))
	for _, address := range addresses {
		mailbox := address.Addr()
		if address.Name != "" {
			parts = append(parts, fmt.Sprintf("%s <%s>", address.Name, mailbox))
			continue
		}
		parts = append(parts, mailbox)
	}
	return strings.Join(parts, ", ")
}

func buildIMAPAlert(provider settings.Provider, message imapMessage) Alert {
	subject := message.subject
	if subject == "" {
		subject = "no subject"
	}

	lines := []string{fmt.Sprintf("📨 <b>[IMAP] %s</b> new email", envelope.EscapeHTML(provider.Name))}
	if message.from != "" {
		lines = append(lines, "From: "+envelope.EscapeHTML(message.from))
	}
	if message.subject != "" {
		lines = append(lines, "Subject: <b>"+envelope.EscapeHTML(message.subject)+"</b>")
	}
	if message.date != "" {
		lines = append(lines, "Date: <code>"+envelope.EscapeHTML(message.date)+"</code>")
	}
	if message.body != "" {
		lines = append(lines, "<pre>"+envelope.EscapeHTML(truncate(message.body, 4000))+"</pre>")
	}

	snippet := ""
	if message.body != "" {
		snippet = truncate(message.body, 200)
	}

	return Alert{
		MessageHTML:  strings.Join(lines, "<br/>"),
		ProviderID:   provider.ID,
		ProviderName: provider.Name,
		ProviderType: string(settings.ProviderIMAP),
		Event:        "new email: " + subject,
		EventType:    "imap.email.received",
		EventTitle:   firstNonEmpty(message.subject, "New email"),
		OccurredAt:   message.date,
		Actor:        &envelope.Actor{Email: envelope.Str(message.from)},
		Subject: &envelope.Subject{
			Type: envelope.SubjectEmail,
			ID:   envelope.Str(strconv.FormatUint(uint64(message.uid), 10)),
			Name: envelope.Str(firstNonEmpty(message.subject, "No subject")),
		},
		Normalized: map[string]any{
			"uid":       message.uid,
			"subject":   message.subject,
			"from":      message.from,
			"to":        message.to,
			"date":      message.date,
			"body":      message.body,
			"snippet":   snippet,
			"messageId": message.messageID,
		},
		// An inbox is personal data by construction, whatever the message says.
		ContainsPII: true,
	}
}

// TestConnection verifies an IMAP provider's credentials, for the dashboard's
// "Test connection" button.
func TestConnection(provider settings.Provider) error {
	if provider.Host == "" || provider.Username == "" || provider.Password == nil || *provider.Password == "" {
		return fmt.Errorf("IMAP host, username, and password are required")
	}

	client, err := dialIMAP(provider)
	if err != nil {
		return err
	}
	defer client.Close()

	if err := client.Login(provider.Username, *provider.Password).Wait(); err != nil {
		return err
	}
	defer client.Logout().Wait()

	_, err = client.Select(imapMailbox, nil).Wait()
	return err
}
