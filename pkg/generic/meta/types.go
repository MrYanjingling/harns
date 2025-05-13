package meta

import (
	"net/url"
	"time"
)

type TypeMeta struct {
	Resource string
}

type ObjectMeta struct {
	Tenant  string    `json:"tenant"`
	Name    string    `json:"name"`
	ID      string    `json:"id"`
	Version string    `json:"eTag"`
	ModTime time.Time `json:"-"`
}

type CreateOptions struct {
	Query url.Values
}

type GetOptions struct {
	Version string
	Query   url.Values
}

type ListOptions struct {
	Filter map[string]interface{}
	Query  url.Values
}

type UpdateOptions struct {
	Version string
	Query   url.Values
}

type DeleteOptions struct {
	Version string
	Query   url.Values
}
