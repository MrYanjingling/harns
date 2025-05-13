package add

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"lightiot/cmd/iotadm/app/resource"
	"lightiot/cmd/iotadm/app/util"
	"lightiot/pkg/client"
	ins "lightiot/pkg/installation"
	v1 "lightiot/pkg/installation/v1"
	"net/http"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/util/sets"
)

var (
	uiLong = `Add a UI service to Harns.`

	uiExample = `# Start an EMS UI
iotadm add ui emsapplication ./emsapplication

# Start an EMS UI and add a customized icon
iotadm add ui --icon=./ems.png emsapplication ./emsapplication

# Start an EMS UI and disable injecting os-bar into UI
iotadm add ui --disable-os-bar emsapplication ./emsapplication

# Start an EMS UI and add a displayName
iotadm add ui --display-name=能源管理 emsapplication ./emsapplication

# Dry run; print the corresponding UI object without creating them
iotadm add ui emsapplication ./emsapplication --dry-run
`
)

type uiOptions struct {
	name          string
	srcPath       string
	icons         []string
	displayName   string
	disabledOsBar bool

	harnsClient client.Client
	util.IOStreams
}

func newCmdAddUI(ioStreams util.IOStreams) *cobra.Command {
	o := newAddUIOptions(ioStreams)
	cmd := &cobra.Command{
		Use:                   "ui NAME RESOURCE",
		DisableFlagsInUseLine: true,
		Short:                 "Add a UI with the specified name",
		Long:                  uiLong,
		Example:               uiExample,
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
			if err := o.complete(args); err != nil {
				return err
			}
			if err := o.validate(); err != nil {
				return err
			}
			return o.run()
		},
	}

	addUIFlags(cmd.Flags(), o)
	return cmd
}

func newAddUIOptions(ioStreams util.IOStreams) *uiOptions {
	return &uiOptions{
		IOStreams: ioStreams,
	}
}

func addUIFlags(fs *flag.FlagSet, o *uiOptions) {
	fs.StringSliceVar(&o.icons, "icon", o.icons, "The icon that is displayed on Harns launchpad.")
	fs.BoolVar(&o.disabledOsBar, "disable-os-bar", o.disabledOsBar, "Disable injecting os-bar into UI.")
	fs.StringVar(&o.displayName, "display-name", o.displayName, "The name that is displayed on Harns launchpad.")
}

func (o *uiOptions) complete(args []string) error {
	if name, src, err := nameAndSource(args); err != nil {
		return err
	} else {
		o.name = name
		o.srcPath = src
	}

	o.harnsClient, _ = util.CreateHttpClient("http://127.0.0.1", "", o.IOStreams)
	return nil
}

func (o *uiOptions) validate() error {
	if _, err := os.Stat(o.srcPath); err != nil {
		_, _ = fmt.Fprintf(o.ErrOut, "Failed to locate path %s\n", o.srcPath)
		return err
	}

	if len(o.icons) == 0 {
		_, _ = fmt.Fprintf(o.Out, "Warning! no icon for UI\n")
		return nil
	}

	var icons []string
	for _, iconPath := range o.icons {
		if _, err := os.Stat(iconPath); err == nil {
			_ = filepath.Walk(iconPath, func(path string, info fs.FileInfo, err error) error {
				ext := filepath.Ext(path)
				if legalIconExt.Has(ext) {
					icons = append(icons, path)
				}
				return nil
			})
		}
	}
	if len(icons) == 0 {
		_, _ = fmt.Fprintf(o.ErrOut, "Failed to load icons\n")
		return fmt.Errorf("falied loading icons")
	}
	o.icons = icons

	filter := struct {
		Name string `json:"name"`
	}{
		Name: o.name,
	}
	if installation := resource.GetInstallation(o.harnsClient, filter, o.IOStreams); installation != nil {
		return fmt.Errorf("installation %s is already in use", o.name)
	}
	return nil
}

func (o *uiOptions) run() error {
	if err := util.CopyFileWithRecursive(o.srcPath, resource.GetEdgeIoTAppFilesPath(), o.IOStreams); err != nil {
		return err
	}

	if err := util.ChownFile(resource.EdgeIoTUser, resource.GetEdgeIoTAppFilesPath(), o.IOStreams); err != nil {
		return err
	}
	ei := o.convert2EdgeInstallation()

	i := ei.Convert()

	i.Components[0].Uri = fmt.Sprintf("/%s", ei.Name)

	resp, err := o.harnsClient.Post("/api/installation/v1/installations", http.Header{"Content-Type": []string{"application/json"}}, nil, i)
	if err != nil {
		_, _ = fmt.Fprintf(o.ErrOut, "Failed to create installation %s\n", i.Name)
		return err
	}
	defer client.Drain(resp)
	var insRes ins.Installation
	if resp.StatusCode == http.StatusCreated {
		if err = json.NewDecoder(resp.Body).Decode(&insRes); err != nil {
			_, _ = fmt.Fprintf(o.ErrOut, "Failed to parse installation\n")
			return err
		}
		if len(insRes.ID) > 0 {
			for _, icon := range o.icons {
				if err = resource.UploadImage(icon, insRes, o.IOStreams); err != nil {
					_, _ = fmt.Fprintf(o.ErrOut, "Failed to upload image %s for installation %s\n", icon, i.Name)
				}
			}
		}
	}
	_, _ = fmt.Fprintf(o.Out, "Successful to add UI %s\n", o.name)
	return nil
}

func (o *uiOptions) convert2EdgeInstallation() *resource.EdgeInstallation {
	var osBar v1.EnableOsBar
	if !o.disabledOsBar {
		osBar = true
	}
	if len(o.displayName) == 0 {
		o.displayName = o.name
	}
	return &resource.EdgeInstallation{
		Name:        o.name,
		DisplayName: o.displayName,
		Type:        v1.InstallationTypeUI,
		Version:     "v1",
		Replicas:    1,
		Resource:    "",
		OsBar:       &osBar,
	}
}

var (
	legalIconExt = sets.NewString(".png", ".jpeg", ".gif")
)
