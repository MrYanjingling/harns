package delete

import (
	"fmt"
	"lightiot/cmd/iotadm/app/resource"
	"lightiot/cmd/iotadm/app/util"
	"strings"

	"github.com/spf13/cobra"
)

var (
	deleteLong = `Delete an api/ui from a file or from stdin.
Only YAML formats are accepted.
`

	deleteExample = `# Delete an API/UI using the data in ems.yaml
iotadm delete -f ems.yaml

# Delete an API/UI using the data from stdin
iotadm delete ui ems
`
)

func NewDeleteCmd(ioStreams util.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete",
		Short:   "Delete an API/UI from a file or from stdin",
		Long:    deleteLong,
		Example: deleteExample,
	}

	cmd.AddCommand(newCmdDeleteAPI(ioStreams))
	cmd.AddCommand(newCmdDeleteUI(ioStreams))
	return cmd
}

func sourceName(args []string) (string, error) {
	if len(args) > 1 {
		return args[0], fmt.Errorf("expected at most one arguments, unexpected arguments: %v", strings.Join(args[1:], ", "))
	} else if len(args) == 0 {
		return "", fmt.Errorf("exactly one NAME is required")
	}
	return args[0], nil
}

func validEdgeIotResource(installation string) error {
	if resource.EdgeIotUIs.Has(installation) {
		return fmt.Errorf(`UI %s is fundamental service provisioned by EdgeIoT. Please delete it via "iotadm uninstall"`, installation)
	} else if resource.EdgeIotAPIs.Has(installation) {
		return fmt.Errorf(`API %s is fundamental service provisioned by EdgeIoT. Please delete it via "iotadm uninstall"`, installation)
	}
	return nil
}
