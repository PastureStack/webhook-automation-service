package drivers

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/PastureStack/webhook-automation-service/model"
)

func TestForwardPostPreservesRequestWithoutLeakingInboundCredentials(t *testing.T) {
	var received atomic.Bool
	target := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		received.Store(true)
		if request.URL.Path != "/r/projects/1a5/pipeline-server:60080/v1" {
			t.Errorf("unexpected path: %s", request.URL.Path)
		}
		if request.URL.RawQuery != "delivery=fixture" {
			t.Errorf("unexpected query: %s", request.URL.RawQuery)
		}
		accessKey, secretKey, ok := request.BasicAuth()
		if !ok || accessKey != "fixture-access" || secretKey != "fixture-secret" {
			t.Error("control-plane API credentials were not set")
		}
		if request.Header.Get("Cookie") != "" || request.Header.Get("X-Forwarded-Authorization") != "" {
			t.Error("sensitive inbound headers were forwarded")
		}
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Error(err)
		}
		if string(body) != `{"event":"fixture"}` {
			t.Errorf("unexpected body: %s", body)
		}
		response.WriteHeader(http.StatusNoContent)
	}))
	defer target.Close()
	t.Setenv("PASTURESTACK_API_URL", target.URL+"/v2-beta")
	t.Setenv("PASTURESTACK_API_ACCESS_KEY", "fixture-access")
	t.Setenv("PASTURESTACK_API_SECRET_KEY", "fixture-secret")

	request := httptest.NewRequest(http.MethodPost, "/v1-webhooks/endpoint?delivery=fixture", strings.NewReader(`{"event":"fixture"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer inbound-secret")
	request.Header.Set("X-Forwarded-Authorization", "inbound-secret")
	request.Header.Set("Cookie", "session=inbound-secret")
	code, err := (&ForwardPostDriver{}).Execute(forwardPostFixture(), nil, request)
	if err != nil || code != http.StatusNoContent {
		t.Fatalf("forward failed: code=%d err=%v", code, err)
	}
	if !received.Load() {
		t.Fatal("target did not receive request")
	}
}

func TestForwardPostHandlesNoQueryAndRejectsOversizedBodies(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.RawQuery != "" {
			t.Errorf("unexpected query: %s", request.URL.RawQuery)
		}
		response.WriteHeader(http.StatusOK)
	}))
	defer target.Close()
	t.Setenv("PASTURESTACK_API_URL", target.URL+"/v2-beta")

	request := httptest.NewRequest(http.MethodPost, "/v1-webhooks/endpoint", bytes.NewReader(nil))
	code, err := (&ForwardPostDriver{}).Execute(forwardPostFixture(), nil, request)
	if err != nil || code != http.StatusOK {
		t.Fatalf("no-query request failed: code=%d err=%v", code, err)
	}

	request = httptest.NewRequest(http.MethodPost, "/v1-webhooks/endpoint", strings.NewReader(strings.Repeat("x", maximumWebhookBodyBytes+1)))
	code, err = (&ForwardPostDriver{}).Execute(forwardPostFixture(), nil, request)
	if err == nil || code != http.StatusBadRequest {
		t.Fatalf("oversized body accepted: code=%d err=%v", code, err)
	}
}

func TestForwardPostRefusesRedirects(t *testing.T) {
	var redirected atomic.Bool
	destination := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		redirected.Store(true)
		response.WriteHeader(http.StatusNoContent)
	}))
	defer destination.Close()
	redirector := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		http.Redirect(response, request, destination.URL, http.StatusTemporaryRedirect)
	}))
	defer redirector.Close()
	t.Setenv("PASTURESTACK_API_URL", redirector.URL+"/v2-beta")

	request := httptest.NewRequest(http.MethodPost, "/v1-webhooks/endpoint", bytes.NewReader(nil))
	code, err := (&ForwardPostDriver{}).Execute(forwardPostFixture(), nil, request)
	if err != nil || code != http.StatusTemporaryRedirect {
		t.Fatalf("unexpected redirect result: code=%d err=%v", code, err)
	}
	if redirected.Load() {
		t.Fatal("redirect was followed")
	}
}

func TestForwardPostRejectsAmbiguousControlPlaneOriginsBeforeNetwork(t *testing.T) {
	var received atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		received.Add(1)
		response.WriteHeader(http.StatusNoContent)
	}))
	defer target.Close()

	tests := []string{
		"file:///var/run/platform.sock",
		strings.Replace(target.URL, "://", "://access:secret@", 1),
		target.URL + "/v2-beta?redirect=https://foreign.example",
		target.URL + "/v2-beta#foreign",
	}
	for _, apiURL := range tests {
		t.Run(apiURL, func(t *testing.T) {
			t.Setenv("PASTURESTACK_API_URL", apiURL)
			request := httptest.NewRequest(http.MethodPost, "/v1-webhooks/endpoint", bytes.NewReader(nil))
			code, err := (&ForwardPostDriver{}).Execute(forwardPostFixture(), nil, request)
			if err == nil || code != http.StatusInternalServerError {
				t.Fatalf("ambiguous API origin accepted: code=%d err=%v", code, err)
			}
		})
	}
	if received.Load() != 0 {
		t.Fatalf("rejected origins reached the network %d times", received.Load())
	}
}

func forwardPostFixture() model.ForwardPost {
	return model.ForwardPost{
		ProjectID:   "1a5",
		ServiceName: "pipeline-server",
		Port:        "60080",
		Path:        "/v1",
	}
}
