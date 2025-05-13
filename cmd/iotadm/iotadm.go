package main

import (
	"lightiot/cmd/iotadm/app"
	"math/rand"
	"os"
	"time"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	command := app.NewEdgeIotCommand(os.Stdin, os.Stdout, os.Stderr)

	if err := command.Execute(); err != nil {
		os.Exit(1)
	}
}
