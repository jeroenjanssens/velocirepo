package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type OAuthFlow struct {
	AuthURL      string
	TokenURL     string
	ClientID     string
	ClientSecret string
	RedirectURI  string
	Scopes       []string
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
}

func (f *OAuthFlow) Run(ctx context.Context, onReady func(authURL string)) (*TokenResponse, error) {
	state, err := randomState()
	if err != nil {
		return nil, fmt.Errorf("generate state: %w", err)
	}

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			errCh <- fmt.Errorf("state mismatch")
			http.Error(w, "State mismatch", http.StatusBadRequest)
			return
		}
		if errMsg := r.URL.Query().Get("error"); errMsg != "" {
			desc := r.URL.Query().Get("error_description")
			errCh <- fmt.Errorf("oauth error: %s: %s", errMsg, desc)
			fmt.Fprintf(w, "Authorization failed: %s", desc)
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			errCh <- fmt.Errorf("no code in callback")
			http.Error(w, "No code received", http.StatusBadRequest)
			return
		}
		codeCh <- code
		fmt.Fprint(w, "Authorization successful! You can close this tab.")
	})

	host, port := splitHostPort(f.RedirectURI)
	listener, err := net.Listen("tcp", host+":"+port)
	if err != nil {
		return nil, fmt.Errorf("listen on %s:%s: %w", host, port, err)
	}

	srv := &http.Server{Handler: mux}
	go srv.Serve(listener)
	defer srv.Close()

	if onReady != nil {
		onReady(f.BuildAuthURL(state))
	}

	var code string
	select {
	case code = <-codeCh:
	case err := <-errCh:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	return f.exchangeCode(ctx, code)
}

func (f *OAuthFlow) BuildAuthURL(state string) string {
	params := url.Values{
		"response_type": {"code"},
		"client_id":     {f.ClientID},
		"redirect_uri":  {f.RedirectURI},
		"scope":         {strings.Join(f.Scopes, " ")},
		"state":         {state},
	}
	return f.AuthURL + "?" + params.Encode()
}

func (f *OAuthFlow) exchangeCode(ctx context.Context, code string) (*TokenResponse, error) {
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {f.RedirectURI},
		"client_id":     {f.ClientID},
		"client_secret": {f.ClientSecret},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", f.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token exchange request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var body map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&body)
		return nil, fmt.Errorf("token exchange failed (%d): %v", resp.StatusCode, body)
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("decode token response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return nil, fmt.Errorf("empty access token in response")
	}

	return &tokenResp, nil
}

func randomState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func splitHostPort(redirectURI string) (string, string) {
	u, err := url.Parse(redirectURI)
	if err != nil {
		return "127.0.0.1", "9876"
	}
	host := u.Hostname()
	port := u.Port()
	if host == "" {
		host = "127.0.0.1"
	}
	if port == "" {
		port = "9876"
	}
	return host, port
}
