package service

import (
	"fmt"
	"net/http"

	"github.com/PastureStack/webhook-automation-service/drivers"
	"github.com/rancher/go-rancher/v2"
)

func (rh *RouteHandler) Execute(w http.ResponseWriter, r *http.Request) (int, error) {
	jwtSigned := r.FormValue("token")
	if jwtSigned != "" {
		code, err := rh.ExecuteWithJwt(jwtSigned, r)
		if err != nil {
			return code, err
		}
		return 200, nil
	}
	uuid := r.FormValue("key")
	if uuid == "" {
		return 400, fmt.Errorf("Invalid execute url, should have 'token' or 'key'")
	}

	projectID := r.FormValue("projectId")
	if projectID == "" {
		return 400, fmt.Errorf("Invalid execute url, url must contain projectId")
	}

	code, err := rh.ExecuteWithKey(uuid, projectID, r)
	if err != nil {
		return code, err
	}

	return 200, nil
}

func (rh *RouteHandler) ExecuteWithJwt(jwtSigned string, request *http.Request) (int, error) {
	claims, err := verifyJWT(jwtSigned, rh.PublicKey)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("invalid token: %v", err)
	}
	driverID, ok := claims["driver"].(string)
	if !ok || driverID == "" {
		return http.StatusBadRequest, fmt.Errorf("driver not found after decode")
	}

	driver := drivers.GetDriver(driverID)
	if driver == nil {
		return http.StatusBadRequest, fmt.Errorf("driver %s is not registered", driverID)
	}

	projectID, ok := claims["projectId"].(string)
	if !ok || projectID == "" {
		return http.StatusBadRequest, fmt.Errorf("projectId not provided by server")
	}

	uuid, ok := claims["uuid"].(string)
	if !ok || uuid == "" {
		return http.StatusBadRequest, fmt.Errorf("uuid not found after decode")
	}

	apiClient, err := rh.ClientFactory.GetClient(projectID)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	code, err := validateWebhook(uuid, apiClient)
	if err != nil {
		return code, err
	}

	responseCode, err := driver.Execute(claims["config"], apiClient, request)
	if err != nil {
		return responseCode, fmt.Errorf("error %v in executing driver for %s", err, driverID)
	}
	return http.StatusOK, nil
}

func (rh *RouteHandler) ExecuteWithKey(uuid string, projectID string, request *http.Request) (int, error) {
	apiClient, err := rh.ClientFactory.GetClient(projectID)
	if err != nil {
		return 500, err
	}

	filters := make(map[string]interface{})
	filters["key"] = uuid
	goCollection, err := apiClient.GenericObject.List(&client.ListOpts{
		Filters: filters,
	})
	if err != nil {
		return 500, fmt.Errorf("Error %v filtering genericObjects by key", err)
	}

	if len(goCollection.Data) == 0 {
		return 403, fmt.Errorf("Requested webhook has been revoked/does not exist for this account")
	}

	resourceData := goCollection.Data[0].ResourceData
	driverID, ok := resourceData["driver"].(string)
	if !ok {
		return 400, fmt.Errorf("No driver provided")
	}

	driver := drivers.GetDriver(driverID)
	if driver == nil {
		return 400, fmt.Errorf("Driver %s is not registered", driverID)
	}

	driverConfig, ok := resourceData["config"]
	if !ok {
		return 400, fmt.Errorf("Driver config not found")
	}

	responseCode, err := driver.Execute(driverConfig, apiClient, request)
	if err != nil {
		return responseCode, fmt.Errorf("Error %v in executing driver for %s", err, driverID)
	}

	return 200, nil
}

func validateWebhook(uuid string, apiClient *client.RancherClient) (int, error) {
	filters := make(map[string]interface{})
	filters["key"] = uuid
	webhookCollection, err := apiClient.GenericObject.List(&client.ListOpts{
		Filters: filters,
	})
	if err != nil {
		return 500, err
	}
	if len(webhookCollection.Data) > 0 {
		return 0, nil
	}
	return 403, fmt.Errorf("Requested webhook has been revoked")
}
