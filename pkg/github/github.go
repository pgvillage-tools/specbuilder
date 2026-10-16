// Package github implements a minimal client for the GitHub REST API, reading repository and release info.
package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// DefaultTimeout is the timeout for GitHub API requests.
const DefaultTimeout = 10 * time.Second

// Repo holds the repository information that is relevant for spec files.
type Repo struct {
	Name        string
	Description string
	License     string
}

// Asset is a downloadable file attached to a release.
type Asset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
}

// Release holds the release information that is relevant for spec files.
type Release struct {
	TagName string  `json:"tag_name"`
	Body    string  `json:"body"`
	Assets  []Asset `json:"assets"`
}

// Client reads info from the GitHub API.
type Client struct {
	HTTPClient *http.Client
	// Token is an optional GitHub token, used to raise the API rate limit.
	Token string
	// BaseURL returns the API base url for a site. Defaults to https://api.<site>.
	BaseURL func(site string) string
}

// NewClient returns a Client with sane defaults.
func NewClient(token string) *Client {
	return &Client{
		HTTPClient: &http.Client{Timeout: DefaultTimeout},
		Token:      token,
	}
}

func (c *Client) baseURL(site string) string {
	if c.BaseURL != nil {
		return c.BaseURL(site)
	}
	return "https://api." + site
}

func (c *Client) get(url string, target any) (err error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: DefaultTimeout}
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("requesting %s: %w", url, err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("closing response body from %s: %w", url, closeErr)
		}
	}()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("requesting %s: unexpected status %s", url, resp.Status)
	}
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("decoding response from %s: %w", url, err)
	}
	return nil
}

// GetRepo reads repository info like name, description and license.
func (c *Client) GetRepo(site, organization, repository string) (Repo, error) {
	var data struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		License     *struct {
			SPDXID string `json:"spdx_id"`
		} `json:"license"`
	}
	url := fmt.Sprintf("%s/repos/%s/%s", c.baseURL(site), organization, repository)
	if err := c.get(url, &data); err != nil {
		return Repo{}, err
	}
	if data.Name == "" {
		return Repo{}, fmt.Errorf("missing 'name' in response from %s", url)
	}
	repo := Repo{Name: data.Name, Description: data.Description}
	if data.License != nil {
		repo.License = data.License.SPDXID
	}
	return repo, nil
}

// GetRelease reads release info. release is 'latest', or another release path like 'tags/v1.0.0'.
func (c *Client) GetRelease(site, organization, repository, release string) (Release, error) {
	var data Release
	url := fmt.Sprintf("%s/repos/%s/%s/releases/%s", c.baseURL(site), organization, repository, release)
	if err := c.get(url, &data); err != nil {
		return Release{}, err
	}
	return data, nil
}
