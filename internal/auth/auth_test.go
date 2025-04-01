package auth

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// Test getting a hashed password
func TestHashPassword(t *testing.T) {
	samplePasswords := []string{
		"password",
		"anotherPassword",
		"yetAnotherPassword",
	}
	// Want to make sure we have no errors
	for i, _ := range samplePasswords {
		_, err := HashPassword(samplePasswords[i])
		if err != nil {
			t.Error(err)
		}
	}
}

// Test able to check the hash
func TestCheckPasswordHash(t *testing.T) {
	passwords := []string{
		"password",
		"anotherPassword",
		"yetAnotherPassword",
	}

	for i, _ := range passwords {
		hashed, _ := HashPassword(passwords[i])
		err := CheckPasswordHash(hashed, passwords[i])
		if err != nil {
			t.Error(err)
		}

	}
}

// Test MakeJWT & Validate JWT
// Make sure that you can create and validate JWTs, and that expired tokens are rejected and JWTs signed with the wrong secret are rejected.
func TestMakeAndValidateValidJWT(t *testing.T) {
	tokenSecret := "Super secret!"
	// Valid - should pass
	validUserIDs := []uuid.UUID{
		uuid.New(),
		uuid.New(),
		uuid.New(),
	}
	validDuration, _ := time.ParseDuration("1m")
	// We expect all to be valid
	for i, _ := range validUserIDs {
		validJWT, err := MakeJWT(
			validUserIDs[i],
			tokenSecret,
			validDuration,
		)
		if err != nil {
			t.Error(err)
		}
		// Verify we can get the correct UUID back
		checkUUID, err := ValidateJWT(validJWT, tokenSecret)
		if err != nil {
			t.Errorf("Got back the following error: %v", err)
		}
		if checkUUID != validUserIDs[i] {
			t.Errorf(`UserIDs do not match:
				Expecting: "%v"
				Got back "%v" instead.`, validUserIDs[i], checkUUID)
		}
	}

}

// Testing invalid JWTs
func TestExpiredJWT(t *testing.T) {
	// Create a token that's already expired (negative duration)
	userID := uuid.New()
	token, err := MakeJWT(userID, "secret", -1*time.Hour)
	if err != nil {
		t.Fatalf("Error creating expired token: %v", err)
	}

	// Validate should fail with expiration error
	_, err = ValidateJWT(token, "secret")
	if err == nil {
		t.Error("Expected error for expired token, got nil")
	}
}

func TestInvalidSignature(t *testing.T) {
	// Create a token with one secret
	userID := uuid.New()
	token, err := MakeJWT(userID, "correct-secret", time.Hour)
	if err != nil {
		t.Fatalf("Error creating token: %v", err)
	}

	// Validate with different secret
	_, err = ValidateJWT(token, "wrong-secret")
	if err == nil {
		t.Error("Expected error for invalid signature, got nil")
	}
}

func TestMalformedJWT(t *testing.T) {
	// Create a deliberately malformed token
	malformedToken := "not.a.validtoken"

	_, err := ValidateJWT(malformedToken, "secret")
	if err == nil {
		t.Error("Expected error for malformed token, got nil")
	}
}

func TestTamperedJWT(t *testing.T) {
	// This is trickier - you'd need to manually decode, modify, and re-encode
	// a token without
	// 1. Create a valid token
	userID := uuid.New()
	validToken, err := MakeJWT(userID, "secret", time.Hour)
	if err != nil {
		t.Fatalf("Error creating token: %v", err)
	}

	// 2. Split the token into its three parts (header, payload, signature)
	parts := strings.Split(validToken, ".")
	if len(parts) != 3 {
		t.Fatalf("Token doesn't have three parts: %s", validToken)
	}
	// 3. Decode the payload (middle part)
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("Error decoding payload: %v", err)
	}

	// 4. Modify the payload (e.g., change the subject to a different UUID)
	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		t.Fatalf("Error unmarshaling claims: %v", err)
	}
	claims["sub"] = uuid.New().String() // Change subject to a different UUID

	// 5. Re-encode the modified payload
	modifiedPayload, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("Error marshaling modified claims: %v", err)
	}

	// 6. Create the tampered token (keep header and signature the same)
	encodedModifiedPayload := base64.RawURLEncoding.EncodeToString(modifiedPayload)
	tamperedToken := parts[0] + "." + encodedModifiedPayload + "." + parts[2]

	// Now try to validate the tampered token
	_, err = ValidateJWT(tamperedToken, "secret")

	// The validation should fail because the signature no longer matches the modified payload
	if err == nil {
		t.Error("Expected error when validating tampered token, got nil")
	}

}

// Test GetBearerToken
func TestGetBearerTokenValid(t *testing.T) {
	newHeader := http.Header{}
	newHeader.Add("Authorization", "Bearer TOKEN_STRING")

	tokenString, err := GetBearerToken(newHeader)
	if err != nil {
		t.Errorf("ERROR getting token_string: %v", err)
	}
	if tokenString != "TOKEN_STRING" {
		t.Errorf("Expected: %v\nGot: %v", "TOKEN_STRING", tokenString)
	}

}

// Test Invalid Token
func TestGetBearerTokenInvalid(t *testing.T) {
	newHeader := http.Header{}

	_, err := GetBearerToken(newHeader)
	if err == nil {
		t.Error("Expected there to be no header found, found it anyway")
	}
}
