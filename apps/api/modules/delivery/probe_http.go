package delivery

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/FacileStudio/Antenne/apps/api/modules/health"
	"github.com/FacileStudio/Antenne/apps/api/modules/settings"
)

// probeDiscord uses Get Webhook with Token: the webhook URL answers GET with the
// webhook object and needs no bot credential.
//
// This is the channel where misclassification is dangerous. Around 10k invalid
// requests (401/403/429) in 10 minutes ban the instance's IP for about an hour,
// so a deleted webhook has to be KindDead — terminal, probing suspended — and
// never something we retry on a schedule.
func (s *Sender) probeDiscord(ctx context.Context, target settings.DeliveryTarget) health.Observation {
	endpoint, err := url.Parse(target.WebhookURL)
	if err != nil {
		return probeFail(health.KindUnreachable, SafeErrorText(err))
	}
	// ?wait= and ?thread_id= belong to the POST, not to the lookup.
	endpoint.RawQuery = ""
	endpoint.Fragment = ""

	response, err := s.probeGET(ctx, endpoint.String(), nil)
	if err != nil {
		return probeFail(health.KindUnreachable, SafeErrorText(err))
	}
	return classifyDiscord(response)
}

func classifyDiscord(response probeResponse) health.Observation {
	detail := httpDetail("Discord webhook", response)

	var decoded struct {
		Code       int     `json:"code"`
		Name       string  `json:"name"`
		RetryAfter float64 `json:"retry_after"`
	}
	decodeHint(response.body, &decoded)

	switch {
	case response.ok():
		return probeOK(decoded.Name)
	case response.status == http.StatusTooManyRequests:
		wait := response.retryAfter()
		if wait == 0 && decoded.RetryAfter > 0 {
			wait = time.Duration(decoded.RetryAfter * float64(time.Second))
		}
		return probeRateLimited(detail, wait)
	case decoded.Code == 10015 || decoded.Code == 50027:
		return probeFail(health.KindDead, detail)
	case response.status == http.StatusNotFound || response.status == http.StatusUnauthorized:
		return probeFail(health.KindDead, detail)
	case response.status == http.StatusForbidden:
		return probeFail(health.KindDenied, detail)
	}
	return probeFail(health.KindUnreachable, detail)
}

// probeNtfy checks the server, then the token if there is one.
//
// The token check has a trap: /v1/account is allowed anonymously, so a token
// that never reached the server (a proxy stripping Authorization, most often)
// comes back 200 as the anonymous user. Reporting that as healthy is a false
// green, so the username is asserted rather than the status code.
//
// The per-topic /{topic}/auth endpoint is deliberately not used: it checks READ
// access, and a correctly configured write-only publish token fails it.
func (s *Sender) probeNtfy(ctx context.Context, target settings.DeliveryTarget) health.Observation {
	base := target.ServerURL
	if !strings.HasSuffix(base, "/") {
		base += "/"
	}
	server, err := url.Parse(base)
	if err != nil {
		return probeFail(health.KindUnreachable, SafeErrorText(err))
	}

	if observation, healthy := s.probeNtfyServer(ctx, server); !healthy {
		return observation
	}

	if target.AccessToken == "" {
		// An anonymous publish target is as verified as it can be: topics are
		// created on publish, so "the topic exists" is not a question.
		return probeOK("server healthy")
	}
	return s.probeNtfyToken(ctx, server, target.AccessToken)
}

// probeNtfyServer reports whether the server itself answered healthy, and what
// to report when it did not. The observation is only meaningful when the second
// result is false.
func (s *Sender) probeNtfyServer(ctx context.Context, server *url.URL) (health.Observation, bool) {
	response, err := s.probeGET(ctx, server.JoinPath("v1", "health").String(), nil)
	if err != nil {
		return probeFail(health.KindUnreachable, SafeErrorText(err)), false
	}
	detail := httpDetail("ntfy", response)
	if response.status == http.StatusTooManyRequests {
		return probeRateLimited(detail, response.retryAfter()), false
	}
	if !response.ok() {
		return probeFail(health.KindUnreachable, detail), false
	}

	var reported struct {
		Healthy bool `json:"healthy"`
	}
	if !decodeHint(response.body, &reported) || !reported.Healthy {
		return probeFail(health.KindUnreachable, detail), false
	}
	return health.Observation{}, true
}

func (s *Sender) probeNtfyToken(ctx context.Context, server *url.URL, token string) health.Observation {
	response, err := s.probeGET(ctx, server.JoinPath("v1", "account").String(),
		http.Header{"Authorization": []string{"Bearer " + token}})
	if err != nil {
		return probeFail(health.KindUnreachable, SafeErrorText(err))
	}
	detail := httpDetail("ntfy", response)

	switch {
	case response.status == http.StatusTooManyRequests:
		return probeRateLimited(detail, response.retryAfter())
	case response.status == http.StatusUnauthorized:
		return probeFail(health.KindDead, detail)
	case response.status == http.StatusForbidden:
		return probeFail(health.KindDenied, detail)
	case !response.ok():
		return probeFail(health.KindUnreachable, detail)
	}

	var account struct {
		Username string `json:"username"`
	}
	decodeHint(response.body, &account)
	// "*" is ntfy's anonymous user.
	if account.Username == "" || account.Username == "*" {
		return probeFail(health.KindDenied,
			"ntfy answered as the anonymous user, so the access token never reached it")
	}

	return probeOK("authenticated as " + account.Username)
}

// probePerception asks Perception's own health endpoint, resolved as a sibling
// of the configured ingest URL so an instance served under a path prefix still
// works. It proves the service is up; it cannot prove the shared secret is
// still accepted, because ingest is the only route that checks it and ingest
// writes.
func (s *Sender) probePerception(ctx context.Context, target settings.DeliveryTarget) health.Observation {
	endpoint, err := siblingURL(target.URL, "health")
	if err != nil {
		return probeFail(health.KindUnreachable, SafeErrorText(err))
	}

	response, err := s.probeGET(ctx, endpoint, nil)
	if err != nil {
		return probeFail(health.KindUnreachable, SafeErrorText(err))
	}
	detail := httpDetail("Perception", response)

	switch {
	case response.ok():
		return probeOK("server healthy")
	case response.status == http.StatusTooManyRequests:
		return probeRateLimited(detail, response.retryAfter())
	case response.status == http.StatusUnauthorized || response.status == http.StatusForbidden:
		return probeFail(health.KindDenied, detail)
	}
	return probeFail(health.KindUnreachable, detail)
}
