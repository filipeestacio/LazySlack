package auth

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"runtime"

	"github.com/filipeestacio/lazyslack/internal/config"
	slackclient "github.com/filipeestacio/lazyslack/internal/slack"
)

func ValidateToken(cfg *config.Config, baseURL string) (*slackclient.AuthInfo, error) {
	opts := []slackclient.ClientOption{}
	if baseURL != "" {
		opts = append(opts, slackclient.WithBaseURL(baseURL))
	}

	client := slackclient.NewClient(cfg.Auth.Token, cfg.Auth.Cookie, opts...)
	return client.AuthTest()
}

func RunOAuthFlow(clientID, clientSecret string, scopes []string) (string, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", fmt.Errorf("failed to start callback server: %w", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	redirectURI := fmt.Sprintf("http://localhost:%d/callback", port)
	tokenCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			errCh <- fmt.Errorf("no code in callback")
			fmt.Fprint(w, "Error: no authorization code received. You can close this tab.")
			return
		}

		token, err := exchangeCode(clientID, clientSecret, code, redirectURI)
		if err != nil {
			errCh <- err
			fmt.Fprint(w, "Error exchanging code. Check terminal for details.")
			return
		}
		tokenCh <- token
		fmt.Fprint(w, "Authentication successful! You can close this tab and return to LazySlack.")
	})

	server := &http.Server{Handler: mux}
	go server.Serve(listener)
	defer server.Close()

	scopeStr := ""
	for i, s := range scopes {
		if i > 0 {
			scopeStr += ","
		}
		scopeStr += s
	}

	authURL := fmt.Sprintf(
		"https://slack.com/oauth/v2/authorize?client_id=%s&user_scope=%s&redirect_uri=%s",
		clientID, scopeStr, redirectURI,
	)
	openBrowser(authURL)

	select {
	case token := <-tokenCh:
		return token, nil
	case err := <-errCh:
		return "", err
	}
}

func exchangeCode(clientID, clientSecret, code, redirectURI string) (string, error) {
	resp, err := http.PostForm("https://slack.com/api/oauth.v2.access", map[string][]string{
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"code":          {code},
		"redirect_uri":  {redirectURI},
	})
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		OK         bool   `json:"ok"`
		Error      string `json:"error"`
		AuthedUser struct {
			AccessToken string `json:"access_token"`
		} `json:"authed_user"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if !result.OK {
		return "", fmt.Errorf("oauth exchange failed: %s", result.Error)
	}
	return result.AuthedUser.AccessToken, nil
}

func openBrowser(url string) {
	switch runtime.GOOS {
	case "darwin":
		exec.Command("open", url).Start()
	case "linux":
		exec.Command("xdg-open", url).Start()
	}
}
