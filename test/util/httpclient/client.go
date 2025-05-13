package httpclient

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	defaultConnectTimeout      = 10 * time.Second
	defaultResponseReadTimeout = 60 * time.Second
)

type TestClient interface {
	Put(path string, header http.Header, obj string) (*http.Response, error)
	Post(path string, header http.Header, query map[string]interface{}, obj string) (*http.Response, error)
	PostForm(path string, header http.Header, obj map[string]interface{}) (*http.Response, error)
	Get(path string, header http.Header, query *url.Values) (*http.Response, error)
	Delete(path string, header http.Header, query *url.Values) (*http.Response, error)
}

type testClient struct {
	url      url.URL
	client   *http.Client
	basePath string
}

func CreateTestClient(portocol, host, port, basePath string) TestClient {
	u, _ := url.Parse(fmt.Sprintf("%s://%s:%s", portocol, host, port))
	return NewTestClient(u, basePath)
}

func NewTestClient(url *url.URL, basePath string) TestClient {
	return NewTestClientWithTimeOut(url, basePath, defaultResponseReadTimeout, defaultConnectTimeout)
}

func NewTestClientWithTimeOut(url *url.URL, basePath string, readTimeout, connTimeout time.Duration) TestClient {
	tp := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   connTimeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   connTimeout,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return &testClient{
		url: *url,
		client: &http.Client{
			Timeout:   readTimeout,
			Transport: tp,
		},
		basePath: basePath,
	}
}

func (c *testClient) Put(path string, header http.Header, json string) (*http.Response, error) {
	u := c.url
	u.Path = c.basePath + path
	req, err := http.NewRequest(http.MethodPut, u.String(), strings.NewReader(json))
	if err != nil {
		return nil, err
	}
	req.Header = header
	return c.client.Do(req)
}

func (c *testClient) Post(path string, header http.Header, query map[string]interface{}, obj string) (*http.Response, error) {
	u := c.url
	u.Path = c.basePath + path
	if len(query) != 0 {
		q := u.Query()
		for k, v := range query {
			q.Set(k, fmt.Sprintf("%v", v))
		}
		u.RawQuery = q.Encode()
	}
	req, err := http.NewRequest(http.MethodPost, u.String(), strings.NewReader(obj))
	if err != nil {
		return nil, err
	}
	req.Header = header
	return c.client.Do(req)
}

func (c *testClient) PostForm(path string, header http.Header, obj map[string]interface{}) (*http.Response, error) {
	u := c.url
	u.Path = c.basePath + path
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

func (c *testClient) Get(path string, header http.Header, query *url.Values) (*http.Response, error) {
	u := c.url
	u.Path = c.basePath + path
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

func (c *testClient) Delete(path string, header http.Header, query *url.Values) (*http.Response, error) {
	u := c.url
	u.Path = c.basePath + path
	if query != nil {
		u.RawQuery = query.Encode()
	}
	req, err := http.NewRequest(http.MethodDelete, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header = header
	return c.client.Do(req)
}

// Drain consumes and closes the response's body to make sure that the
// HTTP client can reuse existing connections.
func Drain(r *http.Response) {
	_, _ = io.Copy(io.Discard, r.Body)
	_ = r.Body.Close()
}
