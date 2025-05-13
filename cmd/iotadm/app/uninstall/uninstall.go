package uninstall

import (
	"fmt"
	"lightiot/cmd/iotadm/app/install"
	"lightiot/cmd/iotadm/app/resource"
	"lightiot/cmd/iotadm/app/util"

	"github.com/spf13/cobra"
)

var (
	unInstallLongDescription = `Uninstall Harns.
`
	unInstallExample = `# Uninstall Harns
iotadm uninstall
`
)

func NewCmdUnInstall(ioStreams util.IOStreams) *cobra.Command {
	toolsList := make(map[string]resource.ToolsInstaller)
	cmd := &cobra.Command{
		Use:     "uninstall",
		Short:   "Uninstall Harns",
		Long:    unInstallLongDescription,
		Example: unInstallExample,
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
			if err := AddToolsList(toolsList, ioStreams); err != nil {
				return err
			}

			if err := TearExecute(toolsList); err != nil {
				return err
			}

			_, _ = fmt.Fprintf(ioStreams.Out, "Successful to uninstall all\n")
			return nil
		},
	}

	addUnInstallOtherFlags(cmd)
	return cmd
}

func addUnInstallOtherFlags(cmd *cobra.Command) {
}

func AddToolsList(toolsList map[string]resource.ToolsInstaller, ioStreams util.IOStreams) error {
	toolsList[util.TOOLEDGIOT] = &install.EdgeIotInstTool{
		Common: util.Common{IOStreams: ioStreams},
	}

	toolsList[util.TOOLINFLUXDB] = &install.TimeSeriesDBInstTool{
		Common: util.Common{IOStreams: ioStreams},
	}
	toolsList[util.TOOLMOSQUITTO] = &install.MqttInstTool{
		Common: util.Common{IOStreams: ioStreams},
	}
	return nil
}

func TearExecute(toolsList map[string]resource.ToolsInstaller) error {
	if err := toolsList[util.TOOLEDGIOT].TearDown(); err != nil {
		return err
	}
	for name, tool := range toolsList {
		if name != util.TOOLEDGIOT {
			err := tool.TearDown()
			if err != nil {
				return err
			}
		}
	}
	return nil
}
