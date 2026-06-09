package shopify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	APIVersion = "2025-07"
	UserAgent  = "vinyligo/0.1"
)

type Client struct {
	shopName      string
	accessToken   string
	locationID    string
	publicationID string
	httpClient    *http.Client
}

func NewClient(shopName, accessToken, locationID, publicationID string) *Client {
	return &Client{
		shopName:      shopName,
		accessToken:   accessToken,
		locationID:    locationID,
		publicationID: publicationID,
		httpClient:    &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) baseURL() string {
	return fmt.Sprintf("https://%s.myshopify.com/admin/api/%s", c.shopName, APIVersion)
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

	req, err := http.NewRequest(method, c.baseURL()+endpoint, reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Shopify-Access-Token", c.accessToken)
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
		return nil, fmt.Errorf("shopify api %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

func (c *Client) Post(endpoint string, payload interface{}) ([]byte, error) {
	return c.doRequest(http.MethodPost, endpoint, payload)
}

func (c *Client) GraphQL(query string, variables interface{}) ([]byte, error) {
	payload := map[string]interface{}{
		"query":     query,
		"variables": variables,
	}
	return c.doRequest(http.MethodPost, "/graphql.json", payload)
}
