package checker

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"

	"uptui/internal/models"
)

func Check(ctx context.Context, m models.Monitor) models.Result {
	timeout := time.Duration(m.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()

	switch m.Type {
	case models.HTTP:
		return checkHTTP(ctx, m, start)
	case models.TCP:
		return checkTCP(ctx, m, start)
	default:
		return models.Result{
			Timestamp: time.Now(),
			Status:    models.StatusDown,
			Message:   fmt.Sprintf("unknown type: %s", m.Type),
		}
	}
}

func checkHTTP(ctx context.Context, m models.Monitor, start time.Time) models.Result {
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
			DialContext: (&net.Dialer{
				Timeout:   15 * time.Second,
				KeepAlive: 15 * time.Second,
			}).DialContext,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	req, err := http.NewRequestWithContext(ctx, "GET", m.Target, nil)
	if err != nil {
		return models.Result{
			Timestamp: time.Now(),
			Status:    models.StatusDown,
			Message:   err.Error(),
		}
	}
	req.Header.Set("User-Agent", "uptui/1.0")

	resp, err := client.Do(req)
	latency := int(time.Since(start).Milliseconds())

	if err != nil {
		return models.Result{
			Timestamp: time.Now(),
			Status:    models.StatusDown,
			Latency:   latency,
			Message:   err.Error(),
		}
	}
	resp.Body.Close()

	status := models.StatusUp
	msg := fmt.Sprintf("HTTP %d", resp.StatusCode)
	if resp.StatusCode >= 400 {
		status = models.StatusDown
	}

	return models.Result{
		Timestamp: time.Now(),
		Status:    status,
		Latency:   latency,
		Message:   msg,
	}
}

func checkTCP(ctx context.Context, m models.Monitor, start time.Time) models.Result {
	d := &net.Dialer{}
	conn, err := d.DialContext(ctx, "tcp", m.Target)
	latency := int(time.Since(start).Milliseconds())

	if err != nil {
		return models.Result{
			Timestamp: time.Now(),
			Status:    models.StatusDown,
			Latency:   latency,
			Message:   err.Error(),
		}
	}
	conn.Close()

	return models.Result{
		Timestamp: time.Now(),
		Status:    models.StatusUp,
		Latency:   latency,
		Message:   "TCP ok",
	}
}
