package install

import (
	"fmt"
	"lightiot/cmd/iotadm/app/resource"
	"lightiot/cmd/iotadm/app/util"

	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
)

const (
	preflightExample = `
		# Run pre-flight checks for iotadm install.
		iotadm install preflight --config harns-config.yaml
		`
)

func NewCmdPreflight(ioStreams util.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "preflight",
		Short:   "Run pre-flight checks",
		Long:    "Run pre-flight checks for iotadm install.",
		Example: preflightExample,
		Run: func(cmd *cobra.Command, args []string) {

			// TODO get os version

			tools := []resource.ToolsInstaller{
				&TimeSeriesDBInstTool{},
				&MqttInstTool{},
				&EdgeIotInstTool{},
				// &ServiceInstall{},
			}
			execute(tools)
		},
	}
	return cmd
}

func execute(toolList []resource.ToolsInstaller) {
	for _, tool := range toolList {
		desc, err := tool.Check()
		if err != nil {
			klog.Warningln("Check tool failed, error:", err)
			continue
		}
		printToolDesc(&desc)
	}
}

func printToolDesc(desc *resource.ToolDesc) {
	fmt.Printf("Tool %s status\n", desc.Name)
	fmt.Printf("Installed: %t\n", desc.IsInstalled)
	if desc.IsInstalled != true {
		return
	}
	fmt.Printf("Version: %s\n", desc.Version)
	fmt.Printf("Running: %t\n", desc.IsRunning)
	fmt.Printf("Profile: %s\n", desc.ProfilePath)
}
