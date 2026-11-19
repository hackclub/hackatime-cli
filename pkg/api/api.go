package api

import (
	"context"
	"net/http"
)

const (
	// BaseURL is the base url of the Hackatime api.
	BaseURL = "https://hackatime.hackclub.com/api/hackatime/v1"
	// DefaultTimeoutSecs is the default timeout used for requests to the Hackatime api.
	DefaultTimeoutSecs = 120
)

// Client communicates with the Hackatime api.
type Client struct {
	baseURL string
	client  *http.Client
	// doFunc allows api client options to manipulate request/response handling.
	// default function will be set in constructor.
	//
	// wrapping by api options should be performed as follows:
	//
	//	next := c.doFunc
	//	c.doFunc = func(c *Client, req *http.Request) (*http.Response, error) {
	//		// do something
	//		resp, err := next(c, req)
	//		// do more
	//		return resp, err
	//	}
	doFunc func(c *Client, req *http.Request) (*http.Response, error)
}

// NewClient creates a new Client. Any number of Options can be provided.
func NewClient(baseURL string, opts ...Option) *Client {
	c := &Client{
		baseURL: baseURL,
		client: &http.Client{
			Transport: NewTransport(),
		},
		doFunc: func(c *Client, req *http.Request) (*http.Response, error) {
			req.Header.Set("Accept", "application/json")
			//nolint:gosec // Request target comes from configured API URL and internal request construction.
			return c.client.Do(req)
		},
	}

	for _, option := range opts {
		option(c)
	}

	return c
}

// Do executes c.doFunc(), which in turn allows wrapping c.client.Do() and manipulating
// the request behavior of the api client.
func (c *Client) Do(_ context.Context, req *http.Request) (*http.Response, error) {
	return c.doFunc(c, req)
}
