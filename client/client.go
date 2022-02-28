// Copyright (c) 2019-2022, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the LICENSE.md file
// distributed with the sources of this project regarding your rights to use or distribute this
// software.

package client

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// errUnsupportedProtocolScheme is returned when an unsupported protocol scheme is encountered.
var errUnsupportedProtocolScheme = errors.New("unsupported protocol scheme")

// normalizeURL parses rawURL, and ensures the path component is terminated with a separator.
func normalizeURL(rawURL string) (*url.URL, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("%w %s", errUnsupportedProtocolScheme, u.Scheme)
	}

	// Ensure path is terminated with a separator, to prevent url.ResolveReference from stripping
	// the final path component of BaseURL when constructing request URL from a relative path.
	if !strings.HasSuffix(u.Path, "/") {
		u.Path += "/"
	}

	return u, nil
}

// clientOptions describes the options for a Client.
type clientOptions struct {
	baseURL     string
	bearerToken string
	userAgent   string
	httpClient  *http.Client
}

// Option are used to populate co.
type Option func(co *clientOptions) error

// OptBaseURL sets the base URL of the build server to url.
func OptBaseURL(url string) Option {
	return func(co *clientOptions) error {
		co.baseURL = url
		return nil
	}
}

// OptBearerToken sets the bearer token to include in the "Authorization" header of each request.
func OptBearerToken(token string) Option {
	return func(co *clientOptions) error {
		co.bearerToken = token
		return nil
	}
}

// OptUserAgent sets the HTTP user agent to include in the "User-Agent" header of each request.
func OptUserAgent(agent string) Option {
	return func(co *clientOptions) error {
		co.userAgent = agent
		return nil
	}
}

// OptHTTPClient sets the client to use to make HTTP requests.
func OptHTTPClient(c *http.Client) Option {
	return func(co *clientOptions) error {
		co.httpClient = c
		return nil
	}
}

// Client describes the client details.
type Client struct {
	// Base URL of the service.
	BaseURL *url.URL
	// Auth token to include in the Authorization header of each request (if supplied).
	AuthToken string
	// User agent to include in each request (if supplied).
	UserAgent string
	// HTTPClient to use to make HTTP requests.
	HTTPClient *http.Client
}

const defaultBaseURL = "https://build.sylabs.io/"

// NewClient returns a Client configured according to opts.
//
// By default, the Sylabs Build Service is used. To override this behaviour, use OptBaseURL.
//
// By default, requests are not authenticated. To override this behaviour, use OptBearerToken.
func NewClient(opts ...Option) (*Client, error) {
	co := clientOptions{
		baseURL:    defaultBaseURL,
		httpClient: http.DefaultClient,
	}

	// Apply options.
	for _, opt := range opts {
		if err := opt(&co); err != nil {
			return nil, fmt.Errorf("%w", err)
		}
	}

	c := Client{
		AuthToken:  co.bearerToken,
		UserAgent:  co.userAgent,
		HTTPClient: co.httpClient,
	}

	// Normalize base URL.
	u, err := normalizeURL(co.baseURL)
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}
	c.BaseURL = u

	return &c, nil
}

// newRequest returns a new Request given a method, relative path, query, and optional body.
func (c *Client) newRequest(method, path string, body io.Reader) (r *http.Request, err error) {
	u := c.BaseURL.ResolveReference(&url.URL{
		Path: strings.TrimPrefix(path, "/"), // trim leading separator as path is relative.
	})

	r, err = http.NewRequest(method, u.String(), body)
	if err != nil {
		return nil, err
	}
	c.setRequestHeaders(r.Header)

	return r, nil
}

func (c *Client) setRequestHeaders(h http.Header) {
	if v := c.AuthToken; v != "" {
		h.Set("Authorization", fmt.Sprintf("BEARER %s", v))
	}
	if v := c.UserAgent; v != "" {
		h.Set("User-Agent", v)
	}
}
