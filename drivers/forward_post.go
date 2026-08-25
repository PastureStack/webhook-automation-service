package drivers

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/PastureStack/webhook-automation-service/config"
	"github.com/PastureStack/webhook-automation-service/internal/originhttp"
	"github.com/PastureStack/webhook-automation-service/model"
	"github.com/go-viper/mapstructure/v2"
	v1client "github.com/rancher/go-rancher/client"
	"github.com/rancher/go-rancher/v2"
)

const (
	maximumWebhookBodyBytes = 1 << 20
	maximumForwardResponse  = 1 << 20
	forwardRequestTimeout   = 15 * time.Second
)

type ForwardPostDriver struct {
}

func (s *ForwardPostDriver) ValidatePayload(conf interface{}, apiClient *client.RancherClient) (int, error) {
	forwardConfig, ok := conf.(model.ForwardPost)
	if !ok {
		return http.StatusInternalServerError, fmt.Errorf("can't process config")
	}
	if err := validateForwardConfig(&forwardConfig); err != nil {
		return http.StatusBadRequest, err
	}
	return http.StatusOK, nil
}

func (s *ForwardPostDriver) Execute(conf interface{}, apiClient *client.RancherClient, request *http.Request) (int, error) {
	payload, err := readBoundedBody(request.Body, maximumWebhookBodyBytes)
	if err != nil {
		return http.StatusBadRequest, err
	}

	forwardConfig := &model.ForwardPost{}
	if err = mapstructure.Decode(conf, forwardConfig); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("couldn't unmarshal config: %w", err)
	}
	if err := validateForwardConfig(forwardConfig); err != nil {
		return http.StatusBadRequest, err
	}

	apiConfig := config.GetConfig()
	destination, err := forwardDestination(apiConfig.APIURL, forwardConfig, request.URL.RawQuery)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	hopRequest, err := http.NewRequestWithContext(request.Context(), http.MethodPost, destination.String(), bytes.NewReader(payload))
	if err != nil {
		return http.StatusInternalServerError, err
	}
	copyForwardHeaders(hopRequest.Header, request.Header)
	hopRequest.SetBasicAuth(apiConfig.AccessKey, apiConfig.SecretKey)

	httpClient, err := originhttp.New(&http.Client{Timeout: forwardRequestTimeout}, destination, destination)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("create destination-bound HTTP client: %w", err)
	}
	response, err := httpClient.Do(hopRequest)
	if err != nil {
		return http.StatusBadGateway, fmt.Errorf("forward request failed: %w", err)
	}
	defer response.Body.Close()
	responseBody, err := readBoundedBody(response.Body, maximumForwardResponse)
	if err != nil {
		return http.StatusBadGateway, fmt.Errorf("read forward response: %w", err)
	}
	if response.StatusCode >= http.StatusBadRequest {
		return response.StatusCode, fmt.Errorf("forwarded request returned HTTP %d", response.StatusCode)
	}
	_ = responseBody
	return response.StatusCode, nil
}

func forwardDestination(apiURL string, forwardConfig *model.ForwardPost, rawQuery string) (*url.URL, error) {
	destination, err := url.Parse(apiURL)
	if err != nil || destination.Host == "" || destination.Hostname() == "" ||
		(destination.Scheme != "http" && destination.Scheme != "https") ||
		destination.User != nil || destination.Opaque != "" || destination.RawQuery != "" || destination.Fragment != "" {
		return nil, fmt.Errorf("control-plane API URL is invalid")
	}
	destination.Path = fmt.Sprintf(
		"/r/projects/%s/%s:%s%s",
		url.PathEscape(forwardConfig.ProjectID),
		url.PathEscape(forwardConfig.ServiceName),
		forwardConfig.Port,
		forwardConfig.Path,
	)
	destination.RawPath = ""
	destination.RawQuery = rawQuery
	destination.Fragment = ""
	return destination, nil
}

func validateForwardConfig(forwardConfig *model.ForwardPost) error {
	if forwardConfig.ProjectID == "" || strings.ContainsAny(forwardConfig.ProjectID, "/?#") {
		return fmt.Errorf("projectId is invalid")
	}
	if forwardConfig.ServiceName == "" || strings.ContainsAny(forwardConfig.ServiceName, "/?#") {
		return fmt.Errorf("serviceName is invalid")
	}
	port, err := strconv.Atoi(forwardConfig.Port)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("port is invalid")
	}
	if forwardConfig.Path == "" || !strings.HasPrefix(forwardConfig.Path, "/") || strings.ContainsAny(forwardConfig.Path, "?#") {
		return fmt.Errorf("path is invalid")
	}
	return nil
}

func copyForwardHeaders(destination, source http.Header) {
	for name, values := range source {
		switch strings.ToLower(name) {
		case "authorization", "proxy-authorization", "x-forwarded-authorization", "x-original-authorization",
			"cookie", "set-cookie", "connection",
			"keep-alive", "proxy-authenticate", "te", "trailer", "transfer-encoding", "upgrade":
			continue
		}
		for _, value := range values {
			destination.Add(name, value)
		}
	}
}

func (s *ForwardPostDriver) ConvertToConfigAndSetOnWebhook(conf interface{}, webhook *model.Webhook) error {
	if forwardConfig, ok := conf.(model.ForwardPost); ok {
		webhook.ForwardPostConfig = forwardConfig
		webhook.ForwardPostConfig.Type = webhook.Driver
		return nil
	} else if configMap, ok := conf.(map[string]interface{}); ok {
		decoded := model.ForwardPost{}
		if err := mapstructure.Decode(configMap, &decoded); err != nil {
			return err
		}
		webhook.ForwardPostConfig = decoded
		webhook.ForwardPostConfig.Type = webhook.Driver
		return nil
	}
	return fmt.Errorf("can't convert config")
}

func (s *ForwardPostDriver) GetDriverConfigResource() interface{} {
	return model.ForwardPost{}
}

func (s *ForwardPostDriver) CustomizeSchema(schema *v1client.Schema) *v1client.Schema {
	return schema
}

func readBoundedBody(body io.Reader, maximum int64) ([]byte, error) {
	if body == nil {
		return nil, nil
	}
	limited := io.LimitReader(body, maximum+1)
	payload, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(payload)) > maximum {
		return nil, fmt.Errorf("request body exceeds %d bytes", maximum)
	}
	return payload, nil
}
