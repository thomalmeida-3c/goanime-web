package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Mercado Livre retired open/keyless product search (confirmed live: their
// /sites/MLB/search and /products/search both now answer 403 "forbidden" /
// "UNAUTHORIZED_RESULT_FROM_POLICIES" without a token). Real access needs
// their full server-side OAuth2 "authorization_code" flow — a human has to
// click through their consent screen once, there's no app-only shortcut.
// See README "Produtos" for the one-time setup this requires.
const (
	mlAuthorizeURL = "https://auth.mercadolivre.com.br/authorization"
	mlTokenURL     = "https://api.mercadolibre.com/oauth/token"
)

type mlToken struct {
	AccessToken  string    `json:"accessToken"`
	RefreshToken string    `json:"refreshToken"`
	ExpiresAt    time.Time `json:"expiresAt"`
}

var mlTokenMu sync.Mutex

func mlDataDir() string {
	dir := os.Getenv("DATA_DIR")
	if dir == "" {
		dir = "./data"
	}
	return dir
}

func mlTokenPath() string {
	return filepath.Join(mlDataDir(), "ml_token.json")
}

func loadMLToken() (*mlToken, error) {
	mlTokenMu.Lock()
	defer mlTokenMu.Unlock()

	data, err := os.ReadFile(mlTokenPath())
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var t mlToken
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

func saveMLToken(t *mlToken) error {
	mlTokenMu.Lock()
	defer mlTokenMu.Unlock()

	if err := os.MkdirAll(mlDataDir(), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(mlTokenPath(), data, 0o644)
}

func registerMLRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/ml/status", handleMLStatus)
	mux.HandleFunc("GET /api/ml/connect", handleMLConnect)
	mux.HandleFunc("GET /api/ml/callback", handleMLCallback)
}

// handleMLStatus tells the frontend whether ML_CLIENT_ID/SECRET are even
// set (configured) and whether the one-time OAuth consent has been done
// (connected) — the Produtos page uses this to show the right empty state.
func handleMLStatus(w http.ResponseWriter, r *http.Request) {
	configured := os.Getenv("ML_CLIENT_ID") != "" && os.Getenv("ML_CLIENT_SECRET") != "" && os.Getenv("ML_REDIRECT_URI") != ""
	t, _ := loadMLToken()
	writeJSON(w, http.StatusOK, map[string]any{
		"configured": configured,
		"connected":  t != nil,
	})
}

func handleMLConnect(w http.ResponseWriter, r *http.Request) {
	clientID := os.Getenv("ML_CLIENT_ID")
	redirectURI := os.Getenv("ML_REDIRECT_URI")
	if clientID == "" || redirectURI == "" {
		writeError(w, http.StatusPreconditionFailed, errMLNotConfigured)
		return
	}
	u := mlAuthorizeURL + "?response_type=code&client_id=" + url.QueryEscape(clientID) +
		"&redirect_uri=" + url.QueryEscape(redirectURI)
	http.Redirect(w, r, u, http.StatusFound)
}

func handleMLCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("missing 'code' from Mercado Livre"))
		return
	}

	token, err := exchangeMLCode(r.Context(), code)
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}
	if err := saveMLToken(token); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, `<html><body style="font-family:sans-serif;padding:40px">`+
		`<h2>Mercado Livre conectado</h2><p>Pode fechar esta aba e voltar pro app.</p></body></html>`)
}

func exchangeMLCode(ctx context.Context, code string) (*mlToken, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("client_id", os.Getenv("ML_CLIENT_ID"))
	form.Set("client_secret", os.Getenv("ML_CLIENT_SECRET"))
	form.Set("code", code)
	form.Set("redirect_uri", os.Getenv("ML_REDIRECT_URI"))
	return postMLTokenRequest(ctx, form)
}

func postMLTokenRequest(ctx context.Context, form url.Values) (*mlToken, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, mlTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := mercadoLivreClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var parsed struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		Error        string `json:"error"`
		ErrorDesc    string `json:"error_description"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}
	if parsed.Error != "" {
		return nil, fmt.Errorf("mercado livre oauth: %s: %s", parsed.Error, parsed.ErrorDesc)
	}
	if parsed.AccessToken == "" {
		return nil, fmt.Errorf("mercado livre oauth: empty access token in response")
	}

	return &mlToken{
		AccessToken:  parsed.AccessToken,
		RefreshToken: parsed.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(parsed.ExpiresIn) * time.Second),
	}, nil
}

// getValidMLAccessToken returns a usable access token, transparently
// refreshing it first if it's expired or about to be (access tokens last
// 6h; the refresh token — valid 6 months — renews it with no user
// interaction). Returns "" with no error if the one-time OAuth consent was
// simply never done.
func getValidMLAccessToken(ctx context.Context) (string, error) {
	t, err := loadMLToken()
	if err != nil || t == nil {
		return "", err
	}
	if time.Now().Before(t.ExpiresAt.Add(-1 * time.Minute)) {
		return t.AccessToken, nil
	}

	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("client_id", os.Getenv("ML_CLIENT_ID"))
	form.Set("client_secret", os.Getenv("ML_CLIENT_SECRET"))
	form.Set("refresh_token", t.RefreshToken)

	refreshed, err := postMLTokenRequest(ctx, form)
	if err != nil {
		return "", err
	}
	if err := saveMLToken(refreshed); err != nil {
		return "", err
	}
	return refreshed.AccessToken, nil
}
