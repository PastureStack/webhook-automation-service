package service

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"strings"
	"testing"
	"time"
)

func TestVerifyJWT(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	validClaims := map[string]interface{}{
		"driver":    "scaleService",
		"projectId": "1a1",
		"uuid":      "fixture",
		"exp":       time.Now().Add(time.Minute).Unix(),
	}
	token := signTestToken(t, privateKey, "RS256", validClaims)
	claims, err := verifyJWT(token, &privateKey.PublicKey)
	if err != nil {
		t.Fatalf("valid token rejected: %v", err)
	}
	if claims["driver"] != "scaleService" {
		t.Fatalf("unexpected claims: %#v", claims)
	}

	tests := []struct {
		name  string
		token string
	}{
		{name: "algorithm confusion", token: signTestToken(t, privateKey, "HS256", validClaims)},
		{name: "expired", token: signTestToken(t, privateKey, "RS256", map[string]interface{}{"exp": time.Now().Add(-time.Minute).Unix()})},
		{name: "not active", token: signTestToken(t, privateKey, "RS256", map[string]interface{}{"nbf": time.Now().Add(time.Minute).Unix()})},
		{name: "wrong date type", token: signTestToken(t, privateKey, "RS256", map[string]interface{}{"exp": "tomorrow"})},
		{name: "malformed", token: "one.two"},
		{name: "oversized", token: strings.Repeat("x", maximumTokenBytes+1)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := verifyJWT(test.token, &privateKey.PublicKey); err == nil {
				t.Fatal("invalid token accepted")
			}
		})
	}

	tampered := strings.Split(token, ".")
	tampered[1] = base64.RawURLEncoding.EncodeToString([]byte(`{"driver":"forwardPost"}`))
	if _, err := verifyJWT(strings.Join(tampered, "."), &privateKey.PublicKey); err == nil {
		t.Fatal("tampered token accepted")
	}
}

func TestParseRSAPublicKey(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: encoded})
	parsed, err := parseRSAPublicKey(pemBytes)
	if err != nil {
		t.Fatalf("valid public key rejected: %v", err)
	}
	if parsed.N.Cmp(privateKey.N) != 0 || parsed.E != privateKey.E {
		t.Fatal("parsed public key does not match")
	}
	if _, err := parseRSAPublicKey(append(pemBytes, pemBytes...)); err == nil {
		t.Fatal("multiple PEM blocks accepted")
	}
	privatePEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})
	if _, err := parseRSAPublicKey(privatePEM); err == nil {
		t.Fatal("private key accepted as a public key")
	}
}

func signTestToken(t *testing.T, privateKey *rsa.PrivateKey, algorithm string, claims map[string]interface{}) string {
	t.Helper()
	header, err := json.Marshal(map[string]string{"alg": algorithm, "typ": "JWT"})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	headerPart := base64.RawURLEncoding.EncodeToString(header)
	payloadPart := base64.RawURLEncoding.EncodeToString(payload)
	signed := headerPart + "." + payloadPart
	digest := sha256.Sum256([]byte(signed))
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	return signed + "." + base64.RawURLEncoding.EncodeToString(signature)
}
