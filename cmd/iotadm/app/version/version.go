package version

import (
	"encoding/json"
	"fmt"
	"lightiot/cmd/iotadm/app/util"
	"lightiot/pkg/version"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// Options is a struct to support version command
type Options struct {
	// ClientOnly bool
	Short  bool
	Output string

	util.IOStreams
}

// NewOptions returns initialized Options
func NewOptions(ioStreams util.IOStreams) *Options {
	return &Options{
		IOStreams: ioStreams,
	}
}

// NewCmdVersion returns a cobra command for fetching versions
func NewCmdVersion(ioStreams util.IOStreams) *cobra.Command {
	o := NewOptions(ioStreams)
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version of iotadm",
		Run: func(cmd *cobra.Command, args []string) {
			util.CheckErr(o.Validate())
			util.CheckErr(o.Run())
		},
	}
	// cmd.Flags().BoolVar(&o.ClientOnly, "client", o.ClientOnly, "If true, shows client version only (no server required).")
	cmd.Flags().BoolVar(&o.Short, "short", o.Short, "If true, print just the version number.")
	cmd.Flags().StringVarP(&o.Output, "output", "o", o.Output, "One of 'yaml' or 'json'.")
	return cmd
}

// Validate validates the provided options
func (o *Options) Validate() error {
	if o.Output != "" && o.Output != "yaml" && o.Output != "json" {
		return errors.New(`--output must be 'yaml' or 'json'`)
	}

	return nil
}

// Run executes version command
func (o *Options) Run() error {
	v := version.Get()

	switch o.Output {
	case "":
		if o.Short {
			_, _ = fmt.Fprintf(o.Out, "iotadm Version: %s\n", v.GitVersion)
		} else {
			_, _ = fmt.Fprintf(o.Out, "iotadm Version: %s\n", fmt.Sprintf("%#v", v))
		}
	case "yaml":
		marshalled, err := yaml.Marshal(&v)
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintln(o.Out, string(marshalled))
	case "json":
		marshalled, err := json.MarshalIndent(&v, "", "  ")
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintln(o.Out, string(marshalled))
	default:
		// There is a bug in the program if we hit this case.
		// However, we follow a policy of never panicking.
		return fmt.Errorf("VersionOptions were not validated: --output=%q should have been rejected", o.Output)
	}
	return nil
}
