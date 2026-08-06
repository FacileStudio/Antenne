package providers

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/FacileStudio/Nook/apps/api/internal/envelope"
	"github.com/FacileStudio/Nook/apps/api/modules/settings"
)

// seenGUIDLimit caps the per-feed memory of already-reported items. Well past
// any real feed's page size, and reset on every restart, which is safe because
// the first poll after a restart re-seeds rather than re-alerts.
const seenGUIDLimit = 5000

type rssFeed struct {
	Title   string
	Link    string
	Entries []rssEntry
}

type rssEntry struct {
	GUID        string
	Title       string
	Link        string
	Description string
	Content     string
	Published   string
	Author      string
}

type rssState struct {
	seen        map[string]bool
	order       []string
	initialized bool
}

// RSS polls feed providers and reports items it has not seen.
type RSS struct {
	log    *slog.Logger
	emit   Emit
	client *http.Client

	mu     sync.Mutex
	states map[string]*rssState
	cancel context.CancelFunc
	group  sync.WaitGroup
}

// NewRSS builds the feed poller.
func NewRSS(log *slog.Logger, emit Emit) *RSS {
	return &RSS{
		log:    log,
		emit:   emit,
		client: &http.Client{Timeout: 30 * time.Second},
		states: map[string]*rssState{},
	}
}

// Apply restarts the pollers to match the current settings.
func (r *RSS) Apply(ctx context.Context, current settings.AppSettings) {
	r.Stop()

	loopCtx, cancel := context.WithCancel(ctx)
	r.mu.Lock()
	r.cancel = cancel
	r.states = map[string]*rssState{}
	r.mu.Unlock()

	for _, provider := range current.Providers.Items {
		if provider.Type != settings.ProviderRSS || !provider.Enabled || provider.FeedURL == "" {
			continue
		}

		r.mu.Lock()
		r.states[provider.ID] = &rssState{seen: map[string]bool{}}
		r.mu.Unlock()

		r.log.Info("polling feed",
			slog.String("provider", provider.Name),
			slog.String("feed_url", provider.FeedURL),
			slog.Int("interval_seconds", provider.PollInterval))

		r.group.Add(1)
		go r.watch(loopCtx, provider)
	}
}

// Stop halts every poller.
func (r *RSS) Stop() {
	r.mu.Lock()
	cancel := r.cancel
	r.cancel = nil
	r.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	r.group.Wait()
}

func (r *RSS) watch(ctx context.Context, provider settings.Provider) {
	defer r.group.Done()

	interval := time.Duration(provider.PollInterval) * time.Second
	if interval <= 0 {
		interval = 5 * time.Minute
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	r.poll(ctx, provider)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.poll(ctx, provider)
		}
	}
}

func (r *RSS) poll(ctx context.Context, provider settings.Provider) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, provider.FeedURL, nil)
	if err != nil {
		r.log.Error("feed poll failed", slog.String("provider", provider.Name), slog.Any("error", err))
		return
	}
	request.Header.Set("User-Agent", "Nook/1.0 rss-bot")

	response, err := r.client.Do(request)
	if err != nil {
		r.log.Error("feed poll failed", slog.String("provider", provider.Name), slog.Any("error", err))
		return
	}
	defer response.Body.Close()

	if response.StatusCode >= 400 {
		r.log.Error("feed poll failed",
			slog.String("provider", provider.Name),
			slog.Int("status", response.StatusCode))
		return
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, 8<<20))
	if err != nil {
		r.log.Error("feed poll failed", slog.String("provider", provider.Name), slog.Any("error", err))
		return
	}

	feed, err := parseFeed(body)
	if err != nil {
		r.log.Error("feed could not be parsed", slog.String("provider", provider.Name), slog.Any("error", err))
		return
	}

	r.mu.Lock()
	state, ok := r.states[provider.ID]
	if !ok {
		r.mu.Unlock()
		return
	}

	// The first poll only records what is already published. Without it every
	// restart would replay the whole feed into Matrix.
	if !state.initialized {
		for _, entry := range feed.Entries {
			state.remember(entry.GUID)
		}
		state.initialized = true
		r.mu.Unlock()
		r.log.Info("feed initialized",
			slog.String("provider", provider.Name),
			slog.Int("existing_items", len(feed.Entries)))
		return
	}

	fresh := make([]rssEntry, 0)
	for _, entry := range feed.Entries {
		if state.seen[entry.GUID] {
			continue
		}
		state.remember(entry.GUID)
		fresh = append(fresh, entry)
	}
	r.mu.Unlock()

	for _, entry := range fresh {
		r.emit(buildRSSAlert(provider, feed, entry))
	}
}

func (s *rssState) remember(guid string) {
	if s.seen[guid] {
		return
	}
	s.seen[guid] = true
	s.order = append(s.order, guid)
	if len(s.order) > seenGUIDLimit {
		drop := s.order[:len(s.order)-seenGUIDLimit/2]
		for _, old := range drop {
			delete(s.seen, old)
		}
		s.order = append([]string{}, s.order[len(drop):]...)
	}
}

func buildRSSAlert(provider settings.Provider, feed rssFeed, entry rssEntry) Alert {
	title := entry.Title
	if title == "" {
		title = "untitled"
	}

	lines := []string{fmt.Sprintf("📡 <b>[RSS] %s</b> new item", envelope.EscapeHTML(provider.Name))}
	if entry.Title != "" {
		if entry.Link != "" {
			lines = append(lines, fmt.Sprintf(`<b><a href="%s">%s</a></b>`,
				envelope.EscapeHTML(entry.Link), envelope.EscapeHTML(entry.Title)))
		} else {
			lines = append(lines, "<b>"+envelope.EscapeHTML(entry.Title)+"</b>")
		}
	}

	body := entry.Description
	limit := 300
	if provider.IncludeContent {
		if entry.Content != "" {
			body = entry.Content
		}
		limit = 2000
	}
	if body != "" {
		lines = append(lines, envelope.EscapeHTML(truncate(envelope.HTMLToPlainText(body), limit)))
	}
	if entry.Published != "" {
		lines = append(lines, "<code>"+envelope.EscapeHTML(entry.Published)+"</code>")
	}
	if feed.Title != "" {
		lines = append(lines, "Feed: "+envelope.EscapeHTML(feed.Title))
	}

	alert := Alert{
		MessageHTML:  strings.Join(lines, "<br/>"),
		ProviderID:   provider.ID,
		ProviderName: provider.Name,
		ProviderType: string(settings.ProviderRSS),
		Event:        "new item: " + title,
		EventType:    "rss.item.new",
		EventTitle:   firstNonEmpty(entry.Title, "New feed item"),
		EventURL:     entry.Link,
		OccurredAt:   entry.Published,
		Subject: &envelope.Subject{
			Type: envelope.SubjectUnknown,
			ID:   envelope.Str(entry.GUID),
			Name: envelope.Str(firstNonEmpty(entry.Title, "Feed item")),
			URL:  envelope.Str(entry.Link),
		},
		Normalized: map[string]any{
			"guid":        entry.GUID,
			"title":       entry.Title,
			"link":        entry.Link,
			"description": truncate(envelope.HTMLToPlainText(entry.Description), 500),
			"pubDate":     entry.Published,
			"author":      entry.Author,
			"feedTitle":   feed.Title,
			"feedUrl":     provider.FeedURL,
		},
	}
	if provider.IncludeContent {
		alert.Normalized["content"] = firstNonEmpty(entry.Content, entry.Description)
	}
	if entry.Author != "" {
		alert.Actor = &envelope.Actor{Name: envelope.Str(entry.Author)}
	}

	return alert
}

// rssDocument covers both RSS 2.0 and Atom in one shape, because a feed URL
// does not say which it serves and both put their items at a predictable depth.
type rssDocument struct {
	XMLName xml.Name `xml:"-"`
	Channel struct {
		Title string `xml:"title"`
		Link  string `xml:"link"`
		Items []struct {
			GUID        string `xml:"guid"`
			Title       string `xml:"title"`
			Link        string `xml:"link"`
			Description string `xml:"description"`
			Encoded     string `xml:"encoded"`
			PubDate     string `xml:"pubDate"`
			Author      string `xml:"author"`
			Creator     string `xml:"creator"`
		} `xml:"item"`
	} `xml:"channel"`

	Title   string `xml:"title"`
	Entries []struct {
		ID        string `xml:"id"`
		Title     string `xml:"title"`
		Summary   string `xml:"summary"`
		Content   string `xml:"content"`
		Updated   string `xml:"updated"`
		Published string `xml:"published"`
		Links     []struct {
			Href string `xml:"href,attr"`
			Rel  string `xml:"rel,attr"`
		} `xml:"link"`
		Author struct {
			Name string `xml:"name"`
		} `xml:"author"`
	} `xml:"entry"`
	Links []struct {
		Href string `xml:"href,attr"`
		Rel  string `xml:"rel,attr"`
	} `xml:"link"`
}

func parseFeed(body []byte) (rssFeed, error) {
	var document rssDocument
	decoder := xml.NewDecoder(strings.NewReader(string(body)))
	// Feeds in the wild declare every charset under the sun; without this the
	// decoder refuses anything that is not UTF-8 outright.
	decoder.CharsetReader = func(_ string, input io.Reader) (io.Reader, error) { return input, nil }
	decoder.Strict = false

	if err := decoder.Decode(&document); err != nil {
		return rssFeed{}, err
	}

	if len(document.Entries) > 0 {
		feed := rssFeed{Title: document.Title}
		for _, link := range document.Links {
			if link.Rel == "alternate" || link.Rel == "" {
				feed.Link = link.Href
				break
			}
		}
		for index, entry := range document.Entries {
			link := ""
			for _, candidate := range entry.Links {
				if candidate.Rel == "alternate" || candidate.Rel == "" {
					link = candidate.Href
					break
				}
			}
			guid := firstNonEmpty(entry.ID, link, fmt.Sprintf("atom-%d", index))
			feed.Entries = append(feed.Entries, rssEntry{
				GUID:        guid,
				Title:       strings.TrimSpace(entry.Title),
				Link:        link,
				Description: strings.TrimSpace(entry.Summary),
				Content:     strings.TrimSpace(entry.Content),
				Published:   firstNonEmpty(entry.Published, entry.Updated),
				Author:      strings.TrimSpace(entry.Author.Name),
			})
		}
		return feed, nil
	}

	feed := rssFeed{Title: document.Channel.Title, Link: document.Channel.Link}
	for index, item := range document.Channel.Items {
		guid := firstNonEmpty(item.GUID, item.Link, fmt.Sprintf("rss-%d", index))
		feed.Entries = append(feed.Entries, rssEntry{
			GUID:        guid,
			Title:       strings.TrimSpace(item.Title),
			Link:        strings.TrimSpace(item.Link),
			Description: strings.TrimSpace(item.Description),
			Content:     strings.TrimSpace(item.Encoded),
			Published:   strings.TrimSpace(item.PubDate),
			Author:      firstNonEmpty(strings.TrimSpace(item.Author), strings.TrimSpace(item.Creator)),
		})
	}
	return feed, nil
}

func truncate(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit]) + "…"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
