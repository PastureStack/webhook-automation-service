package originhttp

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type recordingTransport struct {
	requests atomic.Int32
	response *http.Response
	err      error
}

func (r *recordingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	r.requests.Add(1)
	return r.response, r.err
}

func parsedURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	value, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func TestClientAllowsOnlyApprovedOrigin(t *testing.T) {
	transport := &recordingTransport{response: &http.Response{
		StatusCode: http.StatusNoContent,
		Body:       http.NoBody,
		Header:     make(http.Header),
	}}
	origin := parsedURL(t, "https://platform.example/v2-beta")
	client, err := New(&http.Client{Transport: transport, Timeout: time.Second}, origin, origin)
	if err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://platform.example/r/projects/1a1/service:80/hook", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.SetBasicAuth("access", "secret")
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if transport.requests.Load() != 1 {
		t.Fatalf("approved transport received %d requests", transport.requests.Load())
	}

	foreign, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://foreign.example/hook", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Do(foreign); err == nil || !strings.Contains(err.Error(), "not allowed") {
		t.Fatalf("expected foreign origin rejection, got %v", err)
	}
	if transport.requests.Load() != 1 {
		t.Fatal("rejected request reached the network transport")
	}
}

func TestClientRejectsHostOverrideAndUnsupportedMethod(t *testing.T) {
	transport := &recordingTransport{}
	origin := parsedURL(t, "https://platform.example")
	client, err := New(&http.Client{Transport: transport}, origin, origin)
	if err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequest(http.MethodPost, "https://platform.example/hook", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Host = "foreign.example"
	if _, err := client.Do(request); err == nil || !strings.Contains(err.Error(), "Host override") {
		t.Fatalf("expected Host override rejection, got %v", err)
	}
	request.Host = ""
	request.Method = http.MethodGet
	if _, err := client.Do(request); err == nil || !strings.Contains(err.Error(), "not allowed") {
		t.Fatalf("expected method rejection, got %v", err)
	}
}

func TestClientAppliesTimeoutAtTransportBoundary(t *testing.T) {
	transport := roundTripperFunc(func(request *http.Request) (*http.Response, error) {
		<-request.Context().Done()
		return nil, request.Context().Err()
	})
	origin := parsedURL(t, "http://127.0.0.1:8080")
	client, err := New(&http.Client{Transport: transport, Timeout: 5 * time.Millisecond}, origin, origin)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8080/hook", nil)
	if _, err := client.Do(request); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected transport timeout, got %v", err)
	}
}

func TestNewDisablesAmbientProxy(t *testing.T) {
	transport := &http.Transport{Proxy: http.ProxyFromEnvironment}
	origin := parsedURL(t, "https://platform.example")
	client, err := New(&http.Client{Transport: transport}, origin, origin)
	if err != nil {
		t.Fatal(err)
	}
	bound, ok := client.transport.(*http.Transport)
	if !ok || bound.Proxy != nil {
		t.Fatal("destination-bound client retained ambient proxy behavior")
	}
	if transport.Proxy == nil {
		t.Fatal("constructor mutated the caller's transport")
	}
}

func TestNewRejectsInvalidPolicy(t *testing.T) {
	if _, err := New(nil, nil); err == nil {
		t.Fatal("expected empty allowlist to fail")
	}
	foreign := parsedURL(t, "https://foreign.example")
	allowed := parsedURL(t, "https://allowed.example")
	if _, err := New(nil, foreign, allowed); err == nil {
		t.Fatal("expected foreign credential origin to fail")
	}
	if _, err := New(nil, nil, parsedURL(t, "file:///tmp/socket")); err == nil {
		t.Fatal("expected non-HTTP origin to fail")
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (function roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}
