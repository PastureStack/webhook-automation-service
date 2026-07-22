package service

import (
	"fmt"
	"time"

	"github.com/PastureStack/webhook-automation-service/config"
	"github.com/rancher/go-rancher/v2"
)

type APIClientFactory interface {
	GetClient(projectID string) (*client.RancherClient, error)
}

type ClientFactory struct{}

func (f *ClientFactory) GetClient(projectID string) (*client.RancherClient, error) {
	config := config.GetConfig()
	url := fmt.Sprintf("%s/projects/%s/schemas", config.APIURL, projectID)
	apiClient, err := client.NewRancherClient(&client.ClientOpts{
		Timeout:   time.Second * 30,
		Url:       url,
		AccessKey: config.AccessKey,
		SecretKey: config.SecretKey,
	})
	if err != nil {
		return &client.RancherClient{}, fmt.Errorf("Error in creating API client")
	}
	return apiClient, nil
}
