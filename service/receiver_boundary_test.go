package service

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rancher/go-rancher/v2"
)

func TestReceiverDirectIDRejectsOtherGenericObjectKinds(t *testing.T) {
	mock := r.ClientFactory.(*MockAPIClientFactory).mw
	id := "wrong-kind-fixture"
	foreign := &client.GenericObject{
		Kind: "otherGenericObject",
		ResourceData: map[string]interface{}{
			"driver": "scaleService", "url": "http://example.invalid",
			"config": map[string]interface{}{},
		},
	}
	foreign.Id = id
	mock.created[id] = foreign
	t.Cleanup(func() { delete(mock.created, id) })

	for _, method := range []string{http.MethodGet, http.MethodDelete} {
		request := httptest.NewRequest(method,
			"/v1-webhooks/receivers/"+id+"?projectId=1a1", nil)
		request.Header.Set(projectAPIHeader, "1a1")
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != http.StatusNotFound {
			t.Fatalf("%s wrong-kind ID returned %d, want 404", method, response.Code)
		}
		if mock.created[id] != foreign {
			t.Fatalf("%s changed an unrelated generic object", method)
		}
	}
}

func TestReceiverListNeverSerializesOtherGenericObjectKinds(t *testing.T) {
	mock := r.ClientFactory.(*MockAPIClientFactory).mw
	id := "wrong-kind-list-fixture"
	foreign := &client.GenericObject{
		Kind: "otherGenericObject",
		ResourceData: map[string]interface{}{
			"driver": "scaleService", "url": "http://example.invalid",
			"config": map[string]interface{}{},
		},
	}
	foreign.Id = id
	mock.created[id] = foreign
	t.Cleanup(func() { delete(mock.created, id) })
	request := httptest.NewRequest(http.MethodGet, "/v1-webhooks/receivers?projectId=1a1", nil)
	request.Header.Set(projectAPIHeader, "1a1")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("receiver list returned %d, want 200", response.Code)
	}
	var payload struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	for _, item := range payload.Data {
		if item.ID == id {
			t.Fatal("receiver list serialized a foreign generic object")
		}
	}
}

func TestReceiverNameUniquenessIgnoresOtherGenericObjectKinds(t *testing.T) {
	mock := r.ClientFactory.(*MockAPIClientFactory).mw
	id := "wrong-kind-name-fixture"
	foreign := &client.GenericObject{Kind: "otherGenericObject", Name: "shared-name-fixture"}
	foreign.Id = id
	mock.created[id] = foreign
	t.Cleanup(func() { delete(mock.created, id) })
	apiClient, err := r.ClientFactory.GetClient("1a1")
	if err != nil {
		t.Fatal(err)
	}
	code, err := r.isUniqueName(foreign.Name, "1a1", apiClient)
	if err != nil || code != http.StatusOK {
		t.Fatalf("foreign generic object blocked a receiver name: code=%d err=%v", code, err)
	}
	receiver := &client.GenericObject{Kind: webhookReceiverKind, Name: foreign.Name}
	receiver.Id = "same-kind-name-fixture"
	mock.created[receiver.Id] = receiver
	t.Cleanup(func() { delete(mock.created, receiver.Id) })
	code, err = r.isUniqueName(receiver.Name, "1a1", apiClient)
	if err == nil || code != http.StatusBadRequest {
		t.Fatalf("duplicate receiver name was accepted: code=%d err=%v", code, err)
	}
}

func TestReceiverRejectsMismatchedAuthorizedProject(t *testing.T) {
	mock := r.ClientFactory.(*MockAPIClientFactory).mw
	id := "project-mismatch-fixture"
	receiver := &client.GenericObject{Kind: webhookReceiverKind}
	receiver.Id = id
	mock.created[id] = receiver
	t.Cleanup(func() { delete(mock.created, id) })

	for _, method := range []string{http.MethodGet, http.MethodDelete} {
		request := httptest.NewRequest(method,
			"/v1-webhooks/receivers/"+id+"?projectId=1a2", nil)
		request.Header.Set(projectAPIHeader, "1a1")
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != http.StatusForbidden {
			t.Fatalf("%s mismatched project returned %d, want 403", method, response.Code)
		}
		if mock.created[id] != receiver {
			t.Fatalf("%s changed a receiver from another project", method)
		}
	}
}

func TestReceiverRejectsMissingAuthorizedProject(t *testing.T) {
	mock := r.ClientFactory.(*MockAPIClientFactory).mw
	id := "missing-project-header-fixture"
	receiver := &client.GenericObject{Kind: webhookReceiverKind}
	receiver.Id = id
	mock.created[id] = receiver
	t.Cleanup(func() { delete(mock.created, id) })

	for _, method := range []string{http.MethodGet, http.MethodDelete} {
		request := httptest.NewRequest(method,
			"/v1-webhooks/receivers/"+id+"?projectId=1a1", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != http.StatusForbidden {
			t.Fatalf("%s missing trusted project returned %d, want 403", method, response.Code)
		}
		if mock.created[id] != receiver {
			t.Fatalf("%s changed a receiver without trusted project", method)
		}
	}
}
