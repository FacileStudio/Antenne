package delivery

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/FacileStudio/Antenne/apps/api/modules/health"
	"github.com/FacileStudio/Antenne/apps/api/modules/settings"
)

// Probe asks a target whether its credential still works, without delivering
// anything. Every call here is a read, and every classification is chosen so the
// scheduler can tell "fix your token" from "the far side is having a bad day" —
// health.Kind.Terminal() decides whether probing continues at all, and probing a
// confirmed-dead credential is what gets an IP banned or an account locked.
//
// The health package imports nothing but time, so returning health.Observation
// from delivery creates no cycle.
func (s *Sender) Probe(ctx context.Context, target settings.DeliveryTarget) health.Observation {
	if !CanSend(target) {
		// Nothing to measure, and nothing measurable: an unconfigured or disabled
		// target is honestly unknown rather than healthy or broken.
		return probeFail(health.KindUnsupported,
			fmt.Sprintf("target `%s` is not fully configured", target.Name))
	}

	switch target.Type {
	case settings.TargetMatrix:
		return s.probeMatrix(ctx, target)
	case settings.TargetDiscord:
		return s.probeDiscord(ctx, target)
	case settings.TargetSMTP:
		return s.probeSMTP(target)
	case settings.TargetNtfy:
		return s.probeNtfy(ctx, target)
	case settings.TargetPerception:
		return s.probePerception(ctx, target)
	}

	// A generic webhook has no safe read-only probe. GET or HEAD on an arbitrary
	// third-party URL may be unimplemented, may mean something entirely different
	// from the POST we deliver with, or may be answered by a proxy, CDN or WAF
	// that never reaches the handler — so a failure proves nothing and a success
	// is worse, because it paints a dead endpoint green.
	return probeFail(health.KindUnsupported, "a generic HTTP webhook has no safe read-only probe")
}

func probeOK(detail string) health.Observation {
	return health.Observation{OK: true, Kind: health.KindNone, Source: health.SourceProbe, Detail: detail}
}

func probeFail(kind health.Kind, detail string) health.Observation {
	return health.Observation{Kind: kind, Source: health.SourceProbe, Detail: strings.TrimSpace(detail)}
}

func probeRateLimited(detail string, retryAfter time.Duration) health.Observation {
	observation := probeFail(health.KindRateLimited, detail)
	observation.RetryAfter = retryAfter
	return observation
}

// probeResponse is what a read-only probe got back. The three parts travel
// together because every classifier needs all three: the status decides the
// Kind, the body carries the far side's own error words, and the header carries
// the Retry-After that makes a rate limit binding rather than guessed.
type probeResponse struct {
	status int
	body   []byte
	header http.Header
}

// probeGET issues the probe request, capped at 16KiB of body.
//
// A body that stops short is not an error here: the status code has already
// been read, and it is the status — not the prose after it — that decides the
// Kind. Whatever arrived is returned, and a classifier that finds nothing
// useful in it falls back on the status alone.
func (s *Sender) probeGET(ctx context.Context, endpoint string, header http.Header) (probeResponse, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return probeResponse{}, err
	}
	for name, values := range header {
		for _, value := range values {
			request.Header.Add(name, value)
		}
	}
	request.Header.Set("User-Agent", "Antenne/1.0 uptime-bot")

	response, err := s.client.Do(request)
	if err != nil {
		return probeResponse{}, err
	}
	defer response.Body.Close()

	body, readErr := io.ReadAll(io.LimitReader(response.Body, 1<<14))
	if readErr != nil {
		s.log.Debug("probe body truncated", "endpoint", endpoint, "error", readErr)
	}
	return probeResponse{status: response.StatusCode, body: body, header: response.Header}, nil
}

func (r probeResponse) ok() bool { return r.status >= 200 && r.status < 300 }

// retryAfter reads the binding wait the far side asked for, in either of the
// two forms RFC 9110 allows.
func (r probeResponse) retryAfter() time.Duration {
	return retryAfterFrom(r.header)
}

// decodeHint reads the shape a far side's response is expected to have, and
// reports whether it was that shape at all.
//
// What it decodes is a hint and never a verdict. These bodies are classified by
// status code first, and a provider that answers a 429 with its CDN's HTML
// error page must leave the classification untouched rather than turn a rate
// limit into a decoding failure — so a caller that ignores the result is
// reading the zero value as "it told us nothing beyond the status".
func decodeHint(body []byte, into any) bool {
	return json.Unmarshal(body, into) == nil
}

// httpDetail keeps the far side's own words, which is what an operator pastes
// into a search or forwards to their provider.
func httpDetail(label string, response probeResponse) string {
	trimmed := strings.TrimSpace(string(response.body))
	if trimmed == "" {
		return fmt.Sprintf("%s -> HTTP %d", label, response.status)
	}
	return fmt.Sprintf("%s -> HTTP %d %s", label, response.status, trimmed)
}

func retryAfterFrom(header http.Header) time.Duration {
	value := strings.TrimSpace(header.Get("Retry-After"))
	if value == "" {
		return 0
	}
	if seconds, err := strconv.ParseFloat(value, 64); err == nil {
		if seconds <= 0 {
			return 0
		}
		return time.Duration(seconds * float64(time.Second))
	}
	if when, err := http.ParseTime(value); err == nil {
		if wait := time.Until(when); wait > 0 {
			return wait
		}
	}
	return 0
}

// siblingURL replaces the last path segment, so an ingest URL yields the health
// URL beside it whatever prefix the instance is served under.
func siblingURL(raw, name string) (string, error) {
	endpoint, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	endpoint.RawQuery = ""
	endpoint.Fragment = ""
	path := strings.TrimSuffix(endpoint.Path, "/")
	if index := strings.LastIndex(path, "/"); index >= 0 {
		endpoint.Path = path[:index+1] + name
	} else {
		endpoint.Path = "/" + name
	}
	return endpoint.String(), nil
}

func valueOr(value *int, fallback int) int {
	if value == nil {
		return fallback
	}
	return *value
}
