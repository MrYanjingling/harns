package main

import (
	"k8s.io/component-base/logs"
	"lightiot/cmd/iot-rule-manager/app"
	"math/rand"
	"os"
	"time"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	cmd := app.NewRuleManagerCmd()
	logs.InitLogs()
	defer logs.FlushLogs()

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
