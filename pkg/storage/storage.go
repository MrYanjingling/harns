package storage

import (
	gruntime "lightiot/pkg/generic/runtime"
	"time"
)

type StoreGroup byte

const (
	StoreGroupModel StoreGroup = iota
	StoreGroupEvent
	StoreGroupData
	StoreGroupInstall
	StoreGroupNotify
	StoreGroupControl
)

var (
	StoreGroupToString = map[StoreGroup]string{
		StoreGroupModel:   "model",
		StoreGroupEvent:   "event",
		StoreGroupData:    "data",
		StoreGroupInstall: "installation",
		StoreGroupNotify:  "notification",
		StoreGroupControl: "control",
	}
	StoreGroupFromString = map[string]StoreGroup{
		"model":        StoreGroupModel,
		"event":        StoreGroupEvent,
		"data":         StoreGroupData,
		"installation": StoreGroupInstall,
		"notification": StoreGroupNotify,
		"control":      StoreGroupControl,
	}
)

// resources
const (
	// model
	PropertySetTypes = "propertysettypes"
	ThingTypes       = "thingtypes"
	Things           = "things"
	AgentTypes       = "agenttypes"
	Agents           = "agents"
	Mappings         = "mappings"

	// event
	EventTypes = "eventtypes"

	// rule
	Rules = "rules"

	// installation
	Installations = "installations"

	// notification
	Recipients   = "recipients"
	MessageTmpls = "messagetmpls"
	Templates    = "templates"
	Messages     = "messages"
	Servers      = "servers"

	// control
	CommandTypes = "commandtypes"
)

type Getter interface {
	Get(key string) (interface{}, error)
}

type Lister interface {
	List(key string) (interface{}, error)
}

type Creater interface {
	Create(key string, obj interface{}) (interface{}, error)
}

type Updater interface {
	Update(key, version string, obj interface{}) (interface{}, error)
}

type Deleter interface {
	Delete(key, version string) (interface{}, error)
}

type Watcher interface {
	Watch(stopCh <-chan struct{}, key string, rev string) (chan *Event, error)
}

type Storage interface {
	Getter
	Lister
	Creater
	Updater
	Deleter
	Watcher
}

type EventType int8

const (
	Create EventType = iota
	Update
	Remove
)

func (et EventType) String() string {
	return []string{
		"Create",
		"Update",
		"Remove",
	}[et]
}

type Event struct {
	Type EventType
	Data interface{}
}

type FileInfo struct {
	Path    string
	ModTime time.Time
}

type PersistResult struct {
	Err   error
	Saved gruntime.Object
}
