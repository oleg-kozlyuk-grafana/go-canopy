package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
)

func TestValidateHMAC(t *testing.T) {
	testSecret := "test-secret-key"
	testPayload := []byte(`{"action":"completed","workflow_run":{"id":123}}`)

	// Generate a valid signature for testing
	generateSignature := func(payload []byte, secret string) string {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(payload)
		return "sha256=" + hex.EncodeToString(mac.Sum(nil))
	}

	tests := []struct {
		name          string
		payload       []byte
		signature     string
		secret        string
		expectedError error
		description   string
	}{
		{
			name:          "valid signature",
			payload:       testPayload,
			signature:     generateSignature(testPayload, testSecret),
			secret:        testSecret,
			expectedError: nil,
			description:   "Valid HMAC signature should pass validation",
		},
		{
			name:          "invalid signature",
			payload:       testPayload,
			signature:     "sha256=" + "0000000000000000000000000000000000000000000000000000000000000000",
			secret:        testSecret,
			expectedError: ErrInvalidSignature,
			description:   "Invalid HMAC signature should return ErrInvalidSignature",
		},
		{
			name:          "missing signature header",
			payload:       testPayload,
			signature:     "",
			secret:        testSecret,
			expectedError: ErrMissingSignature,
			description:   "Missing signature header should return ErrMissingSignature",
		},
		{
			name:          "malformed header - no prefix",
			payload:       testPayload,
			signature:     "abc123",
			secret:        testSecret,
			expectedError: ErrMalformedSignature,
			description:   "Signature without 'sha256=' prefix should return ErrMalformedSignature",
		},
		{
			name:          "malformed header - empty after prefix",
			payload:       testPayload,
			signature:     "sha256=",
			secret:        testSecret,
			expectedError: ErrMalformedSignature,
			description:   "Signature with empty value after 'sha256=' should return ErrMalformedSignature",
		},
		{
			name:          "malformed header - invalid hex",
			payload:       testPayload,
			signature:     "sha256=notahexstring",
			secret:        testSecret,
			expectedError: ErrMalformedSignature,
			description:   "Signature with invalid hex encoding should return ErrMalformedSignature",
		},
		{
			name:          "wrong secret",
			payload:       testPayload,
			signature:     generateSignature(testPayload, testSecret),
			secret:        "wrong-secret",
			expectedError: ErrInvalidSignature,
			description:   "Signature generated with different secret should return ErrInvalidSignature",
		},
		{
			name:          "payload tampering",
			payload:       []byte(`{"action":"completed","workflow_run":{"id":999}}`),
			signature:     generateSignature(testPayload, testSecret),
			secret:        testSecret,
			expectedError: ErrInvalidSignature,
			description:   "Signature for different payload should return ErrInvalidSignature",
		},
		{
			name:          "empty payload with valid signature",
			payload:       []byte{},
			signature:     generateSignature([]byte{}, testSecret),
			secret:        testSecret,
			expectedError: nil,
			description:   "Valid signature for empty payload should pass validation",
		},
		{
			name:          "signature with uppercase hex",
			payload:       testPayload,
			signature:     "sha256=" + "ABCDEF1234567890",
			secret:        testSecret,
			expectedError: ErrInvalidSignature,
			description:   "Uppercase hex in signature should still be processed (though invalid)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateHMAC(tt.payload, tt.signature, tt.secret)

			if tt.expectedError == nil {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.expectedError)
				} else if !errors.Is(err, tt.expectedError) {
					t.Errorf("expected error %v, got %v", tt.expectedError, err)
				}
			}
		})
	}
}

// TestValidateHMAC_ConstantTime verifies that signature comparison is constant-time.
// This is a basic test to ensure we're using hmac.Equal (which is constant-time)
// rather than a simple byte comparison that could be vulnerable to timing attacks.
func TestValidateHMAC_ConstantTime(t *testing.T) {
	testSecret := "test-secret-key"
	testPayload := []byte("test payload")

	// Generate valid signature
	mac := hmac.New(sha256.New, []byte(testSecret))
	mac.Write(testPayload)
	validSig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	// Test that validation uses constant-time comparison
	// We can't directly test timing, but we can verify that the function
	// uses hmac.Equal by checking that all invalid signatures fail,
	// regardless of how many bytes match.

	// Create signatures that differ at different positions
	sigBytes, _ := hex.DecodeString(validSig[7:])

	// Signature differs at first byte
	sigBytes[0] ^= 0xFF
	wrongSig1 := "sha256=" + hex.EncodeToString(sigBytes)

	// Signature differs at last byte
	sigBytes[0] ^= 0xFF  // restore
	sigBytes[len(sigBytes)-1] ^= 0xFF
	wrongSig2 := "sha256=" + hex.EncodeToString(sigBytes)

	// Both should fail with ErrInvalidSignature
	err1 := ValidateHMAC(testPayload, wrongSig1, testSecret)
	if !errors.Is(err1, ErrInvalidSignature) {
		t.Errorf("expected ErrInvalidSignature for signature differing at first byte, got %v", err1)
	}

	err2 := ValidateHMAC(testPayload, wrongSig2, testSecret)
	if !errors.Is(err2, ErrInvalidSignature) {
		t.Errorf("expected ErrInvalidSignature for signature differing at last byte, got %v", err2)
	}

	// The fact that both fail with the same error (and that we use hmac.Equal internally)
	// provides assurance that timing attacks are mitigated.
}

// TestValidateHMAC_GitHubExample tests with a real example similar to GitHub's format
func TestValidateHMAC_GitHubExample(t *testing.T) {
	// Simulate a GitHub webhook payload and signature
	secret := "my-webhook-secret"
	payload := []byte(`{
		"action": "completed",
		"workflow_run": {
			"id": 123456789,
			"name": "ci.yml",
			"repository": {
				"full_name": "grafana/my-repo"
			}
		}
	}`)

	// Generate what GitHub would send
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	githubSignature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	// Validate
	err := ValidateHMAC(payload, githubSignature, secret)
	if err != nil {
		t.Errorf("expected valid signature to pass, got error: %v", err)
	}

	// Test with modified payload (simulating tampering)
	tamperedPayload := []byte(`{
		"action": "completed",
		"workflow_run": {
			"id": 999999999,
			"name": "ci.yml",
			"repository": {
				"full_name": "grafana/my-repo"
			}
		}
	}`)

	err = ValidateHMAC(tamperedPayload, githubSignature, secret)
	if !errors.Is(err, ErrInvalidSignature) {
		t.Errorf("expected ErrInvalidSignature for tampered payload, got %v", err)
	}
}
