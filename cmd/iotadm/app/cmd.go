package app

import (
	"io"
	"lightiot/cmd/iotadm/app/add"
	"lightiot/cmd/iotadm/app/delete"
	"lightiot/cmd/iotadm/app/install"
	"lightiot/cmd/iotadm/app/uninstall"
	"lightiot/cmd/iotadm/app/util"
	"lightiot/cmd/iotadm/app/version"

	"github.com/spf13/cobra"
)

var (
	edgeIotLongDescription = `
+----------------------------------------------------------+
| iotadm                                                  |
+----------------------------------------------------------+
| Easily bootstrap a Harns.                                |
+----------------------------------------------------------+
`
)

func NewEdgeIotCommand(in io.Reader, out, err io.Writer) *cobra.Command {
	cmds := &cobra.Command{
		Use:   "iotadm",
		Short: "iotadm: Bootstrap a Harns",
		Long:  edgeIotLongDescription,
	}
	cmds.ResetFlags()

	ioStreams := util.IOStreams{In: in, Out: out, ErrOut: err}

	cmds.AddCommand(install.NewCmdInstall(ioStreams))
	cmds.AddCommand(add.NewCmdAdd(ioStreams))
	cmds.AddCommand(delete.NewDeleteCmd(ioStreams))
	cmds.AddCommand(uninstall.NewCmdUnInstall(ioStreams))
	cmds.AddCommand(version.NewCmdVersion(ioStreams))
	return cmds
}
