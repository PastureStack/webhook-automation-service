package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-viper/mapstructure/v2"
	v1client "github.com/rancher/go-rancher/client"
	"github.com/rancher/go-rancher/v2"
	"github.com/sirupsen/logrus"

	"github.com/PastureStack/webhook-automation-service/drivers"
	"github.com/PastureStack/webhook-automation-service/model"
)

func requireEqual[T comparable](t *testing.T, field string, actual, expected T) {
	t.Helper()
	if actual != expected {
		t.Fatalf("%s: got %v, want %v", field, actual, expected)
	}
}

func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func requireWebhook(t *testing.T, webhook *model.Webhook) {
	t.Helper()
	requireEqual(t, "name", webhook.Name, "wh-name")
	requireEqual(t, "driver", webhook.Driver, "forwardPost")
	requireEqual(t, "id", webhook.Id, "1")
	if webhook.URL == "" {
		t.Fatal("webhook URL is empty")
	}
	requireEqual(t, "project ID", webhook.ForwardPostConfig.ProjectID, "1a5")
	requireEqual(t, "service name", webhook.ForwardPostConfig.ServiceName, "pipeline-server")
	requireEqual(t, "port", webhook.ForwardPostConfig.Port, "60080")
	requireEqual(t, "path", webhook.ForwardPostConfig.Path, "/v1")
}

func requireSelfLink(t *testing.T, self string) {
	t.Helper()
	if !strings.HasSuffix(self, "/v1-webhooks/receivers/1?projectId=1a1") {
		t.Fatalf("unexpected self URL: %s", self)
	}
}

func TestCreateUpdateExecuteListAndDelete(t *testing.T) {
	// Test creating a webhook
	constructURL := fmt.Sprintf("%s/v1-webhooks/receivers?projectId=1a1", server.URL)
	jsonStr := []byte(`{"driver":"forwardPost","name":"wh-name",
		"forwardPostConfig": {"projectId": "1a5","serviceName": "pipeline-server", "port": "60080", "path": "/v1"}}`)
	request, err := http.NewRequest("POST", constructURL, bytes.NewBuffer(jsonStr))
	requireNoError(t, err)

	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler := HandleError(schemas, r.ConstructPayload)
	handler.ServeHTTP(response, request)
	requireEqual(t, "create status", response.Code, http.StatusOK)

	resp, err := io.ReadAll(response.Body)
	requireNoError(t, err)

	wh := &model.Webhook{}
	err = json.Unmarshal(resp, wh)
	requireNoError(t, err)
	requireWebhook(t, wh)
	requireSelfLink(t, wh.Links["self"])

	// Test getting the created webhook by id
	byID := fmt.Sprintf("%s/v1-webhooks/receivers/1?projectId=1a1", server.URL)
	request, err = http.NewRequest("GET", byID, nil)
	requireNoError(t, err)

	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	requireEqual(t, "get status", response.Code, http.StatusOK)

	resp, err = io.ReadAll(response.Body)
	requireNoError(t, err)

	wh = &model.Webhook{}
	err = json.Unmarshal(resp, wh)
	requireNoError(t, err)
	requireWebhook(t, wh)

	// Test executing the webhook
	url := wh.URL
	requestExecute, err := http.NewRequest("POST", url, nil)
	requireNoError(t, err)
	response = httptest.NewRecorder()
	handler = HandleError(schemas, r.Execute)
	handler.ServeHTTP(response, requestExecute)
	requireEqual(t, "execute status", response.Code, http.StatusOK)

	//List webhooks
	requestList, err := http.NewRequest("GET", constructURL, nil)
	requireNoError(t, err)

	requestList.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	router.ServeHTTP(response, requestList)
	requireEqual(t, "list status", response.Code, http.StatusOK)

	resp, err = io.ReadAll(response.Body)
	requireNoError(t, err)

	whCollection := &model.WebhookCollection{}
	err = json.Unmarshal(resp, whCollection)
	requireNoError(t, err)
	requireEqual(t, "webhook count", len(whCollection.Data), 1)

	wh = &whCollection.Data[0]
	requireWebhook(t, wh)
	requireSelfLink(t, wh.Links["self"])

	//Delete
	request, err = http.NewRequest("DELETE", byID, nil)
	requireNoError(t, err)

	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	requireEqual(t, "delete status", response.Code, http.StatusNoContent)
}

type MockForwardPostDriver struct {
	expectedConfig model.ForwardPost
}

func (s *MockForwardPostDriver) Execute(conf interface{}, apiClient *client.RancherClient, request *http.Request) (int, error) {
	config := &model.ForwardPost{}

	if err := mapstructure.Decode(conf, config); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("Couldn't unmarshal config: %v", err)
	}

	if config.ServiceName != s.expectedConfig.ServiceName {
		return 500, fmt.Errorf("Tag. Expected %v, Actual %v", s.expectedConfig.ServiceName, config.ServiceName)
	}
	logrus.Infof("Execute of mock upgradeService driver")
	return 0, nil
}

func (s *MockForwardPostDriver) ValidatePayload(conf interface{}, apiClient *client.RancherClient) (int, error) {
	if _, ok := conf.(model.ForwardPost); !ok {
		return http.StatusInternalServerError, fmt.Errorf("Can't process config")
	}

	logrus.Infof("Validate payload of mock forwardPost driver")
	return 0, nil
}

func (s *MockForwardPostDriver) GetDriverConfigResource() interface{} {
	return model.ForwardPost{}
}

func (s *MockForwardPostDriver) CustomizeSchema(schema *v1client.Schema) *v1client.Schema {
	return schema
}

func (s *MockForwardPostDriver) ConvertToConfigAndSetOnWebhook(conf interface{}, webhook *model.Webhook) error {
	ss := &drivers.ForwardPostDriver{}
	return ss.ConvertToConfigAndSetOnWebhook(conf, webhook)
}
