package util

import (
	"fmt"
)

type RpmOS struct {
}

func (rpm *RpmOS) InstallGateway() error {
	// open the 80 port of the firewall.
	cmd := NewCommand(fmt.Sprintf("firewall-cmd --zone=public --add-port=%d/tcp --permanent && firewall-cmd --reload", GatewayTCPPort))
	return cmd.Exec()
}
