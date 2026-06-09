package ml

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

const (
	APIURL      = "https://api.mercadolibre.com"
	AuthURL     = "https://auth.mercadolivre.com.br/authorization"
	TokenURL    = "https://api.mercadolibre.com/oauth/token"
	UserAgent   = "vinyligo/0.1"
	mlTokenFile = "./ml_token.json"
)

type Client struct {
	accessToken string
	httpClient  *http.Client
}

type MLTokenData struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
}

func NewClient(clientID, clientSecret string) *Client {
	token := loadOrRefreshToken(clientID, clientSecret)
	return &Client{
		accessToken: token,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
	}
}

func loadOrRefreshToken(clientID, clientSecret string) string {
	data, err := loadMLToken()
	if err != nil {
		return authenticateML(clientID, clientSecret)
	}
	if time.Now().Unix() < data.ExpiresAt-60 {
		return data.AccessToken
	}
	newData, err := refreshMLToken(clientID, clientSecret, data.RefreshToken)
	if err != nil {
		return authenticateML(clientID, clientSecret)
	}
	return newData.AccessToken
}

func authenticateML(clientID, clientSecret string) string {
	const redirectURI = "https://httpbin.org/anything"

	authURL := fmt.Sprintf("%s?response_type=code&client_id=%s&redirect_uri=%s",
		AuthURL, clientID, url.QueryEscape(redirectURI))

	fmt.Printf("abrindo navegador para autorização do mercado livre...\n%s\n", authURL)
	openMLBrowser(authURL)

	fmt.Println("\napós autorizar, copie o valor do 'code' que aparece na página e cole aqui.")
	fmt.Print("code: ")

	var code string
	fmt.Scan(&code)

	data, err := exchangeMLCode(clientID, clientSecret, code, redirectURI)
	if err != nil {
		fmt.Printf("erro ao trocar código ML: %v\n", err)
		os.Exit(1)
	}
	return data.AccessToken
}

func exchangeMLCode(clientID, clientSecret, code, redirectURI string) (*MLTokenData, error) {
	payload := url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"code":          {code},
		"redirect_uri":  {redirectURI},
	}

	resp, err := http.PostForm(TokenURL, payload)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("ml token error %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, err
	}

	data := &MLTokenData{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresAt:    time.Now().Unix() + tokenResp.ExpiresIn,
	}
	saveMLToken(data)
	return data, nil
}

func refreshMLToken(clientID, clientSecret, refreshToken string) (*MLTokenData, error) {
	payload := url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"refresh_token": {refreshToken},
	}

	resp, err := http.PostForm(TokenURL, payload)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("ml refresh error %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, err
	}

	data := &MLTokenData{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresAt:    time.Now().Unix() + tokenResp.ExpiresIn,
	}
	saveMLToken(data)
	return data, nil
}

func loadMLToken() (*MLTokenData, error) {
	f, err := os.Open(mlTokenFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var data MLTokenData
	err = json.NewDecoder(f).Decode(&data)
	return &data, err
}

func saveMLToken(data *MLTokenData) {
	f, err := os.Create(mlTokenFile)
	if err != nil {
		return
	}
	defer f.Close()
	json.NewEncoder(f).Encode(data)
}

func openMLBrowser(url string) {
	for _, cmd := range []string{"xdg-open", "sensible-browser", "x-www-browser"} {
		attr := &os.ProcAttr{Files: []*os.File{os.Stdin, os.Stdout, os.Stderr}}
		p, err := os.StartProcess("/usr/bin/env", []string{"env", cmd, url}, attr)
		if err == nil {
			go p.Wait()
			return
		}
	}
	fmt.Printf("acesse manualmente: %s\n", url)
}

func (c *Client) doRequest(method, endpoint string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewBuffer(data)
	}

	req, err := http.NewRequest(method, APIURL+endpoint, reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", UserAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("ml api %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

func (c *Client) Get(endpoint string) ([]byte, error) {
	return c.doRequest("GET", endpoint, nil)
}

func (c *Client) Post(endpoint string, payload interface{}) ([]byte, error) {
	return c.doRequest("POST", endpoint, payload)
}

func (c *Client) Put(endpoint string, payload interface{}) ([]byte, error) {
	return c.doRequest("PUT", endpoint, payload)
}

var _ = context.Background
