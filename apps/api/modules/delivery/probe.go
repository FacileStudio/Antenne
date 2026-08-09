package delivery

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/smtp"
	"net/textproto"
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

// -- Matrix ------------------------------------------------------------------

type matrixWhoamiResponse struct {
	UserID   string `json:"user_id"`
	DeviceID string `json:"device_id"`
	IsGuest  bool   `json:"is_guest"`
}

type matrixJoinedRoomsResponse struct {
	JoinedRooms []string `json:"joined_rooms"`
}

// probeMatrix checks the token, then the destinations, because a valid token
// says nothing about deliverability: the two failures need different fixes and
// only one of them is a credential problem.
func (s *Sender) probeMatrix(ctx context.Context, target settings.DeliveryTarget) health.Observation {
	session := s.matrixSessionFor(target)
	token, err := s.ensureMatrixSession(ctx, target, session, false)
	if err != nil {
		return classifyMatrixError(err)
	}

	var whoami matrixWhoamiResponse
	err = s.matrixJSON(ctx, target.Homeserver, token, http.MethodGet,
		"/_matrix/client/v3/account/whoami", nil, &whoami)
	if isUnknownMatrixToken(err) && hasPasswordAuth(target) {
		// The cached token is gone, which is not the same as the credential
		// being gone. Revoking a session invalidates its token — removing the
		// device from Element does exactly that — and so does a homeserver
		// rotating one. The password that would fix it is right here, and the
		// send path already re-logs in on this errcode, so a probe that instead
		// reports KindDead suspends itself for an hour and tells an operator to
		// rotate a credential that still works.
		//
		// Once, and only on this errcode: if the re-login is genuinely refused
		// it answers M_FORBIDDEN, which classifies as dead on its own merits.
		token, err = s.ensureMatrixSession(ctx, target, session, true)
		if err != nil {
			return classifyMatrixError(err)
		}
		err = s.matrixJSON(ctx, target.Homeserver, token, http.MethodGet,
			"/_matrix/client/v3/account/whoami", nil, &whoami)
	}
	if err != nil {
		return classifyMatrixError(err)
	}

	// A token that answers as somebody else is not the credential this target was
	// configured with. It is not KindDead — the homeserver accepted it — but it
	// cannot deliver as the configured identity until an operator intervenes.
	if target.UserID != "" && whoami.UserID != target.UserID {
		return probeFail(health.KindDenied,
			fmt.Sprintf("token belongs to %s, target is configured as %s", whoami.UserID, target.UserID))
	}
	if target.DeviceID != nil && *target.DeviceID != "" && whoami.DeviceID != "" && whoami.DeviceID != *target.DeviceID {
		return probeFail(health.KindDenied,
			fmt.Sprintf("token belongs to device %s, target is configured for device %s", whoami.DeviceID, *target.DeviceID))
	}

	var joined matrixJoinedRoomsResponse
	if err := s.matrixJSON(ctx, target.Homeserver, token, http.MethodGet,
		"/_matrix/client/v3/joined_rooms", nil, &joined); err != nil {
		return classifyMatrixError(err)
	}

	members := make(map[string]struct{}, len(joined.JoinedRooms))
	for _, roomID := range joined.JoinedRooms {
		members[roomID] = struct{}{}
	}
	var missing []string
	for _, roomID := range target.RoomIDs {
		if _, ok := members[roomID]; !ok {
			missing = append(missing, roomID)
		}
	}
	if len(missing) > 0 {
		return probeFail(health.KindDenied,
			fmt.Sprintf("not joined to %s", strings.Join(missing, ", ")))
	}

	return probeOK(fmt.Sprintf("%s in %d room(s)", whoami.UserID, len(target.RoomIDs)))
}

type matrixPowerLevels struct {
	Users         map[string]int `json:"users"`
	UsersDefault  *int           `json:"users_default"`
	Events        map[string]int `json:"events"`
	EventsDefault *int           `json:"events_default"`
}

// ProbeMatrixPowerLevels checks that the bot's power level still clears the bar
// for posting in each configured room — a bot can be joined to a room whose
// events_default was raised above it and be silently unable to speak.
//
// It is deliberately not part of Probe: it costs one request per room, where the
// rest of the Matrix probe costs two in total. Run it on a much longer cadence.
func (s *Sender) ProbeMatrixPowerLevels(ctx context.Context, target settings.DeliveryTarget) health.Observation {
	if target.Type != settings.TargetMatrix || !CanSend(target) {
		return probeFail(health.KindUnsupported, "power levels only apply to a configured Matrix target")
	}

	session := s.matrixSessionFor(target)
	token, err := s.ensureMatrixSession(ctx, target, session, false)
	if err != nil {
		return classifyMatrixError(err)
	}

	userID := target.UserID
	if userID == "" {
		var whoami matrixWhoamiResponse
		if err := s.matrixJSON(ctx, target.Homeserver, token, http.MethodGet,
			"/_matrix/client/v3/account/whoami", nil, &whoami); err != nil {
			return classifyMatrixError(err)
		}
		userID = whoami.UserID
	}

	for _, roomID := range target.RoomIDs {
		path := fmt.Sprintf("/_matrix/client/v3/rooms/%s/state/m.room.power_levels", url.PathEscape(roomID))

		var levels matrixPowerLevels
		err := s.matrixJSON(ctx, target.Homeserver, token, http.MethodGet, path, nil, &levels)
		if err != nil {
			var apiErr *matrixError
			// A room with no m.room.power_levels event is healthy, not broken: the
			// spec's default is events_default 0, which anyone in the room clears.
			if asMatrixError(err, &apiErr) && apiErr.status == http.StatusNotFound {
				continue
			}
			if apiErr != nil && apiErr.status == http.StatusForbidden {
				return probeFail(health.KindDenied,
					fmt.Sprintf("cannot read power levels in %s: %s", roomID, apiErr.body))
			}
			return classifyMatrixError(err)
		}

		required := valueOr(levels.EventsDefault, 0)
		if level, ok := levels.Events["m.room.message"]; ok {
			required = level
		}
		mine := valueOr(levels.UsersDefault, 0)
		if level, ok := levels.Users[userID]; ok {
			mine = level
		}
		if mine < required {
			return probeFail(health.KindDenied,
				fmt.Sprintf("power level %d in %s, m.room.message needs %d", mine, roomID, required))
		}
	}

	return probeOK(fmt.Sprintf("can post in %d room(s)", len(target.RoomIDs)))
}

// classifyMatrixError turns a homeserver answer into a Kind. The distinction
// that matters is soft logout: a 401 M_UNKNOWN_TOKEN carrying soft_logout means
// the device ID and the encryption keys are still valid and a refresh token can
// revive the session, so calling it dead throws away working state.
func classifyMatrixError(err error) health.Observation {
	var apiErr *matrixError
	if !asMatrixError(err, &apiErr) {
		return probeFail(health.KindUnreachable, SafeErrorText(err))
	}

	var decoded struct {
		SoftLogout   bool  `json:"soft_logout"`
		RetryAfterMS int64 `json:"retry_after_ms"`
	}
	_ = json.Unmarshal([]byte(apiErr.body), &decoded)

	switch {
	case apiErr.status == http.StatusUnauthorized:
		if decoded.SoftLogout {
			return probeFail(health.KindRefreshable, apiErr.Error())
		}
		return probeFail(health.KindDead, apiErr.Error())
	case apiErr.status == http.StatusTooManyRequests:
		return probeRateLimited(apiErr.Error(), time.Duration(decoded.RetryAfterMS)*time.Millisecond)
	case apiErr.errcode == "M_FORBIDDEN" || apiErr.errcode == "M_USER_DEACTIVATED":
		return probeFail(health.KindDead, apiErr.Error())
	case apiErr.status == http.StatusForbidden:
		return probeFail(health.KindDenied, apiErr.Error())
	}
	return probeFail(health.KindUnreachable, apiErr.Error())
}

// -- Discord -----------------------------------------------------------------

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

	status, body, header, err := s.probeGET(ctx, endpoint.String(), nil)
	if err != nil {
		return probeFail(health.KindUnreachable, SafeErrorText(err))
	}
	detail := httpDetail("Discord webhook", status, body)

	var decoded struct {
		Code       int     `json:"code"`
		Name       string  `json:"name"`
		RetryAfter float64 `json:"retry_after"`
	}
	_ = json.Unmarshal(body, &decoded)

	switch {
	case status >= 200 && status < 300:
		return probeOK(decoded.Name)
	case status == http.StatusTooManyRequests:
		wait := retryAfterFrom(header)
		if wait == 0 && decoded.RetryAfter > 0 {
			wait = time.Duration(decoded.RetryAfter * float64(time.Second))
		}
		return probeRateLimited(detail, wait)
	case decoded.Code == 10015 || decoded.Code == 50027:
		return probeFail(health.KindDead, detail)
	case status == http.StatusNotFound || status == http.StatusUnauthorized:
		return probeFail(health.KindDead, detail)
	case status == http.StatusForbidden:
		return probeFail(health.KindDenied, detail)
	}
	return probeFail(health.KindUnreachable, detail)
}

// -- ntfy --------------------------------------------------------------------

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

	status, body, header, err := s.probeGET(ctx, server.JoinPath("v1", "health").String(), nil)
	if err != nil {
		return probeFail(health.KindUnreachable, SafeErrorText(err))
	}
	detail := httpDetail("ntfy", status, body)
	if status == http.StatusTooManyRequests {
		return probeRateLimited(detail, retryAfterFrom(header))
	}
	if status < 200 || status >= 300 {
		return probeFail(health.KindUnreachable, detail)
	}
	var reported struct {
		Healthy bool `json:"healthy"`
	}
	if json.Unmarshal(body, &reported) != nil || !reported.Healthy {
		return probeFail(health.KindUnreachable, detail)
	}

	if target.AccessToken == "" {
		// An anonymous publish target is as verified as it can be: topics are
		// created on publish, so "the topic exists" is not a question.
		return probeOK("server healthy")
	}

	status, body, header, err = s.probeGET(ctx, server.JoinPath("v1", "account").String(),
		http.Header{"Authorization": []string{"Bearer " + target.AccessToken}})
	if err != nil {
		return probeFail(health.KindUnreachable, SafeErrorText(err))
	}
	detail = httpDetail("ntfy", status, body)

	switch {
	case status == http.StatusTooManyRequests:
		return probeRateLimited(detail, retryAfterFrom(header))
	case status == http.StatusUnauthorized:
		return probeFail(health.KindDead, detail)
	case status == http.StatusForbidden:
		return probeFail(health.KindDenied, detail)
	case status < 200 || status >= 300:
		return probeFail(health.KindUnreachable, detail)
	}

	var account struct {
		Username string `json:"username"`
	}
	_ = json.Unmarshal(body, &account)
	// "*" is ntfy's anonymous user.
	if account.Username == "" || account.Username == "*" {
		return probeFail(health.KindDenied,
			"ntfy answered as the anonymous user, so the access token never reached it")
	}

	return probeOK("authenticated as " + account.Username)
}

// -- Perception --------------------------------------------------------------

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

	status, body, header, err := s.probeGET(ctx, endpoint, nil)
	if err != nil {
		return probeFail(health.KindUnreachable, SafeErrorText(err))
	}
	detail := httpDetail("Perception", status, body)

	switch {
	case status >= 200 && status < 300:
		return probeOK("server healthy")
	case status == http.StatusTooManyRequests:
		return probeRateLimited(detail, retryAfterFrom(header))
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return probeFail(health.KindDenied, detail)
	}
	return probeFail(health.KindUnreachable, detail)
}

// -- SMTP --------------------------------------------------------------------

// probeSMTP connects, negotiates TLS, authenticates and quits. It never issues
// MAIL FROM (the classic spammer fingerprint, and a fast way to get greylisted)
// and never VRFY (disabled nearly everywhere, and its replies prove nothing).
// QUIT is the right ending; RSET would be noise with no transaction to reset.
func (s *Sender) probeSMTP(target settings.DeliveryTarget) health.Observation {
	address := net.JoinHostPort(target.Host, strconv.Itoa(target.Port))

	var client *smtp.Client
	var err error

	if target.Secure {
		connection, dialErr := tls.Dial("tcp", address, &tls.Config{ServerName: target.Host})
		if dialErr != nil {
			return probeFail(health.KindUnreachable, dialErr.Error())
		}
		client, err = smtp.NewClient(connection, target.Host)
		if err != nil {
			connection.Close()
			return classifySMTPError(err)
		}
	} else {
		client, err = smtp.Dial(address)
		if err != nil {
			return probeFail(health.KindUnreachable, SafeErrorText(err))
		}
	}
	defer client.Close()

	// EHLO before anything else, so the greeting carries our name and StartTLS
	// can re-issue it: RFC 3207 requires discarding the capability list learned
	// before TLS, and net/smtp only re-sends EHLO for the name it already holds.
	if err := client.Hello("antenne"); err != nil {
		return classifySMTPError(err)
	}

	if !target.Secure {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(&tls.Config{ServerName: target.Host}); err != nil {
				return classifySMTPError(err)
			}
		}
	}

	if target.Username != "" && target.Password != nil && *target.Password != "" {
		auth := smtp.PlainAuth("", target.Username, *target.Password, target.Host)
		if err := client.Auth(auth); err != nil {
			return classifySMTPError(err)
		}
	}

	if err := client.Quit(); err != nil {
		return classifySMTPError(err)
	}
	return probeOK("connected to " + address)
}

func classifySMTPError(err error) health.Observation {
	var reply *textproto.Error
	if errors.As(err, &reply) {
		return classifySMTPReply(reply.Code, reply.Msg)
	}
	return probeFail(health.KindUnreachable, SafeErrorText(err))
}

// classifySMTPReply maps a reply code to a Kind, and is a pure function because
// this mapping is the part worth testing and the part worth getting right.
//
// Only 535 proves the credential is wrong. 454 is a temporary auth failure —
// Google's throttle and Microsoft's smart lockout both answer with it — and
// treating it as a bad password is how a monitor locks the account it monitors.
func classifySMTPReply(code int, message string) health.Observation {
	detail := strings.TrimSpace(fmt.Sprintf("%d %s", code, message))

	switch code {
	case 535:
		return probeFail(health.KindDead, detail)
	case 432:
		return probeFail(health.KindRefreshable, detail)
	case 454:
		return probeRateLimited(detail, 0)
	case 421, 450, 451, 452:
		return probeFail(health.KindUnreachable, detail)
	case 530, 534, 538:
		return probeFail(health.KindDenied, detail)
	}

	switch {
	case code >= 200 && code < 400:
		return probeOK(detail)
	case code < 500:
		return probeFail(health.KindUnreachable, detail)
	}
	// Every other permanent refusal is the far side declining us, not proof the
	// password is wrong. Blaming the credential here would suspend probing on a
	// server that is merely misconfigured.
	return probeFail(health.KindDenied, detail)
}

// -- shared ------------------------------------------------------------------

func (s *Sender) probeGET(ctx context.Context, endpoint string, header http.Header) (int, []byte, http.Header, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, nil, nil, err
	}
	for name, values := range header {
		for _, value := range values {
			request.Header.Add(name, value)
		}
	}
	request.Header.Set("User-Agent", "Antenne/1.0 uptime-bot")

	response, err := s.client.Do(request)
	if err != nil {
		return 0, nil, nil, err
	}
	defer response.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(response.Body, 1<<14))
	return response.StatusCode, body, response.Header, nil
}

// httpDetail keeps the far side's own words, which is what an operator pastes
// into a search or forwards to their provider.
func httpDetail(label string, status int, body []byte) string {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return fmt.Sprintf("%s -> HTTP %d", label, status)
	}
	return fmt.Sprintf("%s -> HTTP %d %s", label, status, trimmed)
}

// retryAfterFrom reads the binding wait the far side asked for, in either of the
// two forms RFC 9110 allows.
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
