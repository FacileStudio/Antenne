package health

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// Jitter on a heartbeat goes one way only. Symmetric jitter sends some pings late,
// which spends the receiver's grace time on our own randomness and manufactures
// "late" alarms about a perfectly healthy service.
func TestNextBeatInJittersDownwardOnly(t *testing.T) {
	interval := 5 * time.Minute

	for _, draw := range []float64{0, 0.1, 0.5, 0.9, 0.999999} {
		got := NextBeatIn(interval, func() float64 { return draw })
		if got >= interval {
			t.Fatalf("draw %v produced %s, which is not shorter than the interval %s", draw, got, interval)
		}
		if got < interval/2 {
			t.Fatalf("draw %v produced %s, under half the interval — that is wasted requests, not safety", draw, got)
		}
	}
}

func TestNewHeartbeatIsOffWithoutAURL(t *testing.T) {
	if NewHeartbeat(quietLogger(), "   ", time.Minute, nil) != nil {
		t.Fatal("an absent HEARTBEAT_URL must turn the feature off, silently")
	}
	beat := NewHeartbeat(quietLogger(), "https://hc.example/ping/abc/", 0, nil)
	if beat == nil {
		t.Fatal("a configured URL must produce a heartbeat")
	}
	if beat.url != "https://hc.example/ping/abc" {
		t.Fatalf("url = %q; the trailing slash must go or the /fail suffix lands on a double slash", beat.url)
	}
	if beat.Interval() != DefaultHeartbeatInterval {
		t.Fatalf("interval = %s, want the default %s", beat.Interval(), DefaultHeartbeatInterval)
	}
}

type recordingReceiver struct {
	mu    sync.Mutex
	paths []string
}

func (r *recordingReceiver) ServeHTTP(_ http.ResponseWriter, request *http.Request) {
	r.mu.Lock()
	r.paths = append(r.paths, request.URL.Path)
	r.mu.Unlock()
}

func (r *recordingReceiver) seen() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.paths...)
}

func TestBeat(t *testing.T) {
	cases := []struct {
		name   string
		vitals Vitals
		want   []string
	}{
		{
			name:   "functional and healthy pings the plain URL",
			vitals: Vitals{Functional: true},
			want:   []string{"/ping"},
		},
		{
			name:   "a target down self-reports through /fail",
			vitals: Vitals{Functional: true, Degraded: true},
			want:   []string{"/ping/fail"},
		},
		{
			// A heartbeat a zombie process can still send is worthless. Withholding it
			// entirely, rather than sending /fail, is deliberate: a process too broken
			// to do its job is too broken to be trusted about how broken it is, and the
			// receiver's grace timer is a signal that does not depend on us at all.
			name:   "a stalled loop sends nothing at all",
			vitals: Vitals{Functional: false, Reason: "the monitoring sweep last completed 40m ago"},
			want:   []string{},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			receiver := &recordingReceiver{}
			server := httptest.NewServer(receiver)
			defer server.Close()

			beat := NewHeartbeat(quietLogger(), server.URL+"/ping", time.Minute,
				func(context.Context) Vitals { return testCase.vitals })
			beat.beat(context.Background())

			seen := receiver.seen()
			if strings.Join(seen, ",") != strings.Join(testCase.want, ",") {
				t.Fatalf("receiver saw %v, want %v", seen, testCase.want)
			}
		})
	}
}

func TestStartAndStopIsNilSafe(t *testing.T) {
	var beat *Heartbeat
	beat.Start(context.Background())
	beat.Stop()
	if beat.Interval() != 0 {
		t.Fatal("a nil heartbeat must report no interval")
	}
}
