package delete

import (
	"fmt"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/sets"
	"lightiot/cmd/iotadm/app/resource"
	"lightiot/cmd/iotadm/app/util"
	"lightiot/pkg/client"
	ins "lightiot/pkg/installation"
	v1 "lightiot/pkg/installation/v1"
	"net/http"
	"os"
	"path/filepath"
)

var (
	uiLong = `Delete a UI service from Harns.`

	uiExample = `# Delete EMS UI
iotadm delete ui emsapplication
`
)

type uiOptions struct {
	name         string
	installation *ins.Installation
	harnsClient  client.Client
	util.IOStreams
}

func newCmdDeleteUI(ioStreams util.IOStreams) *cobra.Command {
	o := newAddUIOptions(ioStreams)
	cmd := &cobra.Command{
		Use:                   "ui NAME",
		DisableFlagsInUseLine: true,
		Short:                 "Delete a UI with the specified name",
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
	return cmd
}

func newAddUIOptions(ioStreams util.IOStreams) *uiOptions {
	return &uiOptions{
		IOStreams: ioStreams,
	}
}

func (o *uiOptions) complete(args []string) error {
	if name, err := sourceName(args); err != nil {
		return err
	} else {
		o.name = name
	}
	o.harnsClient, _ = util.CreateHttpClient("http://127.0.0.1", "", o.IOStreams)
	return nil
}

func (o *uiOptions) validate() error {
	filter := struct {
		Name string              `json:"name"`
		Type v1.InstallationType `json:"type"`
	}{
		Name: o.name,
		Type: v1.InstallationTypeUI,
	}
	installation := resource.GetInstallation(o.harnsClient, filter, o.IOStreams)
	if installation == nil {
		return fmt.Errorf("UI %s dose not exist", o.name)
	}
	if err := validEdgeIotResource(o.name); err != nil {
		return err
	}
	o.installation = installation
	return nil
}

func (o *uiOptions) run() error {
	if images, err := resource.GetUiImages(o.harnsClient, o.installation.ID, o.IOStreams); err != nil {
		return err
	} else if len(images) > 0 {
		for _, image := range images {
			if err := resource.DeleteUiImages(o.harnsClient, image.Name, o.installation.ID, o.IOStreams); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(o.Out, "Deleted image %s for installation %s\n", image.Name, o.installation.Name)
		}
	}
	if resp, err := o.harnsClient.Delete(fmt.Sprintf("/api/installation/v1/installations/%s", o.installation.ID)); err != nil {
		return err
	} else if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to delete %s UI installation", o.installation.Name)
	}
	_, _ = fmt.Fprintf(o.Out, "Deleted %s UI installation\n", o.installation.Name)

	uiPath := filepath.Join(resource.GetEdgeIoTAppFilesPath(), o.installation.Name)
	if _, err := os.Stat(uiPath); err == nil {
		if err := os.RemoveAll(uiPath); err != nil {
			return fmt.Errorf("failed to remove %s file path", o.installation.Name)
		}
		_, _ = fmt.Fprintf(o.Out, "Removed %s file\n", o.installation.Name)
	}
	_, _ = fmt.Fprintf(o.Out, "Successful to delete UI %s\n", o.name)
	return nil
}

func (o *uiOptions) convert2EdgeInstallation() *resource.EdgeInstallation {
	return &resource.EdgeInstallation{
		Name:     o.name,
		Type:     v1.InstallationTypeUI,
		Version:  "v1",
		Replicas: 1,
		Resource: "",
	}
}

var (
	legalIconExt = sets.NewString(".png", ".jpeg", ".gif")
)
