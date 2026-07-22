package service

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestConstructPayloadAcceptsJSONCharset(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/v1-webhooks/receivers", strings.NewReader(`{}`))
	request.Header.Set("Content-Type", "application/json; charset=utf-8")
	response := httptest.NewRecorder()
	code, err := r.ConstructPayload(response, request)
	if err == nil || code != http.StatusBadRequest {
		t.Fatalf("unexpected result: code=%d err=%v", code, err)
	}
	if strings.Contains(err.Error(), "Content-Type") {
		t.Fatalf("valid JSON media type was rejected: %v", err)
	}
}

func TestConstructPayloadRejectsOversizedBody(t *testing.T) {
	request := httptest.NewRequest(
		http.MethodPost,
		"/v1-webhooks/receivers?projectId=1a1",
		strings.NewReader(strings.Repeat("x", maximumConfigurationBodyBytes+1)),
	)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	code, err := r.ConstructPayload(response, request)
	if err == nil || code != http.StatusBadRequest {
		t.Fatalf("oversized body accepted: code=%d err=%v", code, err)
	}
}
