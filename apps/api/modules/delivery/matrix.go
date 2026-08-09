package delivery

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/FacileStudio/Antenne/apps/api/internal/envelope"
	"github.com/FacileStudio/Antenne/apps/api/modules/settings"
)

// matrixSession is the live credential for one Matrix target, kept in memory so
// a password-authenticated target logs in once per boot rather than per alert.
type matrixSession struct {
	mu           sync.Mutex
	accessToken  string
	refreshToken string
	deviceID     string
	expiresAt    time.Time
}

type matrixError struct {
	status  int
	errcode string
	body    string
	method  string
	path    string
}

func (e *matrixError) Error() string {
	if e.body != "" {
		return fmt.Sprintf("Matrix API %s %s -> HTTP %d %s", e.method, e.path, e.status, e.body)
	}
	return fmt.Sprintf("Matrix API %s %s -> HTTP %d", e.method, e.path, e.status)
}

var matrixTransaction atomic.Uint64

func (s *Sender) sendMatrix(ctx context.Context, target settings.DeliveryTarget, message string) error {
	session := s.matrixSessionFor(target)

	var failures []string
	for _, roomID := range target.RoomIDs {
		if err := s.sendMatrixRoom(ctx, target, session, roomID, message); err != nil {
			s.log.Error("matrix delivery failed",
				slog.String("target", target.Name),
				slog.String("room", roomID),
				slog.Any("error", err))
			failures = append(failures, roomID)
		}
	}

	if len(failures) > 0 {
		return fmt.Errorf("matrix delivery failed for %d/%d rooms", len(failures), len(target.RoomIDs))
	}
	return nil
}

func (s *Sender) sendMatrixRoom(ctx context.Context, target settings.DeliveryTarget, session *matrixSession, roomID, message string) error {
	token, err := s.ensureMatrixSession(ctx, target, session, false)
	if err != nil {
		return err
	}

	transaction := fmt.Sprintf("%d-%d", time.Now().UnixMilli(), matrixTransaction.Add(1))
	path := fmt.Sprintf("/_matrix/client/v3/rooms/%s/send/m.room.message/%s",
		url.PathEscape(roomID), url.PathEscape(transaction))

	body := map[string]string{
		"msgtype":        "m.text",
		"body":           envelope.HTMLToPlainText(message),
		"format":         "org.matrix.custom.html",
		"formatted_body": message,
	}

	err = s.matrixRequest(ctx, target.Homeserver, token, http.MethodPut, path, body)
	if err == nil {
		return nil
	}

	// A homeserver that has rotated or revoked the token answers M_UNKNOWN_TOKEN.
	// That is the one failure worth a forced re-login before giving up.
	var apiErr *matrixError
	if !asMatrixError(err, &apiErr) || apiErr.errcode != "M_UNKNOWN_TOKEN" {
		return err
	}

	token, err = s.ensureMatrixSession(ctx, target, session, true)
	if err != nil {
		return err
	}
	return s.matrixRequest(ctx, target.Homeserver, token, http.MethodPut, path, body)
}

func (s *Sender) matrixSessionFor(target settings.DeliveryTarget) *matrixSession {
	s.matrixMu.Lock()
	defer s.matrixMu.Unlock()

	session, ok := s.matrix[target.ID]
	if !ok {
		session = &matrixSession{}
		s.matrix[target.ID] = session
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	// The configured values win whenever they change: an operator pasting a
	// fresh access token into the dashboard must take effect without a restart.
	if target.AccessToken != "" && target.AccessToken != session.accessToken && session.expiresAt.IsZero() {
		session.accessToken = target.AccessToken
	}
	if session.accessToken == "" {
		session.accessToken = target.AccessToken
	}
	if target.RefreshToken != nil && *target.RefreshToken != "" && session.refreshToken == "" {
		session.refreshToken = *target.RefreshToken
	}
	// A login always names a device. Without one the homeserver mints a fresh
	// device on every login, which means one per restart, forever, until the
	// account hits its device limit and stops being able to log in at all. The
	// derived id covers targets configured before deviceId existed.
	if session.deviceID == "" {
		if target.DeviceID != nil && *target.DeviceID != "" {
			session.deviceID = *target.DeviceID
		} else {
			session.deviceID = settings.MatrixDeviceID(target.ID)
		}
	}

	return session
}

func (s *Sender) ensureMatrixSession(ctx context.Context, target settings.DeliveryTarget, session *matrixSession, force bool) (string, error) {
	session.mu.Lock()
	defer session.mu.Unlock()

	fresh := session.expiresAt.IsZero() || session.expiresAt.After(time.Now().Add(15*time.Second))
	if !force && session.accessToken != "" && fresh {
		return session.accessToken, nil
	}

	if session.refreshToken != "" {
		if err := s.refreshMatrix(ctx, target, session); err == nil {
			return session.accessToken, nil
		} else if !hasPasswordAuth(target) {
			return "", err
		}
	}

	if hasPasswordAuth(target) {
		if err := s.loginMatrix(ctx, target, session); err != nil {
			return "", err
		}
		return session.accessToken, nil
	}

	if session.accessToken != "" && !force {
		return session.accessToken, nil
	}

	return "", fmt.Errorf("matrix target `%s` needs either an access token or password login", target.Name)
}

type matrixLoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	UserID       string `json:"user_id"`
	DeviceID     string `json:"device_id"`
	ExpiresInMS  int64  `json:"expires_in_ms"`
}

func (s *Sender) loginMatrix(ctx context.Context, target settings.DeliveryTarget, session *matrixSession) error {
	// A full user id (@antenne:server) has to be reduced to its localpart for the
	// m.id.user identifier; passing the full id makes some homeservers 403.
	loginUser := target.UserID
	if strings.HasPrefix(loginUser, "@") && strings.Contains(loginUser, ":") {
		loginUser = loginUser[1:strings.Index(loginUser, ":")]
	}

	body := map[string]any{
		"type":                        "m.login.password",
		"identifier":                  map[string]string{"type": "m.id.user", "user": loginUser},
		"password":                    deref(target.Password),
		"initial_device_display_name": "Antenne",
		"refresh_token":               true,
	}
	if session.deviceID != "" {
		body["device_id"] = session.deviceID
	}

	var response matrixLoginResponse
	if err := s.matrixJSON(ctx, target.Homeserver, "", http.MethodPost, "/_matrix/client/v3/login", body, &response); err != nil {
		return fmt.Errorf("matrix login failed: %w", err)
	}

	session.accessToken = response.AccessToken
	if response.RefreshToken != "" {
		session.refreshToken = response.RefreshToken
	}
	if response.DeviceID != "" {
		session.deviceID = response.DeviceID
	}
	session.expiresAt = time.Time{}
	if response.ExpiresInMS > 0 {
		session.expiresAt = time.Now().Add(time.Duration(response.ExpiresInMS) * time.Millisecond)
	}
	return nil
}

func (s *Sender) refreshMatrix(ctx context.Context, target settings.DeliveryTarget, session *matrixSession) error {
	var response matrixLoginResponse
	body := map[string]string{"refresh_token": session.refreshToken}
	if err := s.matrixJSON(ctx, target.Homeserver, "", http.MethodPost, "/_matrix/client/v3/refresh", body, &response); err != nil {
		return fmt.Errorf("matrix token refresh failed: %w", err)
	}

	session.accessToken = response.AccessToken
	if response.RefreshToken != "" {
		session.refreshToken = response.RefreshToken
	}
	session.expiresAt = time.Time{}
	if response.ExpiresInMS > 0 {
		session.expiresAt = time.Now().Add(time.Duration(response.ExpiresInMS) * time.Millisecond)
	}
	return nil
}

func (s *Sender) matrixRequest(ctx context.Context, homeserver, token, method, path string, body any) error {
	return s.matrixJSON(ctx, homeserver, token, method, path, body, nil)
}

func (s *Sender) matrixJSON(ctx context.Context, homeserver, token, method, path string, body any, out any) error {
	base := homeserver
	if !strings.HasSuffix(base, "/") {
		base += "/"
	}
	endpoint, err := url.JoinPath(base, path)
	if err != nil {
		return err
	}

	var payload io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return err
		}
		payload = bytes.NewReader(encoded)
	}

	request, err := http.NewRequestWithContext(ctx, method, endpoint, payload)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", "Antenne/1.0 uptime-bot")
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}

	response, err := s.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	raw, _ := io.ReadAll(io.LimitReader(response.Body, 1<<16))

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		apiErr := &matrixError{status: response.StatusCode, body: strings.TrimSpace(string(raw)), method: method, path: path}
		var decoded struct {
			ErrCode string `json:"errcode"`
		}
		if json.Unmarshal(raw, &decoded) == nil {
			apiErr.errcode = decoded.ErrCode
		}
		return apiErr
	}

	if out != nil {
		return json.Unmarshal(raw, out)
	}
	return nil
}

func asMatrixError(err error, target **matrixError) bool {
	for err != nil {
		if typed, ok := err.(*matrixError); ok {
			*target = typed
			return true
		}
		unwrapper, ok := err.(interface{ Unwrap() error })
		if !ok {
			return false
		}
		err = unwrapper.Unwrap()
	}
	return false
}
