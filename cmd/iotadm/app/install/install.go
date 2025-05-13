package install

import (
	"fmt"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"lightiot/cmd/iotadm/app/resource"
	"lightiot/cmd/iotadm/app/util"
	"lightiot/pkg/client"
	"lightiot/pkg/generic"
	"os"

	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/spf13/cobra"
)

var (
	installLongDescription = `Install Harns from a package or from URL.
`

	// TODO usage
	installExample = `# Install base harns
iotadm install

# Install Harns with offering
iotadm install --offerings=event
`
)

func NewCmdInstall(ioStreams util.IOStreams) *cobra.Command {
	o := NewOptions(ioStreams)
	cmd := &cobra.Command{
		Use:     "install",
		Short:   "Install Harns",
		Long:    installLongDescription,
		Example: installExample,
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
			if err := o.validate(); err != nil {
				return err
			}
			c, err := o.config()
			if err != nil {
				return err
			}
			if err = run(c, ioStreams); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(ioStreams.Out, "Please try to access Harns via your browser\n")
			return nil
		},
	}

	addInstallOtherFlags(cmd, o)

	cmd.AddCommand(NewCmdPreflight(ioStreams))

	return cmd
}

func run(c *config, ioStreams util.IOStreams) error {
	if err := util.CreateUser(resource.EdgeIoTUser, resource.EdgeIoTGroup, ioStreams); err != nil {
		return err
	}

	if err := Execute(c.tools); err != nil {
		return err
	}
	return nil
}

func Execute(toolsList map[resource.ToolType]resource.ToolsInstaller) error {
	for name, tool := range toolsList {
		if name != resource.ToolEdgeIoT {
			if err := tool.InstallTools(); err != nil {
				return err
			}
		}
	}
	return toolsList[resource.ToolEdgeIoT].InstallTools()
}

type installOptions struct {
	srcFile      string
	bases        []string
	offerings    []string
	ignores      []string
	TSDBUrl      string
	TSDBToken    string
	fileName     resource.FilenameOptions
	ClientCAFile string

	util.IOStreams
}

type config struct {
	tools map[resource.ToolType]resource.ToolsInstaller
}

func addInstallOtherFlags(cmd *cobra.Command, o *installOptions) {
	cmd.Flags().StringVar(&o.srcFile, "pkg-url", o.srcFile, "Filename or URL to files of Harns.")
	cmd.Flags().StringSliceVar(&o.offerings, "offerings", o.offerings, "The offerings provide more functionalities, the available options are: 'event', 'rule', 'notification', 'control' and 'rollup'.")
	cmd.Flags().StringSliceVar(&o.ignores, "ignores", o.ignores, "The services are ignored during this installation.")
	cmd.Flags().StringVar(&o.TSDBUrl, "tsdb-url", "http://127.0.0.1:8086", "The TSDB URL")
	cmd.Flags().StringVar(&o.TSDBToken, "tsdb-token", o.TSDBToken, "The TSDB token")
	cmd.Flags().StringVar(&o.ClientCAFile, "client-ca-file", o.ClientCAFile, ""+
		"If set, any request presenting a client certificate signed by one of the authorities in the client-ca-file "+
		"is authenticated with an identity corresponding to the CommonName of the client certificate.")
	resource.AddFilenameOptionFlags(cmd.Flags(), &o.fileName)
}

func NewOptions(ioStreams util.IOStreams) *installOptions {
	return &installOptions{
		IOStreams: ioStreams,
	}
}

func (o *installOptions) validate() error {
	if diffs := sets.NewString(o.offerings...).Difference(resource.EdgeIoTOfferings); diffs.Len() != 0 {
		return fmt.Errorf("invalid offerings %v", diffs)
	}

	if diffs := sets.NewString(o.ignores...).Difference(resource.EdgeIoTAllSvcs); diffs.Len() != 0 {
		return fmt.Errorf("invalid ignores %v", diffs)
	}
	return nil
}

func (o *installOptions) config() (*config, error) {
	c := &config{
		tools: make(map[resource.ToolType]resource.ToolsInstaller),
	}

	var (
		srcPath string
		err     error
	)
	if srcPath, err = os.Getwd(); err != nil {
		return nil, err
	} else if len(o.srcFile) != 0 {
		_, _ = fmt.Fprintf(o.Out, "--pkg-url is not supported now\n")
	}

	svcs := resource.EdgeIoTBaseSvcs.Difference(sets.NewString(o.ignores...)).Insert(o.offerings...)

	if svcs.Has(resource.ToolTypeToString[resource.ToolMQTT]) {
		c.tools[resource.ToolMQTT] = &MqttInstTool{
			Common:  util.Common{IOStreams: o.IOStreams},
			SrcPath: srcPath,
		}
		svcs.Delete(resource.ToolTypeToString[resource.ToolMQTT])
	}

	if svcs.Has(resource.ToolTypeToString[resource.ToolTimeSeriesDB]) {
		influxdbClient, err := o.createHttpClient(o.TSDBUrl)
		if err != nil {
			return nil, err
		}
		c.tools[resource.ToolTimeSeriesDB] = &TimeSeriesDBInstTool{
			Common:         util.Common{IOStreams: o.IOStreams},
			SrcPath:        srcPath,
			InfluxdbClient: influxdbClient,
		}
		svcs.Delete(resource.ToolTypeToString[resource.ToolTimeSeriesDB])
	}

	if svcs.Len() != 0 {
		// TODO: support https
		httpClient, err := o.createHttpClient("http://127.0.0.1")
		if err != nil {
			return nil, err
		}
		if len(o.TSDBToken) == 0 {
			o.TSDBToken = resource.TSDBToken
		}
		c.tools[resource.ToolEdgeIoT] = &EdgeIotInstTool{
			Common:         util.Common{IOStreams: o.IOStreams},
			SrcPath:        srcPath,
			Services:       svcs.UnsortedList(),
			Client:         httpClient,
			freePort:       resource.NewFreePort(generic.StartPort),
			bidirection:    svcs.Has(resource.EdgeIoTControl),
			enableRollup:   svcs.Has(resource.EdgeIotRollup),
			enableRule:     svcs.Has(resource.EdgeIoTRule),
			influxdbClient: influxdb2.NewClient(o.TSDBUrl, o.TSDBToken),
			apiSvcUrls:     make(map[string]string),
		}
	}

	return c, nil
}

func (o *installOptions) createHttpClient(endpoint string) (client.Client, error) {
	return util.CreateHttpClient(endpoint, o.ClientCAFile, o.IOStreams)
}
