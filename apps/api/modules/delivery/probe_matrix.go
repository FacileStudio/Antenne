package delivery

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/FacileStudio/Antenne/apps/api/modules/health"
	"github.com/FacileStudio/Antenne/apps/api/modules/settings"
)

func whoamiCall(target settings.DeliveryTarget, token string) matrixCall {
	return matrixCall{
		homeserver: target.Homeserver,
		token:      token,
		method:     http.MethodGet,
		path:       "/_matrix/client/v3/account/whoami",
	}
}

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

	whoami, token, err := s.matrixWhoami(ctx, target, session, token)
	if err != nil {
		return classifyMatrixError(err)
	}

	if observation, mismatch := whoamiMismatch(target, whoami); mismatch {
		return observation
	}

	if observation, denied := s.missingRooms(ctx, target, token); denied {
		return observation
	}

	return probeOK(fmt.Sprintf("%s in %d room(s)", whoami.UserID, len(target.RoomIDs)))
}

// matrixWhoami identifies the account the token belongs to, re-logging in once
// if the homeserver says the token is unknown.
//
// The cached token being gone is not the same as the credential being gone.
// Revoking a session invalidates its token — removing the device from Element
// does exactly that — and so does a homeserver rotating one. The password that
// would fix it is right here, and the send path already re-logs in on this
// errcode, so a probe that instead reports KindDead suspends itself for an hour
// and tells an operator to rotate a credential that still works. If the
// re-login is genuinely refused the homeserver answers M_FORBIDDEN, which
// classifies as dead on its own merits.
//
// The token it returns is the one that worked, which is the one the rest of the
// probe must keep using.
func (s *Sender) matrixWhoami(ctx context.Context, target settings.DeliveryTarget, session *matrixSession, token string) (matrixWhoamiResponse, string, error) {
	var whoami matrixWhoamiResponse
	err := s.matrixJSON(ctx, whoamiCall(target, token), &whoami)
	if !isUnknownMatrixToken(err) || !hasPasswordAuth(target) {
		return whoami, token, err
	}

	token, err = s.ensureMatrixSession(ctx, target, session, true)
	if err != nil {
		return whoami, token, err
	}
	err = s.matrixJSON(ctx, whoamiCall(target, token), &whoami)
	return whoami, token, err
}

// whoamiMismatch catches a token that answers as somebody else. It is not
// KindDead — the homeserver accepted it — but it cannot deliver as the
// configured identity until an operator intervenes.
func whoamiMismatch(target settings.DeliveryTarget, whoami matrixWhoamiResponse) (health.Observation, bool) {
	if target.UserID != "" && whoami.UserID != target.UserID {
		return probeFail(health.KindDenied,
			fmt.Sprintf("token belongs to %s, target is configured as %s", whoami.UserID, target.UserID)), true
	}
	if target.DeviceID != nil && *target.DeviceID != "" && whoami.DeviceID != "" && whoami.DeviceID != *target.DeviceID {
		return probeFail(health.KindDenied,
			fmt.Sprintf("token belongs to device %s, target is configured for device %s", whoami.DeviceID, *target.DeviceID)), true
	}
	return health.Observation{}, false
}

// missingRooms checks deliverability separately from the credential: a valid
// token says nothing about being joined to the rooms the target posts in, and
// the two failures need different fixes.
func (s *Sender) missingRooms(ctx context.Context, target settings.DeliveryTarget, token string) (health.Observation, bool) {
	var joined matrixJoinedRoomsResponse
	if err := s.matrixJSON(ctx, matrixCall{homeserver: target.Homeserver, token: token, method: http.MethodGet, path: "/_matrix/client/v3/joined_rooms"}, &joined); err != nil {
		return classifyMatrixError(err), true
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
		return probeFail(health.KindDenied, fmt.Sprintf("not joined to %s", strings.Join(missing, ", "))), true
	}
	return health.Observation{}, false
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

	userID, err := s.matrixUserID(ctx, target, token)
	if err != nil {
		return classifyMatrixError(err)
	}

	for _, roomID := range target.RoomIDs {
		if observation, denied := s.probeRoomPowerLevel(ctx, target, token, userID, roomID); denied {
			return observation
		}
	}

	return probeOK(fmt.Sprintf("can post in %d room(s)", len(target.RoomIDs)))
}

// matrixUserID resolves who the token speaks as, asking the homeserver only
// when the target does not already say.
func (s *Sender) matrixUserID(ctx context.Context, target settings.DeliveryTarget, token string) (string, error) {
	if target.UserID != "" {
		return target.UserID, nil
	}
	var whoami matrixWhoamiResponse
	if err := s.matrixJSON(ctx, whoamiCall(target, token), &whoami); err != nil {
		return "", err
	}
	return whoami.UserID, nil
}

// probeRoomPowerLevel reports whether one room refuses the bot, and why. The
// second result is false for every healthy outcome, including a room with no
// m.room.power_levels event at all: the spec's default is events_default 0,
// which anyone in the room clears.
func (s *Sender) probeRoomPowerLevel(ctx context.Context, target settings.DeliveryTarget, token, userID, roomID string) (health.Observation, bool) {
	path := fmt.Sprintf("/_matrix/client/v3/rooms/%s/state/m.room.power_levels", url.PathEscape(roomID))

	var levels matrixPowerLevels
	if err := s.matrixJSON(ctx, matrixCall{homeserver: target.Homeserver, token: token, method: http.MethodGet, path: path}, &levels); err != nil {
		return classifyPowerLevelError(err, roomID)
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
			fmt.Sprintf("power level %d in %s, m.room.message needs %d", mine, roomID, required)), true
	}
	return health.Observation{}, false
}

func classifyPowerLevelError(err error, roomID string) (health.Observation, bool) {
	var apiErr *matrixError
	if asMatrixError(err, &apiErr) && apiErr.status == http.StatusNotFound {
		return health.Observation{}, false
	}
	if apiErr != nil && apiErr.status == http.StatusForbidden {
		return probeFail(health.KindDenied,
			fmt.Sprintf("cannot read power levels in %s: %s", roomID, apiErr.body)), true
	}
	return classifyMatrixError(err), true
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
	decodeHint([]byte(apiErr.body), &decoded)

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
