package main

import (
    "k8s.io/component-base/logs"
    "lightiot/cmd/iot-event-manager/app"
    "math/rand"
    "os"
    "time"
)

func main() {
    rand.Seed(time.Now().UnixNano())

    cmd := app.NewEventManagerCmd()
    logs.InitLogs()
    defer logs.FlushLogs()

    if err := cmd.Execute(); err != nil {
        os.Exit(1)
    }
}
