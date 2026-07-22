package config

import "testing"

func TestGetConfigPrefersPastureStackNamesAndKeepsLauncherAliases(t *testing.T) {
	t.Setenv("PASTURESTACK_API_URL", "https://api.example.test/v2-beta")
	t.Setenv("PASTURESTACK_API_ACCESS_KEY", "neutral-access")
	t.Setenv("PASTURESTACK_API_SECRET_KEY", "neutral-secret")
	t.Setenv("CATTLE_URL", "https://legacy.example.test/v2-beta")
	t.Setenv("CATTLE_ACCESS_KEY", "legacy-access")
	t.Setenv("CATTLE_SECRET_KEY", "legacy-secret")

	configured := GetConfig()
	if configured.APIURL != "https://api.example.test/v2-beta" || configured.AccessKey != "neutral-access" || configured.SecretKey != "neutral-secret" {
		t.Fatalf("neutral configuration did not take precedence: %#v", configured)
	}

	t.Setenv("PASTURESTACK_API_URL", "")
	t.Setenv("PASTURESTACK_API_ACCESS_KEY", "")
	t.Setenv("PASTURESTACK_API_SECRET_KEY", "")
	configured = GetConfig()
	if configured.APIURL != "https://legacy.example.test/v2-beta" || configured.AccessKey != "legacy-access" || configured.SecretKey != "legacy-secret" {
		t.Fatalf("launcher aliases did not work: %#v", configured)
	}
}
