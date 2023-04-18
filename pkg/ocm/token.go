package ocm

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

var (
	ErrTokenInvalid = errors.New("invalid token")
)

type Token struct {
	RefreshToken string `json:"refresh_token"`
	URL          string `json:"url"`
	TokenURL     string `json:"token_url"`
	ClientID     string `json:"client_id"`
	AccessToken  string `json:"access_token"`
}

func NewToken(file string) (*Token, error) {
	tokenBytes, err := os.ReadFile(file)
	if err != nil {
		return &Token{}, fmt.Errorf("unable to read token from file [%s] - %w", file, err)
	}

	token, err := getTokenData(tokenBytes)
	if err != nil {
		return &Token{}, fmt.Errorf("unable to retrieve token data from file [%s] - %w", file, err)
	}

	return &token, nil
}

func getTokenData(tokenBytes []byte) (Token, error) {
	token := Token{}

	// return immediately if we have an empty token
	if len(tokenBytes) == 0 {
		return token, fmt.Errorf("missing token data - %w", ErrTokenInvalid)
	}

	// serialize token as json
	if err := json.Unmarshal(tokenBytes, &token); err != nil {
		return token, fmt.Errorf("unable to serialize token - %w", ErrTokenInvalid)
	}

	return token, nil
}
