package drivers

import "testing"

func TestValidateScaleHostURLBindsControlPlaneOrigin(t *testing.T) {
	base := "https://api.example.test:8443/v2-beta"
	accepted, err := validateScaleHostURL(base, "https://api.example.test:8443/v2-beta/projects/p1/hosts/h1")
	if err != nil || accepted == "" {
		t.Fatalf("same-origin URL rejected: %q, %v", accepted, err)
	}
	for _, candidate := range []string{
		"https://metadata.example.test/latest",
		"https://user:secret@api.example.test:8443/v2-beta/hosts",
		"file:///etc/passwd",
	} {
		if _, err := validateScaleHostURL(base, candidate); err == nil {
			t.Fatalf("unsafe URL accepted: %s", candidate)
		}
	}
}
