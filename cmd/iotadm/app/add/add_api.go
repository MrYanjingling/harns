package add

import (
	"fmt"
	"io"
	"io/fs"
	"lightiot/cmd/iotadm/app/resource"
	"lightiot/cmd/iotadm/app/util"
	"lightiot/pkg/client"
	"lightiot/pkg/generic"
	v1 "lightiot/pkg/installation/v1"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
)

var (
	apiLong = `Add an API service to Harns.`

	apiExample = `# Start an EMS API service
iotadm add api emsmodelmanager ./ems-model-manager

# Start an EMS API service and let the container expose port 32700
iotadm add api --port=32700 emsmodelmanager ./ems-model-manager

# Start an EMS API service and set environment variables "--SPRING_ACTIVE_PROFILE=prod" and "LOGGING_LEVEL_ROOT=INFO" in the service
iotadm add api emsscmsvc ./ems-model-svc.jar --arg="--SPRING_ACTIVE_PROFILE=prod" --arg="--LOGGING_LEVEL_ROOT=INFO"

# Dry run; print the corresponding API service object without creating them
iotadm add api emsmodelmanager ./ems-model-manager --dry-run
`
)

type apiOptions struct {
	name    string
	srcPath string
	port    int32
	command string
	arg     []string
	args    map[string]interface{}
	daemon  bool
	paths   []string
	actions []string

	harnsClient client.Client
	util.IOStreams
}

func newCmdAddAPI(ioStreams util.IOStreams) *cobra.Command {
	o := newAddAPIOptions(ioStreams)
	cmd := &cobra.Command{
		Use:                   "api NAME RESOURCE",
		DisableFlagsInUseLine: true,
		Short:                 "Add an API service with the specified name",
		Long:                  apiLong,
		Example:               apiExample,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if util.GetOSInterface() == nil {
				_, _ = fmt.Fprintf(ioStreams.ErrOut, "Only Linux is supported now. Other OS supporting is comming\n")
				return fmt.Errorf("unsupported os")
			} else if !util.IsUserRoot() {
				_, _ = fmt.Fprintf(ioStreams.ErrOut, "Should run as system administrator(root)\n")
				return fmt.Errorf("insufficient permission")
			} else {
				return nil
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.complete(cmd, args); err != nil {
				return err
			}
			if err := o.validate(); err != nil {
				return err
			}
			return o.run()
		},
	}
	addAPIFlags(cmd.Flags(), o)

	return cmd
}

func newAddAPIOptions(ioStreams util.IOStreams) *apiOptions {
	return &apiOptions{
		IOStreams: ioStreams,
	}
}

func addAPIFlags(fs *flag.FlagSet, o *apiOptions) {
	fs.Int32Var(&o.port, "port", o.port, "The port that this service exposes (if null, Harns shall assign an available port for this service).")
	fs.StringVar(&o.command, "command", o.command, "Set start command of this service (if null, the resource base name is the default command).")
	fs.StringSliceVar(&o.arg, "arg", o.arg, "Set arguments on the command line (can specify multiple or separate arguments with commas: key1=val1,key2=val2).")
	fs.BoolVar(&o.daemon, "daemon", o.daemon, "If true, this API service serves as a daemon. Meanwhile, it doesn't serve as HTTP server.")
	fs.StringSliceVar(&o.paths, "basePath", o.paths, "The base paths that this service provides.")
	fs.StringSliceVar(&o.actions, "action", o.actions, "The actions that the base path provides. Must be \"GET\",\"POST\",\"PUT\",\"PATCH\" or \"DELETE\" (if null and path is not null, all actions are appended).")
}

func (o *apiOptions) complete(cmd *cobra.Command, args []string) error {
	if name, src, err := nameAndSource(args); err != nil {
		return err
	} else {
		o.name = name
		o.srcPath = src
	}

	if !o.daemon {
		if o.port == 0 {
			o.port = int32(generic.GetFreePort(generic.StartPort))
		}

		o.harnsClient, _ = util.CreateHttpClient("http://127.0.0.1", "", o.IOStreams)

		if len(o.paths) == 0 {
			o.paths = []string{fmt.Sprintf("/api/%s/v1", o.name)}
		}

		if len(o.paths) != 0 && len(o.actions) == 0 {
			o.actions = []string{"GET", "POST", "DELETE", "PUT", "PATCH"}
		}
	}

	base := map[string]interface{}{}

	for _, arg := range o.arg {
		if err := util.ParseInto(arg, base); err != nil {
			return fmt.Errorf("failed parsing --arg data")
		}
	}
	o.args = base

	return nil
}

func (o *apiOptions) validate() error {
	if o.daemon {
		if o.port != 0 {
			return fmt.Errorf("--daemon and --port are mutually exclusive")
		}

		if len(o.paths) != 0 {
			return fmt.Errorf("--daemon and --basePath are mutually exclusive")
		}

		if len(o.actions) != 0 {
			return fmt.Errorf("--daemon and --action are mutually exclusive")
		}
	} else {
		conn, err := net.DialTimeout("tcp", net.JoinHostPort("", strconv.Itoa(int(o.port))), 5*time.Second)
		if conn != nil {
			defer conn.Close()
		}
		if err == nil {
			return fmt.Errorf("port %d is already in use", o.port)
		}

		filter := struct {
			Name string `json:"name"`
		}{
			Name: o.name,
		}
		if ins := resource.GetInstallation(o.harnsClient, filter, o.IOStreams); ins != nil {
			return fmt.Errorf("installation %s is already in use", o.name)
		}

	}

	des, _ := os.ReadDir(util.GetSystemSvcPath())
	for _, de := range des {
		svcUnit := de.Name()
		if strings.EqualFold(o.name, svcUnit[:len(svcUnit)-len(filepath.Ext(svcUnit))]) {
			return fmt.Errorf("service unit %s is already in use", o.name)
		}
	}

	if !isLegalName(o.name) {
		return fmt.Errorf("API name only consists of aplphanumeric characters")
	}

	if len(o.name) > 32 {
		return fmt.Errorf("API name length exceeds 32")
	}

	if _, err := os.Stat(o.srcPath); err != nil {
		_, _ = fmt.Fprintf(o.ErrOut, "Failed to locate path %s\n", o.srcPath)
		return err
	}

	for _, a := range o.actions {
		if _, ok := v1.EndpointActionFromString[a]; !ok {
			return fmt.Errorf("invalid actoin %s. Must be \"GET\",\"POST\",\"PUT\",\"PATCH\" or \"DELETE\"", a)
		}
	}

	return nil
}

func (o *apiOptions) run() error {
	if fi, _ := os.Stat(o.srcPath); fi.IsDir() {
		if err := util.CopyFileWithRecursive(o.srcPath, resource.GetEdgeIoTExecutableFilesPath(), o.IOStreams); err != nil {
			return err
		}
		o.replaceCommand(true)
		// TODO: ugly code
		if err := util.ChownFile(resource.EdgeIoTUser, resource.GetEdgeIoTExecutableFilesPath(), o.IOStreams); err != nil {
			return err
		}
		if err := util.AddExecutablePermission(resource.GetEdgeIoTExecutableFilesPath(), o.IOStreams); err != nil {
			return err
		}
	} else {
		dstPath := filepath.Join(resource.GetEdgeIoTExecutableFilesPath(), filepath.Base(o.srcPath))
		if err := util.CopyFile(o.srcPath, dstPath); err != nil {
			_, _ = fmt.Fprintf(o.ErrOut, "Failed to copy file from %s to %s\n", o.srcPath, resource.GetEdgeIoTExecutableFilesPath())
			return err
		}
		o.replaceCommand(false)
		if err := util.ChownFile(resource.EdgeIoTUser, dstPath, o.IOStreams); err != nil {
			return err
		}
		if err := util.AddExecutablePermission(dstPath, o.IOStreams); err != nil {
			return err
		}
	}
	// todo should support multi instance
	ei := o.convert2EdgeInstallation()
	if !o.daemon {
		ei.StartCommand.Args["--port"] = o.port
	}

	if err := o.registerSystemdSvc(ei); err != nil {
		return err
	}

	if o.daemon {
		_, _ = fmt.Fprintf(o.Out, "Successful to add API %s\n", o.name)
		return nil
	}

	i := ei.Convert()

	i.Components[0].Uri = fmt.Sprintf("http://127.0.0.1:%d", o.port)
	for _, path := range o.paths {
		for _, a := range o.actions {
			action := v1.EndpointActionFromString[a]
			i.Components[0].Endpoints = append(i.Components[0].Endpoints, v1.Endpoint{
				Path:   path,
				Action: action,
			})
		}
	}

	resp, err := o.harnsClient.Post("/api/installation/v1/installations", http.Header{"Content-Type": []string{"application/json"}}, nil, i)
	if err != nil {
		_, _ = fmt.Fprintf(o.ErrOut, "Failed to create installation %s\n", i.Name)
		return err
	}
	defer client.Drain(resp)

	if resp.StatusCode != http.StatusCreated {
		errMsg, _ := io.ReadAll(resp.Body)
		_, _ = fmt.Fprintf(o.ErrOut, "Failed to register installation %s\n", string(errMsg))
		return fmt.Errorf("failed registering installation %s to Harns", i.Name)
	}

	_, _ = fmt.Fprintf(o.Out, "Registered installation %s to Harns\nSuccessful to add API %s\n", o.name, o.name)
	return nil
}

func (o *apiOptions) replaceCommand(srcIsDir bool) error {
	return filepath.Walk(o.srcPath, func(path string, info fs.FileInfo, err error) error {
		if !info.IsDir() {
			index := strings.Index(o.command, info.Name())
			if index != -1 {
				i := index
				for ; i >= 0; i-- {
					if unicode.IsSpace(rune(o.command[i])) {
						break
					}
				}
				old := o.command[i+1:index] + info.Name()
				execPath := resource.GetEdgeIoTExecutableFilesPath()
				if !srcIsDir {
					execPath = filepath.Join(resource.GetEdgeIoTExecutableFilesPath(), info.Name())
				}
				new := strings.Replace(path, o.srcPath, execPath, -1)
				o.command = strings.Replace(o.command, old, new, -1)
			}
		}
		return nil
	})
}

func (o *apiOptions) convert2EdgeInstallation() *resource.EdgeInstallation {
	ei := &resource.EdgeInstallation{
		Name:        o.name,
		DisplayName: o.name,
		Type:        v1.InstallationTypeAPI,
		Version:     "v1",
		Replicas:    1,
		Resource:    "",
		ConfigFile:  "",
		Daemon:      o.daemon,
		StartCommand: &resource.Command{
			Command: o.command,
			Args:    o.args,
		},
		Endpoints: make([]*resource.Endpoint, len(o.paths)),
	}

	for i, p := range o.paths {
		ei.Endpoints[i] = &resource.Endpoint{
			Path:    p,
			Actions: make([]v1.EndpointAction, len(o.actions)),
		}
		for j, a := range o.actions {
			ei.Endpoints[i].Actions[j] = v1.EndpointActionFromString[a]
		}
	}

	return ei
}

func (o *apiOptions) registerSystemdSvc(ei *resource.EdgeInstallation) error {
	svcUnit := ei.Convert3rdSvcUnit()

	if err := util.GenerateFiles(svcUnit.File, o.IOStreams); err != nil {
		return err
	}

	if err := util.ChownFile(resource.EdgeIoTUser, filepath.Join(resource.GetEdgeIoTConfigPath(), fmt.Sprintf("%s.service", svcUnit.SvcName)), o.IOStreams); err != nil {
		return err
	}

	if err := util.RegisterSystemd(svcUnit, o.IOStreams); err != nil {
		return err
	}
	if err := util.StartSvc(svcUnit); err != nil {
		return err
	}

	return nil
}
