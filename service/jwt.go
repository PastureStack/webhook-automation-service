package service

import (
	"bytes"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

const maximumTokenBytes = 16 << 10

func verifyJWT(encoded string, publicKey *rsa.PublicKey) (map[string]interface{}, error) {
	if publicKey == nil {
		return nil, fmt.Errorf("verification key is unavailable")
	}
	if len(encoded) == 0 || len(encoded) > maximumTokenBytes {
		return nil, fmt.Errorf("token size is invalid")
	}
	parts := strings.Split(encoded, ".")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return nil, fmt.Errorf("token must have three non-empty segments")
	}

	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("decode header: %w", err)
	}
	var header struct {
		Algorithm string `json:"alg"`
	}
	if err := decodeOneJSONValue(headerBytes, &header); err != nil {
		return nil, fmt.Errorf("decode header: %w", err)
	}
	if header.Algorithm != "RS256" {
		return nil, fmt.Errorf("unsupported signing algorithm")
	}

	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("decode signature: %w", err)
	}
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if err := rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, digest[:], signature); err != nil {
		return nil, fmt.Errorf("verify signature: %w", err)
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decode payload: %w", err)
	}
	claims := map[string]interface{}{}
	if err := decodeOneJSONValue(payload, &claims); err != nil {
		return nil, fmt.Errorf("decode payload: %w", err)
	}
	if err := validateNumericDate(claims, "exp", time.Now(), false); err != nil {
		return nil, err
	}
	if err := validateNumericDate(claims, "nbf", time.Now(), true); err != nil {
		return nil, err
	}
	return claims, nil
}

func decodeOneJSONValue(encoded []byte, target interface{}) error {
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing interface{}
	if err := decoder.Decode(&trailing); err == nil {
		return fmt.Errorf("trailing JSON value")
	} else if err != io.EOF {
		return fmt.Errorf("invalid trailing JSON: %w", err)
	}
	return nil
}

func validateNumericDate(claims map[string]interface{}, name string, now time.Time, notBefore bool) error {
	value, exists := claims[name]
	if !exists {
		return nil
	}
	number, ok := value.(json.Number)
	if !ok {
		return fmt.Errorf("%s claim must be numeric", name)
	}
	seconds, err := number.Int64()
	if err != nil {
		return fmt.Errorf("%s claim must be an integer", name)
	}
	boundary := time.Unix(seconds, 0)
	if notBefore && now.Before(boundary) {
		return fmt.Errorf("token is not active")
	}
	if !notBefore && !now.Before(boundary) {
		return fmt.Errorf("token has expired")
	}
	return nil
}
