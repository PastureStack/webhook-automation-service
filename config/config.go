package config

import (
	"os"
)

type Config struct {
	APIURL    string
	AccessKey string
	SecretKey string
}

func GetConfig() Config {
	return Config{
		APIURL:    firstEnvironmentValue("PASTURESTACK_API_URL", "CATTLE_URL"),
		AccessKey: firstEnvironmentValue("PASTURESTACK_API_ACCESS_KEY", "CATTLE_ACCESS_KEY"),
		SecretKey: firstEnvironmentValue("PASTURESTACK_API_SECRET_KEY", "CATTLE_SECRET_KEY"),
	}
}

func firstEnvironmentValue(names ...string) string {
	for _, name := range names {
		if value := os.Getenv(name); value != "" {
			return value
		}
	}
	return ""
}
