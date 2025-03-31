package auth

import (
	"testing"
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
