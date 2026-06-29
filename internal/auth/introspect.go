package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type IntrospectResult struct {
	Active    bool   `json:"active"`
	ExpiresAt int64  `json:"expires_at"`
	Scope     string `json:"scope"`
}

const linkedInIntrospectURL = "https://www.linkedin.com/oauth/v2/introspectToken"

func IntrospectLinkedInToken(ctx context.Context, token, clientID, clientSecret string) (*IntrospectResult, error) {
	return introspectWithURL(ctx, linkedInIntrospectURL, token, clientID, clientSecret)
}

func introspectWithURL(ctx context.Context, endpoint, token, clientID, clientSecret string) (*IntrospectResult, error) {
	data := url.Values{
		"token":         {token},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("introspect request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("introspect returned %d", resp.StatusCode)
	}

	var result IntrospectResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode introspect response: %w", err)
	}

	return &result, nil
}

func CheckLinkedInTokenExpiry(ctx context.Context, token, clientID, clientSecret string) {
	checkLinkedInTokenExpiryWith(ctx, linkedInIntrospectURL, token, clientID, clientSecret, os.Stderr)
}

func checkLinkedInTokenExpiryWith(ctx context.Context, endpoint, token, clientID, clientSecret string, w io.Writer) {
	if clientID == "" || clientSecret == "" {
		return
	}

	result, err := introspectWithURL(ctx, endpoint, token, clientID, clientSecret)
	if err != nil {
		return
	}

	if !result.Active {
		fmt.Fprintln(w, "WARNING: LinkedIn token is no longer active. Run 'velocirepo auth linkedin' to refresh.")
		return
	}

	if result.ExpiresAt > 0 {
		expiresAt := time.Unix(result.ExpiresAt, 0)
		remaining := time.Until(expiresAt)
		if remaining < 7*24*time.Hour {
			days := int(remaining.Hours() / 24)
			if days <= 0 {
				fmt.Fprintln(w, "WARNING: LinkedIn token expires today. Run 'velocirepo auth linkedin' to refresh.")
			} else {
				fmt.Fprintf(w, "WARNING: LinkedIn token expires in %d day(s). Run 'velocirepo auth linkedin' to refresh.\n", days)
			}
		}
	}
}
