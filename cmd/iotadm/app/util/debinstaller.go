package util

import (
	"fmt"
)

type DebOS struct {
}

func (deb *DebOS) InstallGateway() error {
	cmd := NewCommand(fmt.Sprintf("ufw allow %d", GatewayTCPPort))
	return cmd.Exec()
}
