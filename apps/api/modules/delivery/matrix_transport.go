package delivery

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

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

// matrixCall is one request to a homeserver. The parts travel together because
// the interesting half of a call site is the method and the path, and the other
// half — which homeserver, under which token — is connection state that does not
// vary within a single probe or delivery.
type matrixCall struct {
	homeserver string
	token      string
	method     string
	path       string
	body       any
}

func (s *Sender) matrixRequest(ctx context.Context, call matrixCall) error {
	return s.matrixJSON(ctx, call, nil)
}

func (s *Sender) matrixJSON(ctx context.Context, call matrixCall, out any) error {
	base := call.homeserver
	if !strings.HasSuffix(base, "/") {
		base += "/"
	}
	endpoint, err := url.JoinPath(base, call.path)
	if err != nil {
		return err
	}

	var payload io.Reader
	if call.body != nil {
		encoded, err := json.Marshal(call.body)
		if err != nil {
			return err
		}
		payload = bytes.NewReader(encoded)
	}

	request, err := http.NewRequestWithContext(ctx, call.method, endpoint, payload)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", "Antenne/1.0 uptime-bot")
	if call.token != "" {
		request.Header.Set("Authorization", "Bearer "+call.token)
	}

	response, err := s.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	raw, _ := io.ReadAll(io.LimitReader(response.Body, 1<<16))

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		apiErr := &matrixError{status: response.StatusCode, body: strings.TrimSpace(string(raw)), method: call.method, path: call.path}
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

// isUnknownMatrixToken reports the one failure a re-login fixes: the token we
// hold is no longer a token the homeserver knows.
func isUnknownMatrixToken(err error) bool {
	var apiErr *matrixError
	return asMatrixError(err, &apiErr) && apiErr.errcode == "M_UNKNOWN_TOKEN"
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
