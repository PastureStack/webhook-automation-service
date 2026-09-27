package service

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/PastureStack/webhook-automation-service/model"
	"github.com/rancher/go-rancher/v2"
)

func roleBoundaryRequest(method, path, roles, body string) *http.Request {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set(projectAPIHeader, "1a1")
	request.Header.Set(RoleAPIHeader, roles)
	request.Header.Set("Content-Type", "application/json")
	return request
}

func roleBoundaryReceiver() *client.GenericObject {
	object := &client.GenericObject{
		Kind: webhookReceiverKind,
		ResourceData: map[string]interface{}{
			"driver": "scaleService",
			"url":    "http://example.invalid/receiver",
			"config": model.ScaleService{
				ServiceID:   "id",
				ScaleAction: "up",
				ScaleChange: 1,
				Min:         1,
				Max:         4,
			},
		},
	}
	object.Id = "role-boundary-receiver"
	return object
}

func TestCombinedReadonlyRolesBlockReceiverWritesAndHideURL(t *testing.T) {
	for _, roles := range []string{
		"owner,readonly",
		"member, restricted",
		"v1-readonly,owner",
		"member,v1-restricted",
	} {
		t.Run(roles, func(t *testing.T) {
			receiver := roleBoundaryReceiver()
			objects := &mockGenericObject{created: map[string]*client.GenericObject{receiver.Id: receiver}}
			handler := NewRouter(&RouteHandler{ClientFactory: &MockAPIClientFactory{mw: objects}})
			collectionPath := "/v1-webhooks/receivers?projectId=1a1"
			receiverPath := "/v1-webhooks/receivers/" + receiver.Id + "?projectId=1a1"

			response := httptest.NewRecorder()
			handler.ServeHTTP(response, roleBoundaryRequest(http.MethodPost, collectionPath, roles, `{}`))
			if response.Code != http.StatusMethodNotAllowed {
				t.Fatalf("create returned %d, want 405", response.Code)
			}

			response = httptest.NewRecorder()
			handler.ServeHTTP(response, roleBoundaryRequest(http.MethodGet, receiverPath, roles, ""))
			if response.Code != http.StatusOK {
				t.Fatalf("get returned %d, want 200", response.Code)
			}
			var got model.Webhook
			if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			if got.URL != "" {
				t.Fatalf("get exposed receiver URL to roles %q", roles)
			}

			response = httptest.NewRecorder()
			handler.ServeHTTP(response, roleBoundaryRequest(http.MethodGet, collectionPath, roles, ""))
			if response.Code != http.StatusOK {
				t.Fatalf("list returned %d, want 200", response.Code)
			}
			var listed model.WebhookCollection
			if err := json.Unmarshal(response.Body.Bytes(), &listed); err != nil {
				t.Fatal(err)
			}
			if len(listed.Data) != 1 || listed.Data[0].URL != "" {
				t.Fatalf("list exposed receiver URL to roles %q: %#v", roles, listed.Data)
			}

			response = httptest.NewRecorder()
			handler.ServeHTTP(response, roleBoundaryRequest(http.MethodDelete, receiverPath, roles, ""))
			if response.Code != http.StatusMethodNotAllowed {
				t.Fatalf("delete returned %d, want 405", response.Code)
			}
			if objects.created[receiver.Id] != receiver {
				t.Fatal("readonly request changed the receiver")
			}
		})
	}
}

func TestCombinedOwnerMemberRolesCanManageReceiver(t *testing.T) {
	objects := &mockGenericObject{created: map[string]*client.GenericObject{}}
	handler := NewRouter(&RouteHandler{ClientFactory: &MockAPIClientFactory{mw: objects}})
	roles := "owner, member"
	collectionPath := "/v1-webhooks/receivers?projectId=1a1"
	createBody := `{"driver":"scaleService","name":"role-boundary",` +
		`"scaleServiceConfig":{"serviceId":"id","amount":1,"action":"up","min":1,"max":4}}`

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, roleBoundaryRequest(http.MethodPost, collectionPath, roles, createBody))
	if response.Code != http.StatusOK {
		t.Fatalf("owner/member create returned %d, want 200: %s", response.Code, response.Body.String())
	}
	receiverPath := "/v1-webhooks/receivers/1?projectId=1a1"
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, roleBoundaryRequest(http.MethodGet, receiverPath, roles, ""))
	if response.Code != http.StatusOK {
		t.Fatalf("owner/member get returned %d, want 200", response.Code)
	}
	var got model.Webhook
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.URL == "" {
		t.Fatal("owner/member receiver URL was hidden")
	}
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, roleBoundaryRequest(http.MethodDelete, receiverPath, roles, ""))
	if response.Code != http.StatusNoContent {
		t.Fatalf("owner/member delete returned %d, want 204", response.Code)
	}
}

func TestRepeatedRoleHeadersHonorReadonlyRole(t *testing.T) {
	request := roleBoundaryRequest(http.MethodGet, "/", "owner", "")
	request.Header.Add(RoleAPIHeader, "restricted")
	if !hasReadonlyRole(request) {
		t.Fatal("a second role header bypassed the readonly guard")
	}
}
