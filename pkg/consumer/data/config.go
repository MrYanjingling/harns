package data

import (
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
)

type Config struct {
	Client       influxdb2.Client
}
