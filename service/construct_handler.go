package service

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"reflect"
	"strings"

	"github.com/PastureStack/webhook-automation-service/drivers"
	"github.com/PastureStack/webhook-automation-service/model"
	"github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
	"github.com/rancher/go-rancher/api"
	"github.com/rancher/go-rancher/v2"
)

const maximumConfigurationBodyBytes = 1 << 20

func (rh *RouteHandler) ConstructPayload(w http.ResponseWriter, r *http.Request) (int, error) {
	if readonlyRoles[getRoles(r)] {
		return http.StatusMethodNotAllowed, fmt.Errorf("user doesn't have the access to create webhook")
	}
	apiContext := api.GetApiContext(r)

	wh := &model.Webhook{}
	logrus.Infof("Construct Payload")
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return http.StatusBadRequest, fmt.Errorf("Content-Type must be application/json")
	}
	r.Body = http.MaxBytesReader(w, r.Body, maximumConfigurationBodyBytes)
	requestBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("read request body: %w", err)
	}

	projectID, errCode, err := getProjectID(r)
	if err != nil {
		return errCode, err
	}

	if err := json.Unmarshal(requestBytes, &wh); err != nil {
		return 400, errors.Wrap(err, "Bad request body")
	}

	if wh.Name == "" {
		return 400, fmt.Errorf("Name not provided")
	}

	if wh.Driver == "" {
		return 400, fmt.Errorf("Driver not provided")
	}

	driver := drivers.GetDriver(wh.Driver)
	if driver == nil {
		return 400, fmt.Errorf("Invalid driver %v", wh.Driver)
	}

	driverConfig := getDriverConfig(wh)
	if driverConfig == nil {
		return 400, fmt.Errorf("Invalid driver %v", wh.Driver)
	}

	apiClient, err := rh.ClientFactory.GetClient(projectID)
	if err != nil {
		return 500, err
	}

	code, err := rh.isUniqueName(wh.Name, projectID, apiClient)
	if err != nil {
		return code, err
	}

	code, err = driver.ValidatePayload(driverConfig, apiClient)
	if err != nil {
		return code, err
	}

	uuid, err := newWebhookKey()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("generate webhook key: %w", err)
	}

	url := apiContext.UrlBuilder.Version("v1-webhooks")
	url = url + "/endpoint?key=" + uuid + "&projectId=" + projectID

	//saveWebhook needs only user fields
	webhook, err := saveWebhook(uuid, wh.Name, wh.Driver, url, driverConfig, apiClient)
	if err != nil {
		return 500, err
	}

	//needs only user fields
	whResponse, err := newWebhook(apiContext, url, webhook.Id, wh.Driver, wh.Name, driverConfig, driver,
		webhook.State, r)
	if err != nil {
		return 500, errors.Wrap(err, "Unable to create webhook response")
	}
	apiContext.WriteResource(whResponse)
	return 200, nil
}

func newWebhookKey() (string, error) {
	random := make([]byte, 30)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(random), nil
}

func saveWebhook(uuid string, name string, driver string, url string, config interface{}, apiClient *client.RancherClient) (*client.GenericObject, error) {
	resourceData := map[string]interface{}{
		"url":    url,
		"driver": driver,
		"config": config,
	}
	obj, err := apiClient.GenericObject.Create(&client.GenericObject{
		Name:         name,
		Key:          uuid,
		ResourceData: resourceData,
		Kind:         "webhookReceiver",
	})

	if err != nil {
		return &client.GenericObject{}, fmt.Errorf("Failed to create webhook: %v", err)
	}
	return obj, nil
}

func getDriverConfig(wh *model.Webhook) interface{} {
	r := reflect.ValueOf(wh)
	fieldName := strings.Title(wh.Driver) + "Config"
	f := reflect.Indirect(r).FieldByName(fieldName)
	return f.Interface()
}
