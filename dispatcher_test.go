package tokka

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/starwalkn/tokka/internal/metric"

	"go.uber.org/zap"
)

type testMetrics struct{}

func (m *testMetrics) IncRequestsTotal()                          {}
func (m *testMetrics) UpdateRequestsDuration(_ time.Time)         {}
func (m *testMetrics) IncResponsesTotal(_ int)                    {}
func (m *testMetrics) IncRequestsInFlight()                       {}
func (m *testMetrics) DecRequestsInFlight()                       {}
func (m *testMetrics) IncFailedRequestsTotal(_ metric.FailReason) {}
func (m *testMetrics) IncCounter(_ string, _ ...zap.Field)        {}

func TestDispatcher_Dispatch_Success(t *testing.T) {
	upstreamA := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		w.Write([]byte("A"))
	}))
	defer upstreamA.Close()

	upstreamB := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		w.Write([]byte("B"))
	}))
	defer upstreamB.Close()

	d := &defaultDispatcher{
		log:     zap.NewNop(),
		metrics: &testMetrics{},
	}

	route := &Route{
		Upstreams: []Upstream{
			&httpUpstream{url: upstreamA.URL, timeout: 1000, client: http.DefaultClient},
			&httpUpstream{url: upstreamB.URL, timeout: 1000, client: http.DefaultClient},
		},
	}

	originalRequest := httptest.NewRequest(http.MethodGet, "http://example.com/test", nil)

	results := d.dispatch(route, originalRequest)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	got := string(results[0].Body) + string(results[1].Body)
	want1 := "AB"
	if got != want1 {
		t.Errorf("unexpected results: %q", got)
	}
}

func TestDispatcher_Dispatch_ForwardQueryAndHeaders(t *testing.T) {
	upstreamA := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("foo")
		h := r.Header.Get("X-Test")

		w.Write([]byte(q + "-" + h))
	}))
	defer upstreamA.Close()

	d := &defaultDispatcher{
		log:     zap.NewNop(),
		metrics: &testMetrics{},
	}

	route := &Route{
		Upstreams: []Upstream{
			&httpUpstream{
				url:                 upstreamA.URL,
				forwardQueryStrings: []string{"foo"},
				forwardHeaders:      []string{"X-Test"},
				timeout:             500,
				client:              http.DefaultClient,
			},
		},
	}

	originalRequest := httptest.NewRequest(http.MethodGet, "http://example.com/test?foo=bar", nil)
	originalRequest.Header.Set("X-Test", "baz")

	results := d.dispatch(route, originalRequest)

	if string(results[0].Body) != "bar-baz" {
		t.Errorf("unexpected result: %q", results[0].Body)
	}
}

func TestDispatcher_Dispatch_PostWithBody(t *testing.T) {
	upstreamA := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Write(body)
	}))
	defer upstreamA.Close()

	d := &defaultDispatcher{
		log:     zap.NewNop(),
		metrics: &testMetrics{},
	}

	route := &Route{
		Upstreams: []Upstream{
			&httpUpstream{
				url:     upstreamA.URL,
				method:  http.MethodPost,
				timeout: 500,
				client:  http.DefaultClient,
			},
		},
	}

	originalRequest := httptest.NewRequest(http.MethodPost, "http://example.com/test", bytes.NewBufferString("hello"))

	results := d.dispatch(route, originalRequest)

	if string(results[0].Body) != "hello" {
		t.Errorf("expected 'hello', got %q", results[0].Body)
	}
}
