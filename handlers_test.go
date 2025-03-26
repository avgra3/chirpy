package main

import (
	"testing"
)

// Test clean words -- checking for a valid return
func TestCleanWords(t *testing.T) {
	input := []string{
		"I had something interesting for breakfast",
		"I hear Mastodon is better than Chirpy. sharbert I need to migrate",
		"I really need a kerfuffle to go to bed sooner, Fornax !",
	}

	expected := []string{
		"I had something interesting for breakfast",
		"I hear Mastodon is better than Chirpy. **** I need to migrate",
		"I really need a **** to go to bed sooner, **** !",
	}

	for i, _ := range input {
		actual := cleanWords(input[i])
		if actual != expected[i] {
			t.Errorf(`cleanWords(%v) = %v, want %v`, input[i], actual, expected[i])
		}

	}

}
