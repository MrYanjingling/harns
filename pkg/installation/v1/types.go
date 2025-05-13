package v1

import "time"

type InstallationType byte
type EnableOsBar bool

const (
	InstallationTypeUI InstallationType = iota
	InstallationTypeAPP
	InstallationTypeAPI
)

var InstallationTypeToString = map[InstallationType]string{
	InstallationTypeUI:  "ui",
	InstallationTypeAPP: "app",
	InstallationTypeAPI: "api",
}

var InstallationTypeFromString = map[string]InstallationType{
	"ui":  InstallationTypeUI,
	"app": InstallationTypeAPP,
	"api": InstallationTypeAPI,
}

func (it InstallationType) String() string {
	return InstallationTypeToString[it]
}

type EndpointAction byte

const (
	EndpointActionGet EndpointAction = iota
	EndpointActionPost
	EndpointActionPut
	EndpointActionPatch
	EndpointActionDelete
	EndpointActionHead
	EndpointActionConnect
	EndpointActionOptions
	EndpointActionTrace
)

var EndpointActionToString = map[EndpointAction]string{
	EndpointActionGet:     "GET",
	EndpointActionPost:    "POST",
	EndpointActionPut:     "PUT",
	EndpointActionPatch:   "PATCH",
	EndpointActionDelete:  "DELETE",
	EndpointActionHead:    "HEAD",
	EndpointActionConnect: "CONNECT",
	EndpointActionOptions: "OPTIONS",
	EndpointActionTrace:   "TRACE",
}

var EndpointActionFromString = map[string]EndpointAction{
	"GET":     EndpointActionGet,
	"POST":    EndpointActionPost,
	"PUT":     EndpointActionPut,
	"PATCH":   EndpointActionPatch,
	"DELETE":  EndpointActionDelete,
	"HEAD":    EndpointActionHead,
	"CONNECT": EndpointActionConnect,
	"OPTIONS": EndpointActionOptions,
	"TRACE":   EndpointActionTrace,
}

type Installation struct {
	Name          string           `json:"name" binding:"required,alphanum,min=1,max=32"`
	DisplayName   string           `json:"displayName" binding:"required,min=1,max=16"`
	ProviderId    string           `json:"providerId"`
	Version       string           `json:"version" binding:"required,min=2,max=4"`
	Type          InstallationType `json:"type" binding:"gte=0"`
	Start         time.Time        `json:"start" binding:"required"`
	End           time.Time        `json:"end" binding:"required"`
	Configuration *Configuration   `json:"configuration,omitempty"`
	Components    []*Component     `json:"components,omitempty" binding:"dive"`
}

type Configuration struct {
	Roles []string `json:"roles"`
}

type Endpoint struct {
	Path   string         `json:"path" binding:"required"`
	Action EndpointAction `json:"action" binding:"gte=0"`
	OsBar  *EnableOsBar   `json:"osBar,omitempty"`
}

type Component struct {
	Name      string     `json:"name" binding:"required"`
	Uri       string     `json:"uri" binding:"required"`
	Endpoints []Endpoint `json:"endpoints" binding:"required,dive"`
}
