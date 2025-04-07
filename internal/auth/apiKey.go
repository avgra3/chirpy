package auth

import (
	"errors"
	"net/http"
	"strings"
)

func GetAPIKey(headers http.Header) (string, error) {
	authHeader, ok := headers["Authorization"]
	if !ok {
		return "", errors.New("Authorization header not found!")
	}
	apiKeyString := strings.Split(authHeader[0], " ")
	apiKey := strings.Split(apiKeyString[1], " ")
	return apiKey[0], nil
}
