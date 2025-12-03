package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrMissingSignature is returned when the signature header is missing
	ErrMissingSignature = errors.New("missing X-Hub-Signature-256 header")

	// ErrMalformedSignature is returned when the signature header format is invalid
	ErrMalformedSignature = errors.New("malformed signature header")

	// ErrInvalidSignature is returned when the signature doesn't match
	ErrInvalidSignature = errors.New("invalid signature")
)

// ValidateHMAC validates the HMAC-SHA256 signature from GitHub webhooks.
// It parses the X-Hub-Signature-256 header, computes the expected signature,
// and performs a constant-time comparison to prevent timing attacks.
//
// The signature header format is: "sha256=<hex-encoded-signature>"
//
// Returns nil if the signature is valid, or an error otherwise.
func ValidateHMAC(payload []byte, signature string, secret string) error {
	// Check for missing signature
	if signature == "" {
		return ErrMissingSignature
	}

	// Parse signature header: "sha256=<hex-signature>"
	if !strings.HasPrefix(signature, "sha256=") {
		return ErrMalformedSignature
	}

	// Extract the hex-encoded signature
	providedSig := strings.TrimPrefix(signature, "sha256=")
	if len(providedSig) == 0 {
		return ErrMalformedSignature
	}

	// Decode the provided signature from hex
	providedBytes, err := hex.DecodeString(providedSig)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrMalformedSignature, err)
	}

	// Compute expected signature
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expectedBytes := mac.Sum(nil)

	// Constant-time comparison to prevent timing attacks
	if !hmac.Equal(expectedBytes, providedBytes) {
		return ErrInvalidSignature
	}

	return nil
}
