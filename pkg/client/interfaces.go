package client

import (
	"net/http"
	"net/url"
)

type Client interface {
	Put(path string, header http.Header, obj interface{}) (*http.Response, error)
	Post(path string, header http.Header, query map[string]interface{}, obj interface{}) (*http.Response, error)
	PostForm(path string, header http.Header, obj map[string]interface{}) (*http.Response, error)
	Get(path string, header http.Header, query *url.Values) (*http.Response, error)
	Delete(path string) (*http.Response, error)
}
