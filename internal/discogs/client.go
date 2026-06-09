package discogs

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	BaseURL   = "https://api.discogs.com"
	UserAgent = "vinyligo/0.1"
)

type Client struct {
	token      string
	httpClient *http.Client
}

type SearchResult struct {
	Results    []Release  `json:"results"`
	Pagination Pagination `json:"pagination"`
}

type Pagination struct {
	Pages int `json:"pages"`
	Items int `json:"items"`
}

type Release struct {
	ID      int      `json:"id"`
	Title   string   `json:"title"`
	Year    string   `json:"year"`
	Country string   `json:"country"`
	Catno   string   `json:"catno"`
	Formats []Format `json:"formats"`
	Labels  []string `json:"label"`
	Artists []Artist `json:"artists"`
	Genres  []string `json:"genre"`
}

type ReleaseDetails struct {
	ID        int      `json:"id"`
	Title     string   `json:"title"`
	Year      int      `json:"year"`
	Country   string   `json:"country"`
	Formats   []Format `json:"formats"`
	Labels    []Label  `json:"labels"`
	Artists   []Artist `json:"artists"`
	Genres    []string `json:"genres"`
	Tracklist []Track  `json:"tracklist"`
}

type Format struct {
	Name         string   `json:"name"`
	Qty          string   `json:"qty"`
	Descriptions []string `json:"descriptions"`
}

type Label struct {
	Name  string `json:"name"`
	Catno string `json:"catno"`
}

type Artist struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	ResourceURL string `json:"resource_url"`
}

type ArtistDetails struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Profile string `json:"profile"`
}

type Track struct {
	Position string `json:"position"`
	Title    string `json:"title"`
	Duration string `json:"duration"`
}

func NewClient(token string) *Client {
	return &Client{
		token:      token,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) doRequest(endpoint string) ([]byte, error) {
	req, err := http.NewRequest("GET", BaseURL+endpoint, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Authorization", "Discogs token="+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("discogs api %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

func (c *Client) Search(query string) ([]Release, error) {
	params := url.Values{}
	params.Set("q", query)
	params.Set("type", "release")

	body, err := c.doRequest("/database/search?" + params.Encode())
	if err != nil {
		return nil, err
	}

	var result SearchResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return result.Results, nil
}

func (c *Client) GetRelease(releaseID int) (*ReleaseDetails, error) {
	body, err := c.doRequest(fmt.Sprintf("/releases/%d", releaseID))
	if err != nil {
		return nil, err
	}

	var release ReleaseDetails
	if err := json.Unmarshal(body, &release); err != nil {
		return nil, err
	}

	return &release, nil
}

func (c *Client) GetArtist(artistID int) (*ArtistDetails, error) {
	body, err := c.doRequest(fmt.Sprintf("/artists/%d", artistID))
	if err != nil {
		return nil, err
	}

	var artist ArtistDetails
	if err := json.Unmarshal(body, &artist); err != nil {
		return nil, err
	}

	return &artist, nil
}

func (r *ReleaseDetails) GetAlbumDuration() float64 {
	totalSeconds := 0.0
	for _, track := range r.Tracklist {
		if track.Duration == "" {
			continue
		}
		parts := strings.Split(track.Duration, ":")
		if len(parts) != 2 {
			continue
		}
		minutes, err1 := strconv.ParseFloat(parts[0], 64)
		seconds, err2 := strconv.ParseFloat(parts[1], 64)
		if err1 != nil || err2 != nil {
			continue
		}
		totalSeconds += minutes*60 + seconds
	}
	return totalSeconds / 60.0
}
