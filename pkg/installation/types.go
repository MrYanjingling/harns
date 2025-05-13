package installation

import (
	"lightiot/pkg/generic/meta"
	v1 "lightiot/pkg/installation/v1"
	"time"
)

type Installation struct {
	meta.ObjectMeta
	DisplayName   string              `json:"displayName"`
	ProviderId    string              `json:"providerId"`
	Version       string              `json:"version"`
	Type          v1.InstallationType `json:"type"`
	Start         time.Time           `json:"start"`
	End           time.Time           `json:"end"`
	Configuration *Configuration      `json:"configuration,omitempty"`
	Images        *Images             `json:"images,omitempty"`
	Components    []*Component        `json:"components,omitempty"`
}

type Configuration struct {
	Roles []string `json:"roles"`
}

type Images struct {
	Href string `json:"href"`
}

type Image struct {
	Name string `json:"name"`
	Href string `json:"href"`
}

type Endpoint struct {
	Path   string            `json:"path"`
	Action v1.EndpointAction `json:"action"`
	OsBar  *v1.EnableOsBar   `json:"osBar,omitempty"`
}

type Component struct {
	Name      string      `json:"name"`
	Uri       string      `json:"uri"`
	Endpoints []*Endpoint `json:"endpoints"`
}
