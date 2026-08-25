package service

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"
)

const maximumPublicKeyBytes = 64 << 10

// GetPublicKey loads the only key material required by the service. The
// control-plane private key is deliberately neither read nor retained.
func GetPublicKey(c *cli.Command) (*rsa.PublicKey, error) {
	keyFile := c.String("public-key-file")
	keyContents := c.String("public-key-contents")
	if keyFile != "" && keyContents != "" {
		return nil, fmt.Errorf("specify either --public-key-file or --public-key-contents, not both")
	}

	var encoded []byte
	var err error
	switch {
	case keyFile != "":
		encoded, err = os.ReadFile(keyFile)
		if err != nil {
			return nil, fmt.Errorf("read public key: %w", err)
		}
	case keyContents != "":
		encoded = []byte(keyContents)
	default:
		return nil, fmt.Errorf("provide --public-key-file or --public-key-contents")
	}
	if len(encoded) > maximumPublicKeyBytes {
		return nil, fmt.Errorf("public key exceeds %d bytes", maximumPublicKeyBytes)
	}
	return parseRSAPublicKey(encoded)
}

func parseRSAPublicKey(encoded []byte) (*rsa.PublicKey, error) {
	block, remainder := pem.Decode(encoded)
	if block == nil || len(bytes.TrimSpace(remainder)) != 0 {
		return nil, fmt.Errorf("public key must contain exactly one PEM block")
	}

	var parsed interface{}
	var err error
	switch block.Type {
	case "PUBLIC KEY":
		parsed, err = x509.ParsePKIXPublicKey(block.Bytes)
	case "RSA PUBLIC KEY":
		parsed, err = x509.ParsePKCS1PublicKey(block.Bytes)
	case "CERTIFICATE":
		var certificate *x509.Certificate
		certificate, err = x509.ParseCertificate(block.Bytes)
		if err == nil {
			parsed = certificate.PublicKey
		}
	default:
		return nil, fmt.Errorf("unsupported public key PEM type %q", block.Type)
	}
	if err != nil {
		return nil, fmt.Errorf("parse public key: %w", err)
	}
	publicKey, ok := parsed.(*rsa.PublicKey)
	if !ok || publicKey.N == nil || publicKey.E < 3 {
		return nil, fmt.Errorf("public key is not a valid RSA key")
	}
	return publicKey, nil
}
