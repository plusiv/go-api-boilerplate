package http

import (
	"context"
	"io"
	"log/slog"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/plusiv/go-api-boilerplate/internal/application/usecases"
)

func TestPrometheusMetrics(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	items := usecases.NewItems(loggingRepo{})
	disabled, err := NewRouter(items, log, RouterConfig{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewRouter(items, log, RouterConfig{MetricsEnabled: true}); err == nil {
		t.Fatal("expected token error")
	}
	router, err := NewRouter(items, log, RouterConfig{MetricsEnabled: true, MetricsToken: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	request := func(handler nethttp.Handler, method, path, token, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequestWithContext(context.Background(), method, path, strings.NewReader(body))
		req.Host = "untrusted.example"
		req.Header.Set("Content-Type", "application/json")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}
	if got := request(disabled, "GET", "/metrics", "", ""); got.Code != 404 {
		t.Fatalf("disabled status %d", got.Code)
	}
	for _, token := range []string{"", "wrong"} {
		if got := request(router, "GET", "/metrics", token, ""); got.Code != 401 {
			t.Fatalf("unauthorized status %d", got.Code)
		}
	}
	cases := []struct {
		method, path, body string
		status             int
	}{
		{"GET", "/healthz", "", 200}, {"POST", "/items", `{"id":" ","name":""}`, 422}, {"GET", "/items/1", "", 500},
		{"GET", "/unknown-one", "", 404}, {"GET", "/unknown-two", "", 404}, {"CUSTOM", "/another-unknown", "", 404},
	}
	for _, tt := range cases {
		if got := request(router, tt.method, tt.path, "", tt.body); got.Code != tt.status {
			t.Fatalf("%s %s status %d", tt.method, tt.path, got.Code)
		}
	}
	scrape := request(router, "GET", "/metrics", "secret", "")
	if scrape.Code != 200 {
		t.Fatalf("scrape status %d: %s", scrape.Code, scrape.Body.String())
	}
	body := scrape.Body.String()
	for _, want := range []string{"go_goroutines", "process_cpu_seconds_total", "go_build_info", "echo_request_duration_seconds", `echo_requests_total{code="200",host="",method="GET",url="/healthz"} 1`, `echo_requests_total{code="422",host="",method="POST",url="/items"} 1`, `echo_requests_total{code="500",host="",method="GET",url="/items/:id"} 1`, `echo_requests_total{code="404",host="",method="GET",url=""} 2`, `method="OTHER"`} {
		if !strings.Contains(body, want) {
			t.Errorf("missing metric %s", want)
		}
	}
	for _, unwanted := range []string{"untrusted.example", "unknown-one", "unknown-two", `url="/metrics"`} {
		if strings.Contains(body, unwanted) {
			t.Errorf("unexpected metric label %q", unwanted)
		}
	}
}
