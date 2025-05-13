package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"k8s.io/klog/v2"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type client struct {
	url    url.URL
	client *http.Client
}

// https://medium.com/@nate510/don-t-use-go-s-default-http-client-4804cb19f779
func NewClient(url *url.URL, timeout time.Duration, tp *http.Transport) *client {
	return &client{
		url: *url,
		client: &http.Client{
			Timeout:   timeout,
			Transport: tp,
		},
	}
}

func (c *client) Put(path string, header http.Header, obj interface{}) (*http.Response, error) {
	u := c.url // copy the related data to prevent race condition
	u.Path = path
	buf := &bytes.Buffer{}
	if err := json.NewEncoder(buf).Encode(obj); err != nil {
		klog.V(3).InfoS("Failed to marshal", "err", err)
	}
	req, err := http.NewRequest(http.MethodPut, u.String(), buf)
	if err != nil {
		return nil, err
	}
	req.Header = header
	return c.client.Do(req)
}

func (c *client) Post(path string, header http.Header, query map[string]interface{}, obj interface{}) (*http.Response, error) {
	u := c.url // copy the related data to prevent race condition
	u.Path = path
	buf := &bytes.Buffer{}
	if err := json.NewEncoder(buf).Encode(obj); err != nil {
		klog.V(3).InfoS("Failed to marshal", "err", err)
	}
	if len(query) != 0 {
		q := u.Query()
		for k, v := range query {
			q.Set(k, fmt.Sprintf("%v", v))
		}
		u.RawQuery = q.Encode()
	}
	req, err := http.NewRequest(http.MethodPost, u.String(), buf)
	if err != nil {
		return nil, err
	}
	req.Header = header
	return c.client.Do(req)
}

func (c *client) PostForm(path string, header http.Header, obj map[string]interface{}) (*http.Response, error) {
	u := c.url // copy the related data to prevent race condition
	u.Path = path
	data := url.Values{}
	for k, v := range obj {
		data.Set(k, fmt.Sprintf("%v", v))
	}
	req, err := http.NewRequest(http.MethodPost, u.String(), strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	header.Add("Content-Type", "application/x-www-form-urlencoded")
	header.Add("Content-Length", strconv.Itoa(len(data.Encode())))
	req.Header = header
	return c.client.Do(req)
}

func (c *client) Get(path string, header http.Header, query *url.Values) (*http.Response, error) {
	u := c.url // copy the related data to prevent race condition
	u.Path = path
	if query != nil {
		u.RawQuery = query.Encode()
	}
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header = header
	return c.client.Do(req)
}

func (c *client) Delete(path string) (*http.Response, error) {
	u := c.url // copy the related data to prevent race condition
	u.Path = path
	req, err := http.NewRequest(http.MethodDelete, u.String(), nil)
	if err != nil {
		return nil, err
	}
	return c.client.Do(req)
}

// Drain consumes and closes the response's body to make sure that the
// HTTP client can reuse existing connections.
func Drain(r *http.Response) {
	_, _ = io.Copy(io.Discard, r.Body)
	_ = r.Body.Close()
}
