package add

import (
	"fmt"
	"lightiot/cmd/iotadm/app/util"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

var (
	addLong = `Add an api/ui from a file or from stdin.
Only YAML formats are accepted.
`

	addExample = `# Add an API/UI using the data in ems.yaml
iotadm add -f ems.yaml
`
)

func NewCmdAdd(ioStreams util.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "add",
		Short:   "Add an API/UI from a file or from stdin",
		Long:    addLong,
		Example: addExample,
	}

	cmd.AddCommand(newCmdAddAPI(ioStreams))
	cmd.AddCommand(newCmdAddUI(ioStreams))
	return cmd
}

func nameAndSource(args []string) (string, string, error) {
	if len(args) > 2 {
		return args[0], args[1], fmt.Errorf("expected at most two arguments, unexpected arguments: %v", strings.Join(args[2:], ", "))
	}

	if len(args) == 0 {
		return "", "", fmt.Errorf("exactly one NAME is required")
	}

	if len(args) == 2 {
		return args[0], args[1], nil
	}

	// only one args
	return args[0], filepath.Join(".", args[0]), nil
}

func isLegalName(name string) bool {
	re := regexp.MustCompile("^[a-zA-Z0-9]*$")
	return re.MatchString(name)
}
