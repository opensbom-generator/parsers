// SPDX-License-Identifier: Apache-2.0

package helper

import (
	"net/http"
	"net/url"
	"time"
)

type Client struct {
	HTTP    *http.Client
	BaseURL string
}

// NewClient
// todo: complete proper client settings
func NewClient(baseURL string) *Client {
	return &Client{
		HTTP: &http.Client{
			Timeout: time.Second * 5,
		},
		BaseURL: baseURL,
	}
}

// IsValidURL ...
func (c *Client) ParseURL(uri string) *url.URL {
	u, err := url.Parse(uri)
	if err == nil && u.Scheme == "" {
		u.Scheme = "http"
	}

	return u
}

// CheckURL ...
func (c *Client) CheckURL(url string) bool {
	r, err := c.HTTP.Get(url)
	if err != nil {
		return false
	}
	defer r.Body.Close()

	return r.StatusCode == http.StatusOK
}

// Get makes a GET request to the specified url and returns the response.
func (c *Client) Get(url string) (*http.Response, error) {
	r, err := c.HTTP.Get(url)
	if err != nil {
		return nil, err
	}

	return r, nil
}
