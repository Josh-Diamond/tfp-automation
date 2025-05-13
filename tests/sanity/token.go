package sanity

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
	"github.com/sirupsen/logrus"
)

const (
	urlProtocol       = "https://"
	v1TokenEndpoint   = "/v1/token"
	v3TokenEndpoint   = "/v3/token"
	post 	          = "POST"
	contentType       = "Content-Type"
	contentTypeJSON   = "application/json"
)

type TokenResponse struct {
	Token string `json:"token"`
}

type TokenRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// GenerateUserTokenV3 retrieves a bearer token using the /v3/token endpoint
func GenerateUserTokenV3(username, password, host string) (string, error) {
	tokenReq := TokenRequest{
		Username: username,
		Password: password,
	}

	bodyContent, err := json.Marshal(tokenReq)
	if err != nil {
		logrus.Errorf("failed to marshal token request: %v", err)
		return "", err
	}

	url := urlProtocol + host + v3TokenEndpoint
	req, err := http.NewRequest(post, url, bytes.NewBuffer(bodyContent))
	if err != nil {
		logrus.Errorf("failed to create token request: %v", err)
		return "", err
	}

	req.Header.Set(contentType, contentTypeJSON)

	client := &http.Client{
		Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
		Timeout:   10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		logrus.Errorf("token request failed: %v", err)
		return "", err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 300 {
		logrus.Errorf("token request failed: status %d - %s", resp.StatusCode, string(respBody))
		return "", fmt.Errorf("token request failed: status %d", resp.StatusCode)
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(respBody, &tokenResp); err != nil {
		logrus.Errorf("failed to unmarshal token response: %v", err)
		return "", err
	}

	if tokenResp.Token == "" {
		logrus.Errorf("received empty token in response")
		return "", fmt.Errorf("received empty token in response")
	}

	return tokenResp.Token, nil
}

// GenerateUserTokenV1 retrieves a bearer token using the /v1/token endpoint
func GenerateUserTokenV1(username, password, host string) (string, error) {
	tokenReq := TokenRequest{
		Username: username,
		Password: password,
	}

	bodyContent, err := json.Marshal(tokenReq)
	if err != nil {
		logrus.Errorf("failed to marshal token request: %v", err)
		return "", err
	}

	url := urlProtocol + host + v1TokenEndpoint
	req, err := http.NewRequest(post, url, bytes.NewBuffer(bodyContent))
	if err != nil {
		logrus.Errorf("failed to create token request: %v", err)
		return "", err
	}

	req.Header.Set(contentType, contentTypeJSON)

	client := &http.Client{
		Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
		Timeout:   10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		logrus.Errorf("token request failed: %v", err)
		return "", err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 300 {
		logrus.Errorf("token request failed: status %d - %s", resp.StatusCode, string(respBody))
		return "", fmt.Errorf("token request failed: status %d", resp.StatusCode)
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(respBody, &tokenResp); err != nil {
		logrus.Errorf("failed to unmarshal token response: %v", err)
		return "", err
	}

	if tokenResp.Token == "" {
		logrus.Errorf("received empty token in response")
		return "", fmt.Errorf("received empty token in response")
	}

	return tokenResp.Token, nil
}
