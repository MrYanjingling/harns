package main

import (
	"k8s.io/component-base/logs"
	"lightiot/cmd/iot-model-manager/app"
	"math/rand"
	"os"
	"time"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	cmd := app.NewModelManagerCmd()
	logs.InitLogs()
	defer logs.FlushLogs()

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
