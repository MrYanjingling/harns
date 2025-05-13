package resource

type ToolType byte

const (
	ToolMQTT ToolType = iota
	ToolTimeSeriesDB
	ToolEdgeIoT
)

var ToolTypeToString = map[ToolType]string{
	ToolMQTT:         "mqtt",
	ToolTimeSeriesDB: "timeSeriesDB",
	ToolEdgeIoT:      "edgeIoT",
}

type ToolDesc struct {
	Name                string
	IsInstalled         bool
	IsRunning           bool
	Version             string
	IsServiceRegistered bool
	ProfilePath         string
}

// ToolsInstaller interface for tools with install and teardown methods.
type ToolsInstaller interface {
	InstallTools() error
	Check() (ToolDesc, error)
	TearDown() error
}

type Installer interface {
	Upgrade() error
	StartNew() error
}
