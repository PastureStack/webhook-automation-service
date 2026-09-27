package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"

	v1client "github.com/rancher/go-rancher/client"
)

func receiverSchemaForRole(t *testing.T, handler http.Handler, roles, path string) v1client.Schema {
	t.Helper()
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, roleBoundaryRequest(http.MethodGet, path, roles, ""))
	if response.Code != http.StatusOK {
		t.Fatalf("GET %s for %q returned %d: %s", path, roles, response.Code, response.Body.String())
	}
	if response.Header().Get("Cache-Control") != "private, no-store" || response.Header().Get("Vary") != RoleAPIHeader {
		t.Fatalf("role-specific schema has unsafe cache headers: %v", response.Header())
	}
	var receiver v1client.Schema
	if path == "/v1-webhooks/schemas/receiver" {
		if err := json.Unmarshal(response.Body.Bytes(), &receiver); err != nil {
			t.Fatal(err)
		}
	} else {
		var collection v1client.Schemas
		if err := json.Unmarshal(response.Body.Bytes(), &collection); err != nil {
			t.Fatal(err)
		}
		receiver = collection.Schema("receiver")
	}
	if receiver.Id != "receiver" {
		t.Fatalf("GET %s for %q did not return receiver schema", path, roles)
	}
	return receiver
}

func TestReceiverSchemaMethodsMatchRoleBoundary(t *testing.T) {
	handler := NewRouter(&RouteHandler{})
	for _, test := range []struct {
		roles      string
		collection []string
		resource   []string
	}{
		{"owner", []string{"GET", "POST"}, []string{"GET", "DELETE"}},
		{"member", []string{"GET", "POST"}, []string{"GET", "DELETE"}},
		{"restricted", []string{"GET"}, []string{"GET"}},
		{"readonly", []string{"GET"}, []string{"GET"}},
		{"owner,readonly", []string{"GET"}, []string{"GET"}},
	} {
		for _, path := range []string{"/v1-webhooks/schemas", "/v1-webhooks/schemas/receiver"} {
			t.Run(test.roles+path, func(t *testing.T) {
				got := receiverSchemaForRole(t, handler, test.roles, path)
				if !reflect.DeepEqual(got.CollectionMethods, test.collection) || !reflect.DeepEqual(got.ResourceMethods, test.resource) {
					t.Fatalf("receiver methods for %q: collection=%v resource=%v", test.roles, got.CollectionMethods, got.ResourceMethods)
				}
			})
		}
	}
	base := schemas.Schema("receiver")
	if !reflect.DeepEqual(base.CollectionMethods, []string{"GET", "POST"}) || !reflect.DeepEqual(base.ResourceMethods, []string{"GET", "DELETE"}) {
		t.Fatalf("shared receiver schema was mutated: %+v", base)
	}
}

func TestConcurrentReceiverSchemaRequestsKeepCapabilitiesIsolated(t *testing.T) {
	handler := NewRouter(&RouteHandler{})
	var group sync.WaitGroup
	errors := make(chan error, 200)
	check := func(roles, path string, collection, resource []string) {
		defer group.Done()
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, roleBoundaryRequest(http.MethodGet, path, roles, ""))
		if response.Code != http.StatusOK {
			errors <- fmt.Errorf("%s for %s returned %d", path, roles, response.Code)
			return
		}
		var got v1client.Schema
		if path == "/v1-webhooks/schemas" {
			var all v1client.Schemas
			if err := json.Unmarshal(response.Body.Bytes(), &all); err != nil {
				errors <- err
				return
			}
			got = all.Schema("receiver")
		} else if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
			errors <- err
			return
		}
		if !reflect.DeepEqual(got.CollectionMethods, collection) || !reflect.DeepEqual(got.ResourceMethods, resource) {
			errors <- fmt.Errorf("receiver methods for %s: %v %v", roles, got.CollectionMethods, got.ResourceMethods)
		}
	}
	for i := 0; i < 100; i++ {
		group.Add(2)
		go check("owner", "/v1-webhooks/schemas", []string{"GET", "POST"}, []string{"GET", "DELETE"})
		go check("restricted", "/v1-webhooks/schemas/receiver", []string{"GET"}, []string{"GET"})
	}
	group.Wait()
	close(errors)
	for err := range errors {
		t.Error(err)
	}
}
