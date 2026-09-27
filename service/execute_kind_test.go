package service

import (
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/PastureStack/webhook-automation-service/model"
	"github.com/rancher/go-rancher/v2"
)

// The mock honors the key filter but intentionally ignores kind, so the
// receiver boundary is checked even if the API returns other object kinds.
type executeKindObjectList struct {
	client.GenericObjectOperations
	objects []client.GenericObject
	filters map[string]interface{}
}

func (m *executeKindObjectList) List(opts *client.ListOpts) (*client.GenericObjectCollection, error) {
	m.filters = opts.Filters
	var matched []client.GenericObject
	for _, object := range m.objects {
		if object.Key == opts.Filters["key"] {
			matched = append(matched, object)
		}
	}
	return &client.GenericObjectCollection{Data: matched}, nil
}

type executeKindClientFactory struct {
	objects *executeKindObjectList
}

func (f *executeKindClientFactory) GetClient(string) (*client.RancherClient, error) {
	return &client.RancherClient{GenericObject: f.objects}, nil
}

func newExecuteKindHandler(objects ...client.GenericObject) (*RouteHandler, *executeKindObjectList) {
	list := &executeKindObjectList{objects: objects}
	return &RouteHandler{ClientFactory: &executeKindClientFactory{objects: list}}, list
}

func executeKindReceiver(kind string) client.GenericObject {
	return client.GenericObject{
		Key:  "receiver-key",
		Kind: kind,
		ResourceData: map[string]interface{}{
			"driver": "scaleService",
			"config": model.ScaleService{
				ServiceID:   "id",
				ScaleAction: "up",
				ScaleChange: 1,
			},
		},
	}
}

func TestExecuteWithKeyRejectsForeignGenericObject(t *testing.T) {
	handler, list := newExecuteKindHandler(executeKindReceiver("otherGenericObject"))
	request := httptest.NewRequest(http.MethodPost, "/v1-webhooks/endpoint", nil)
	code, err := handler.ExecuteWithKey("receiver-key", "1a1", request)
	if code != http.StatusForbidden || err == nil {
		t.Fatalf("foreign object returned (%d, %v), want 403", code, err)
	}
	if list.filters["key"] != "receiver-key" || list.filters["kind"] != webhookReceiverKind {
		t.Fatalf("list filters = %#v, want receiver key and kind", list.filters)
	}
}

func TestExecuteWithSignedJWTRejectsForeignGenericObject(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	handler, list := newExecuteKindHandler(executeKindReceiver("otherGenericObject"))
	handler.PublicKey = &privateKey.PublicKey
	token := signTestToken(t, privateKey, "RS256", map[string]interface{}{
		"driver":    "scaleService",
		"projectId": "1a1",
		"uuid":      "receiver-key",
		"exp":       time.Now().Add(time.Minute).Unix(),
	})
	request := httptest.NewRequest(http.MethodPost, "/v1-webhooks/endpoint?token="+token, nil)
	code, err := handler.Execute(httptest.NewRecorder(), request)
	if code != http.StatusForbidden || err == nil {
		t.Fatalf("signed JWT for foreign object returned (%d, %v), want 403", code, err)
	}
	if list.filters["key"] != "receiver-key" || list.filters["kind"] != webhookReceiverKind {
		t.Fatalf("list filters = %#v, want receiver key and kind", list.filters)
	}
}

func TestExecuteWithKeyUsesReceiverAfterForeignGenericObject(t *testing.T) {
	foreign := executeKindReceiver("otherGenericObject")
	foreign.ResourceData["driver"] = "unregistered"
	handler, _ := newExecuteKindHandler(
		foreign,
		executeKindReceiver(webhookReceiverKind),
	)
	request := httptest.NewRequest(http.MethodPost, "/v1-webhooks/endpoint", nil)
	code, err := handler.ExecuteWithKey("receiver-key", "1a1", request)
	if code != http.StatusOK || err != nil {
		t.Fatalf("valid receiver returned (%d, %v), want 200", code, err)
	}
}

func TestValidateWebhookRequiresReceiverKind(t *testing.T) {
	for _, test := range []struct {
		name    string
		objects []client.GenericObject
		want    int
	}{
		{"foreign only", []client.GenericObject{executeKindReceiver("otherGenericObject")}, http.StatusForbidden},
		{"valid receiver", []client.GenericObject{executeKindReceiver(webhookReceiverKind)}, 0},
		{"foreign then receiver", []client.GenericObject{executeKindReceiver("otherGenericObject"), executeKindReceiver(webhookReceiverKind)}, 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			list := &executeKindObjectList{objects: test.objects}
			apiClient := &client.RancherClient{GenericObject: list}
			code, err := validateWebhook("receiver-key", apiClient)
			if code != test.want || (err == nil) != (test.want == 0) {
				t.Fatalf("validateWebhook returned (%d, %v), want status %d", code, err, test.want)
			}
			if list.filters["key"] != "receiver-key" || list.filters["kind"] != webhookReceiverKind {
				t.Fatalf("list filters = %#v, want receiver key and kind", list.filters)
			}
		})
	}
}
