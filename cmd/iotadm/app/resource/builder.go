package resource

import (
	"fmt"
	"lightiot/cmd/iotadm/app/util"
	"lightiot/pkg/generic"
	v1 "lightiot/pkg/installation/v1"
	"path/filepath"
	"time"

	flag "github.com/spf13/pflag"
)

type FilenameOptions struct {
	Filenames []string
	Recursive bool
}

func AddFilenameOptionFlags(fs *flag.FlagSet, options *FilenameOptions) {
	fs.StringSliceVarP(&options.Filenames, "filename", "f", options.Filenames, "Filename, directory, or URL to files to use to create the resource.")
	fs.BoolVarP(&options.Recursive, "recursive", "R", options.Recursive, "Process the directory used in -f, --filename recursively. Useful when you want to manage related manifests organized within the same directory.")
}

type EdgeInstallation struct {
	Name         string              `yaml:"name"`
	DisplayName  string              `yaml:"displayName"`
	Type         v1.InstallationType `yaml:"type"`
	Version      string              `yaml:"version"`
	Replicas     int                 `yaml:"replicas"`
	Daemon       bool                `yaml:"daemon"`
	Resource     string              `yaml:"resource"`
	ConfigFile   string              `yaml:"configFile,omitempty"`
	StartCommand *Command            `yaml:"startCommand,omitempty"`
	Endpoints    []*Endpoint         `yaml:"endpoints,omitempty"`
	OsBar        *v1.EnableOsBar     `yaml:"osBar,omitempty"`
}

type Command struct {
	Command string                 `yaml:"command"`
	Args    map[string]interface{} `yaml:"args"`
}

type Endpoint struct {
	Path    string              `yaml:"path"`
	Actions []v1.EndpointAction `yaml:"actions"`
}

func (ei *EdgeInstallation) Convert() *v1.Installation {
	i := &v1.Installation{
		Name:        ei.Name,
		DisplayName: ei.DisplayName,
		ProviderId:  "main",
		Version:     "v1",
		Type:        ei.Type,
		Start:       time.Now(),
		End:         time.Now().Add(365 * 24 * time.Hour),
		Components: []*v1.Component{
			{
				Name:      ei.Name,
				Endpoints: make([]v1.Endpoint, 0),
			},
		},
	}
	if i.Type == v1.InstallationTypeAPI {
		if installationName, ok := EdgeIoTAPIInstallations[ei.Name]; ok {
			i.Name = installationName
			i.DisplayName = installationName
		}

		if ep, ok := EdgeIoTAPIEndpoints[ei.Name]; ok {
			path := fmt.Sprintf("/%s/%s/%s", v1.InstallationTypeToString[i.Type], i.Name, i.Version)
			if len(ep.Path) != 0 {
				path += ep.Path
			}
			for _, a := range ep.Actions {
				i.Components[0].Endpoints = append(i.Components[0].Endpoints, v1.Endpoint{
					Path:   path,
					Action: a,
				})
			}
		}
	}

	if i.Type == v1.InstallationTypeUI {
		if installationName, ok := EdgeIoTUIInstallations[ei.Name]; ok {
			i.Name = installationName
		}
		if dn, ok := EdgeIoTUIDisplayNames[i.Name]; ok {
			i.DisplayName = dn
		}
		i.Components[0].Endpoints = append(i.Components[0].Endpoints, v1.Endpoint{
			Path:   filepath.Join(GetEdgeIoTAppFilesPath(), ei.Name, "dist"),
			Action: v1.EndpointActionGet,
			OsBar:  ei.OsBar,
		})
	}

	return i
}

func (ei *EdgeInstallation) ConvertEdgeIotSvcUnit(svcFileData interface{}) *util.SvcUnit {
	svcFileName := fmt.Sprintf("%s.service", ei.Name)
	if ei.Replicas > 1 {
		svcFileName = fmt.Sprintf("%s@.service", ei.Name)
	}
	return &util.SvcUnit{
		SvcName:     ei.Name,
		SvcFileName: svcFileName,
		Replicas:    ei.Replicas,
		Path:        GetEdgeIoTConfigPath(),
		File: []util.FileDesc{
			{
				Path:     filepath.Join(GetEdgeIoTConfigPath(), svcFileName),
				Template: EdgeIoTSvcTmpl,
				Data:     svcFileData,
			},
		},
	}
}

func (ei *EdgeInstallation) Convert3rdSvcUnit() *util.SvcUnit {
	startCmd := ei.StartCommand.Command

	for k, v := range ei.StartCommand.Args {
		startCmd += fmt.Sprintf(" %s=%v", k, v)
	}

	svcFileName := fmt.Sprintf("%s.service", ei.Name)
	if ei.Replicas > 1 {
		svcFileName = fmt.Sprintf("%s@.service", ei.Name)
	}

	return &util.SvcUnit{
		SvcName:     ei.Name,
		SvcFileName: svcFileName,
		Replicas:    ei.Replicas,
		Path:        GetEdgeIoTConfigPath(),
		File: []util.FileDesc{
			{
				Path:     filepath.Join(GetEdgeIoTConfigPath(), svcFileName),
				Template: apiSvcTmpl,
				Data: util.ApiSvcConfig{
					StartCmd: startCmd,
					User:     EdgeIoTUser,
					Group:    EdgeIoTGroup,
				},
			},
		},
	}
}

type FreePort struct {
	next int
}

func NewFreePort(start int) *FreePort {
	return &FreePort{
		next: generic.GetFreePort(start),
	}
}

func (fp *FreePort) Next() int {
	port := fp.next
	fp.next = generic.GetFreePort(port + 1)
	return port
}
