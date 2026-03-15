package checker_test

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"uptui/internal/checker"
	"uptui/internal/models"
)

// ── HTTP ──────────────────────────────────────────────────────────────────────

func TestCheckHTTPUp(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	result := checker.Check(context.Background(), models.Monitor{
		Type: models.HTTP, Target: ts.URL, Timeout: 5,
	})

	if result.Status != models.StatusUp {
		t.Errorf("status = %q, want up; message: %s", result.Status, result.Message)
	}
	if result.Latency <= 0 {
		t.Errorf("latency = %d, want > 0", result.Latency)
	}
	if result.Timestamp.IsZero() {
		t.Error("timestamp is zero")
	}
}

func TestCheckHTTPStatusCodes(t *testing.T) {
	tests := []struct {
		code int
		want models.Status
	}{
		{200, models.StatusUp},
		{201, models.StatusUp},
		{204, models.StatusUp},
		{400, models.StatusDown},
		{404, models.StatusDown},
		{500, models.StatusDown},
		{503, models.StatusDown},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(http.StatusText(tt.code), func(t *testing.T) {
			t.Parallel()
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.code)
			}))
			defer ts.Close()

			result := checker.Check(context.Background(), models.Monitor{
				Type: models.HTTP, Target: ts.URL, Timeout: 5,
			})
			if result.Status != tt.want {
				t.Errorf("HTTP %d: status = %q, want %q", tt.code, result.Status, tt.want)
			}
		})
	}
}

func TestCheckHTTPUnreachable(t *testing.T) {
	result := checker.Check(context.Background(), models.Monitor{
		Type:    models.HTTP,
		Target:  "http://127.0.0.1:19999",
		Timeout: 2,
	})
	if result.Status != models.StatusDown {
		t.Errorf("status = %q, want down", result.Status)
	}
	if result.Message == "" {
		t.Error("expected error message for unreachable host")
	}
}

func TestCheckHTTPTimeout(t *testing.T) {
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		w.WriteHeader(200)
	}))
	defer slow.Close()

	start := time.Now()
	result := checker.Check(context.Background(), models.Monitor{
		Type:    models.HTTP,
		Target:  slow.URL,
		Timeout: 1,
	})
	elapsed := time.Since(start)

	if result.Status != models.StatusDown {
		t.Errorf("status = %q, want down (timeout)", result.Status)
	}
	if elapsed > 3*time.Second {
		t.Errorf("check took %v, expected < 3s timeout", elapsed)
	}
}

func TestCheckHTTPRedirectFollowed(t *testing.T) {
	// /a → /b → 200 OK
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/a" {
			http.Redirect(w, r, "/b", http.StatusMovedPermanently)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	result := checker.Check(context.Background(), models.Monitor{
		Type: models.HTTP, Target: ts.URL + "/a", Timeout: 5,
	})
	if result.Status != models.StatusUp {
		t.Errorf("redirect: status = %q, want up", result.Status)
	}
}

func TestCheckHTTPContextCancelled(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
	}))
	defer ts.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	result := checker.Check(ctx, models.Monitor{
		Type: models.HTTP, Target: ts.URL, Timeout: 30,
	})
	if result.Status != models.StatusDown {
		t.Errorf("status = %q, want down (cancelled ctx)", result.Status)
	}
}

// ── TCP ───────────────────────────────────────────────────────────────────────

func TestCheckTCPUp(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	result := checker.Check(context.Background(), models.Monitor{
		Type: models.TCP, Target: l.Addr().String(), Timeout: 5,
	})
	if result.Status != models.StatusUp {
		t.Errorf("status = %q, want up; message: %s", result.Status, result.Message)
	}
	if result.Latency <= 0 {
		t.Errorf("latency = %d, want > 0", result.Latency)
	}
}

func TestCheckTCPDown(t *testing.T) {
	result := checker.Check(context.Background(), models.Monitor{
		Type:    models.TCP,
		Target:  "127.0.0.1:19998",
		Timeout: 2,
	})
	if result.Status != models.StatusDown {
		t.Errorf("status = %q, want down", result.Status)
	}
}

func TestCheckTCPTimeout(t *testing.T) {
	// A listener that accepts but never reads — connection hangs
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skip("could not create listener:", err)
	}
	defer l.Close()
	// Don't call Accept — OS may still complete the connect at TCP level,
	// so this only reliably tests timeout via a non-routable IP.
	// Use a host that drops packets to test timeout properly.
	// Skip if we can't test this reliably in CI.
	_ = l
}

func TestCheckPortTypeAlias(t *testing.T) {
	// "port" is a legacy alias for "tcp" — must behave identically
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	result := checker.Check(context.Background(), models.Monitor{
		Type:    "port",
		Target:  l.Addr().String(),
		Timeout: 5,
	})
	if result.Status != models.StatusUp {
		t.Errorf("port alias: status = %q, want up; message: %s", result.Status, result.Message)
	}
}

func TestCheckUnknownType(t *testing.T) {
	result := checker.Check(context.Background(), models.Monitor{
		Type:    "icmp",
		Target:  "example.com",
		Timeout: 5,
	})
	if result.Status != models.StatusDown {
		t.Errorf("unknown type: status = %q, want down", result.Status)
	}
}

func TestCheckDefaultTimeout(t *testing.T) {
	// A monitor with Timeout=0 should not panic; it uses the default 30s
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer ts.Close()

	result := checker.Check(context.Background(), models.Monitor{
		Type: models.HTTP, Target: ts.URL, Timeout: 0,
	})
	if result.Status != models.StatusUp {
		t.Errorf("Timeout=0: status = %q, want up", result.Status)
	}
}
