package delete

import (
	"fmt"
	"github.com/spf13/cobra"
	"lightiot/cmd/iotadm/app/resource"
	"lightiot/cmd/iotadm/app/util"
	"lightiot/pkg/client"
	ins "lightiot/pkg/installation"
	v1 "lightiot/pkg/installation/v1"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var (
	apiLong = `Delete an API service from Harns.`

	apiExample = `# Delete EMS API service
iotadm delete api emsmodelmanager
`
)

type apiOptions struct {
	name         string
	installation *ins.Installation
	harnsClient  client.Client
	util.IOStreams
}

func newCmdDeleteAPI(ioStreams util.IOStreams) *cobra.Command {
	o := newAddAPIOptions(ioStreams)
	cmd := &cobra.Command{
		Use:                   "api NAME",
		DisableFlagsInUseLine: true,
		Short:                 "Delete an API service with the specified name",
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

func newAddAPIOptions(ioStreams util.IOStreams) *apiOptions {
	return &apiOptions{
		IOStreams: ioStreams,
	}
}

func (o *apiOptions) complete(args []string) error {
	if name, err := sourceName(args); err != nil {
		return err
	} else {
		o.name = name
	}
	o.harnsClient, _ = util.CreateHttpClient("http://127.0.0.1", "", o.IOStreams)
	return nil
}

func (o *apiOptions) validate() error {
	filter := struct {
		Name string              `json:"name"`
		Type v1.InstallationType `json:"type"`
	}{
		Name: o.name,
		Type: v1.InstallationTypeAPI,
	}
	installation := resource.GetInstallation(o.harnsClient, filter, o.IOStreams)
	if installation == nil {
		_, _ = fmt.Fprintf(o.Out, "Warning! API %s installation is not exist\n", o.name)
	}
	if err := validEdgeIotResource(o.name); err != nil {
		return err
	}
	o.installation = installation
	return nil
}

func (o *apiOptions) run() error {
	if o.installation != nil {
		resp, err := o.harnsClient.Delete(fmt.Sprintf("/api/installation/v1/installations/%s", o.installation.ID))
		if err != nil {
			_, _ = fmt.Fprintf(o.ErrOut, "Failed to access iot installation manager\n")
			return err
		}
		defer client.Drain(resp)
		if resp.StatusCode == http.StatusOK {
			_, _ = fmt.Fprintf(o.ErrOut, "Deleted %s api installation from Harns\n", o.name)
		}
	}

	replicas := resource.DefaultReplicas
	command := util.NewCommand(fmt.Sprintf(`systemctl --type=service | grep %s`, o.name))
	if err := command.Exec(); err != nil {
		return fmt.Errorf("service %s process not exist", o.name)
	} else {
		replicas = len(strings.Split(command.GetStdOut(), "\n"))
	}

	// todo should consider multi instance
	service := fmt.Sprintf("%s.service", o.name)
	svcUnitPath := filepath.Join(resource.GetEdgeIoTConfigPath(), service)
	svcUnit := util.NewSvcUnit(o.name, resource.GetEdgeIoTConfigPath(), replicas, nil)
	if _, err := os.Stat(svcUnitPath); err == nil {
		if err := util.NewCommand(fmt.Sprintf("systemctl stop %s", service)).Exec(); err != nil {
			return fmt.Errorf("failed to stop %s service", o.name)
		}
		_, _ = fmt.Fprintf(o.Out, "Stoped %s service\n", o.name)
		if err := util.CancelSystemd(svcUnit, o.IOStreams); err != nil {
			return err
		}
		if err := os.RemoveAll(svcUnitPath); err != nil {
			return fmt.Errorf("failed to remove %s service unit file", o.name)
		}
		_, _ = fmt.Fprintf(o.Out, "Removed %s service unit file\n", o.name)
		// todo how to delete artifact?
	} else {
		_, _ = fmt.Fprintf(o.Out, "Warning! Service not stop and remove on server\n")
	}
	_, _ = fmt.Fprintf(o.Out, "Successful to delete API %s\n", o.name)
	return nil
}
