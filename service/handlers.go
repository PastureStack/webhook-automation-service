package service

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/PastureStack/webhook-automation-service/drivers"
	"github.com/PastureStack/webhook-automation-service/model"
	"github.com/gorilla/mux"
	"github.com/rancher/go-rancher/api"
	v1client "github.com/rancher/go-rancher/client"
	"github.com/rancher/go-rancher/v2"
	"github.com/sirupsen/logrus"
)

const (
	RoleAPIHeader       = "X-API-Roles"
	projectAPIHeader    = "X-API-Project-Id"
	webhookReceiverKind = "webhookReceiver"
)

var readonlyRoles = map[string]bool{
	"readonly":      true,
	"restricted":    true,
	"v1-readonly":   true,
	"v1-restricted": true,
}

func (rh *RouteHandler) ListWebhooks(w http.ResponseWriter, r *http.Request) (int, error) {
	logrus.Info("Listing webhooks")
	apiContext := api.GetApiContext(r)
	projectID, errCode, err := getProjectID(r)
	if err != nil {
		return errCode, err
	}
	apiClient, err := rh.ClientFactory.GetClient(projectID)
	if err != nil {
		return 500, err
	}
	filters := make(map[string]interface{})
	filters["kind"] = webhookReceiverKind
	objs, err := apiClient.GenericObject.List(&client.ListOpts{
		Filters: filters,
	})
	if err != nil {
		return http.StatusInternalServerError, err
	}
	response := []model.Webhook{}
	for _, obj := range objs.Data {
		if obj.Kind != webhookReceiverKind {
			continue
		}
		webhook, err := rh.convertToWebhookGenericObject(obj)
		if err != nil {
			logrus.Warnf("Skipping webhook %s because: %v", obj.Id, err)
			continue
		}

		driver := drivers.GetDriver(webhook.Driver)
		if driver == nil {
			logrus.Warnf("Skipping webhook %s because driver cannot be located", webhook.ID)
			continue
		}
		respWebhook, err := newWebhook(apiContext, webhook.URL, webhook.ID, webhook.Driver, webhook.Name,
			webhook.Config, driver, webhook.State, r)
		if err != nil {
			logrus.Warnf("Skipping webhook %s an error ocurred while producing response: %v", webhook.ID, err)
			continue
		}
		// we will hide the url to prevent readonly and restricted users to access endpoint
		if hasReadonlyRole(r) {
			respWebhook.URL = ""
		}

		response = append(response, *respWebhook)
	}

	collectionURL := apiContext.UrlBuilder.Current() + "?projectId=" + projectID
	apiContext.Write(&model.WebhookCollection{
		Collection: v1client.Collection{
			ResourceType: "receiver",
			Links:        map[string]string{"self": collectionURL}},
		Data: response})
	return 200, nil
}

func (rh *RouteHandler) GetWebhook(w http.ResponseWriter, r *http.Request) (int, error) {
	apiContext := api.GetApiContext(r)
	vars := mux.Vars(r)
	webhookID := vars["id"]
	logrus.Info("Getting the requested webhook")

	projectID, errCode, err := getProjectID(r)
	if err != nil {
		return errCode, err
	}
	apiClient, err := rh.ClientFactory.GetClient(projectID)
	if err != nil {
		return 500, err
	}
	obj, err := apiClient.GenericObject.ById(webhookID)
	if err != nil {
		return 500, err
	}

	if obj == nil || obj.Kind != webhookReceiverKind {
		return 404, fmt.Errorf("Webhook not found")
	}

	webhook, err := rh.convertToWebhookGenericObject(*obj)
	if err != nil {
		return 500, err
	}

	driver := drivers.GetDriver(webhook.Driver)
	if driver == nil {
		return 400, fmt.Errorf("Can't find driver %v", webhook.Driver)
	}

	respWebhook, err := newWebhook(apiContext, webhook.URL, webhook.ID, webhook.Driver, webhook.Name,
		webhook.Config, driver, webhook.State, r)
	if err != nil {
		return 500, fmt.Errorf("create webhook response: %w", err)
	}
	// we will hide the url to prevent readonly and restricted users to access endpoint
	if hasReadonlyRole(r) {
		respWebhook.URL = ""
	}

	apiContext.WriteResource(respWebhook)
	return 200, nil
}

func (rh *RouteHandler) DeleteWebhook(w http.ResponseWriter, r *http.Request) (int, error) {
	if hasReadonlyRole(r) {
		return http.StatusMethodNotAllowed, fmt.Errorf("user doesn't have the access to delete webhook")
	}
	vars := mux.Vars(r)
	webhookID := vars["id"]

	projectID, errCode, err := getProjectID(r)
	if err != nil {
		return errCode, err
	}

	apiClient, err := rh.ClientFactory.GetClient(projectID)
	if err != nil {
		return 500, err
	}
	obj, err := apiClient.GenericObject.ById(webhookID)
	if err != nil {
		return 500, err
	}

	if obj == nil || obj.Kind != webhookReceiverKind {
		return 404, fmt.Errorf("Webhook not found")
	}

	err = apiClient.GenericObject.Delete(obj)
	if err != nil {
		var apiErr *client.ApiError
		if errors.As(err, &apiErr) {
			return apiErr.StatusCode, err
		}
		return http.StatusInternalServerError, err
	}
	return 204, nil
}

func getProjectID(r *http.Request) (string, int, error) {
	projectID := r.URL.Query().Get("projectId")
	if projectID == "" {
		return "", 400, fmt.Errorf("projectId must be supplied as query parameter")
	}
	// The authenticated control-plane proxy supplies the effective project.
	// Never let a different query project select this service's privileged API
	// client after the caller was authorized for the header project.
	authorized := r.Header.Get(projectAPIHeader)
	if authorized == "" {
		return "", http.StatusForbidden, fmt.Errorf("authorized project is required")
	}
	if authorized != projectID {
		return "", http.StatusForbidden, fmt.Errorf("projectId does not match the authorized project")
	}

	return projectID, 0, nil
}

func newWebhook(context *api.ApiContext, url string, id string, driverName string, name string,
	driverConfig interface{}, driver drivers.WebhookDriver, state string, r *http.Request) (*model.Webhook, error) {

	selfLink := context.UrlBuilder.ReferenceByIdLink("receiver", id)
	projectID := r.URL.Query().Get("projectId")
	if projectID != "" {
		selfLink = selfLink + "?projectId=" + projectID
	}

	webhook := &model.Webhook{
		Resource: v1client.Resource{
			Id:    id,
			Type:  "receiver",
			Links: map[string]string{"self": selfLink},
		},
		URL:    url,
		Driver: driverName,
		Name:   name,
		State:  state,
	}
	driver.ConvertToConfigAndSetOnWebhook(driverConfig, webhook)
	return webhook, nil
}

type webhookGenericObject struct {
	ID     string
	Name   string
	State  string
	Links  map[string]string
	Driver string
	URL    string
	Key    string
	Config interface{}
}

func (rh *RouteHandler) convertToWebhookGenericObject(genericObject client.GenericObject) (webhookGenericObject, error) {
	d, ok := genericObject.ResourceData["driver"].(string)
	if !ok {
		return webhookGenericObject{}, fmt.Errorf("Couldn't read webhook data. Bad driver")
	}

	url, ok := genericObject.ResourceData["url"].(string)
	if !ok {
		return webhookGenericObject{}, fmt.Errorf("Couldn't read webhook data. Bad url")
	}

	config, ok := genericObject.ResourceData["config"]
	if !ok {
		return webhookGenericObject{}, fmt.Errorf("Couldn't read webhook data. Bad config on resource")
	}

	return webhookGenericObject{
		Name:   genericObject.Name,
		ID:     genericObject.Id,
		State:  genericObject.State,
		Links:  genericObject.Links,
		Driver: d,
		URL:    url,
		Key:    genericObject.Key,
		Config: config,
	}, nil
}

func (rh *RouteHandler) isUniqueName(webhookName string, projectID string, apiClient *client.RancherClient) (int, error) {
	filters := make(map[string]interface{})
	filters["name"] = webhookName
	filters["kind"] = webhookReceiverKind
	obj, err := apiClient.GenericObject.List(&client.ListOpts{
		Filters: filters,
	})
	if err != nil {
		return 500, err
	}
	for _, existing := range obj.Data {
		if existing.Kind == webhookReceiverKind && existing.Name == webhookName {
			return 400, fmt.Errorf("Cannot have duplicate webhook name, webhook %s already exists", webhookName)
		}
	}
	return 200, nil
}

func hasReadonlyRole(r *http.Request) bool {
	for _, header := range r.Header.Values(RoleAPIHeader) {
		for _, role := range strings.Split(header, ",") {
			if readonlyRoles[strings.ToLower(strings.TrimSpace(role))] {
				return true
			}
		}
	}
	return false
}
