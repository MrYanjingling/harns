package options

import (
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	flag "github.com/spf13/pflag"
	"lightiot/cmd/iot-consumer/config"
	"lightiot/pkg/consumer/instance"
	"lightiot/pkg/consumer/job"
	baseoptions "lightiot/pkg/generic/options"
	"lightiot/pkg/logstorage"
	"time"
)

const (
	_defaultInfluxDBUrl         = "http://127.0.0.1:8086"
	_defaultInfluxDBToken       = ""
	_defaultHeartbeatInterval   = 20 * 1000
	_defaultRetrieveJobInterval = 4
	_defaultMaxInstances        = 4
	_defaultRollupWorkers       = 4
	_defaultRollupBufferSize    = 1024
	_defaultDeleteWorkers       = 1
	_defaultDeleteBufferSize    = 1024
	_defaultWait                = time.Second * 15
)

var (
	_defaultJobGroups = job.GetGroups()
)

type Options struct {
	InfluxDBUrl         string        `json:"influxdb-url"`
	InfluxDBToken       string        `json:"influxdb-token"`
	JobGroups           []string      `json:"job-groups"`
	HeartbeatInterval   int64         `json:"heartbeat-interval"`
	RetrieveJobInterval int64         `json:"retrieve-job-interval"`
	MaxInstances        uint16        `json:"max-consumers"`
	RollupWorkers       uint16        `json:"rollup-workers"`
	RollupBufferSize    uint32        `json:"rollup-buffer-size"`
	DeleteWorkers       uint16        `json:"delete-workers"`
	DeleteBufferSize    uint32        `json:"delete-buffer-size"`
	Wait                time.Duration `json:"graceful-timeout"`
	baseoptions.BaseOptions
}

func NewDefaultOptions() *Options {
	return &Options{
		InfluxDBUrl:         _defaultInfluxDBUrl,
		InfluxDBToken:       _defaultInfluxDBToken,
		JobGroups:           _defaultJobGroups,
		HeartbeatInterval:   _defaultHeartbeatInterval,
		RetrieveJobInterval: _defaultRetrieveJobInterval,
		MaxInstances:        _defaultMaxInstances,
		RollupWorkers:       _defaultRollupWorkers,
		RollupBufferSize:    _defaultRollupBufferSize,
		DeleteWorkers:       _defaultDeleteWorkers,
		DeleteBufferSize:    _defaultDeleteBufferSize,
		Wait:                _defaultWait,
		BaseOptions:         baseoptions.NewDefaultBaseOptions(),
	}
}

func (o *Options) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&o.InfluxDBUrl, "influxdb-url", o.InfluxDBUrl, "The InfluxDB URL")
	fs.StringVar(&o.InfluxDBToken, "influxdb-token", o.InfluxDBToken, "The InfluxDB token")
	fs.StringSliceVar(&o.JobGroups, "job-groups", o.JobGroups, "The job group that this consumer consumes")
	fs.Int64Var(&o.HeartbeatInterval, "heartbeat-interval", o.HeartbeatInterval, "The heartbeat interval. Unit is millisecond (ms). If the heartbeat is lost more than twenty times the --heartbeat-interval, this instance will be removed")
	fs.Int64Var(&o.RetrieveJobInterval, "retrieve-job-interval", o.RetrieveJobInterval, "The interval in second that the consumer waits for if there is no job assigned to it")
	fs.Uint16Var(&o.MaxInstances, "max-consumers", o.MaxInstances, "The maximum number instances of consumer. The actual number of instances cannot exceed --max-consumers")
	fs.Uint16Var(&o.RollupWorkers, "rollup-workers", o.RollupWorkers, "The number of workers (threads) which consume the rollup job")
	fs.Uint32Var(&o.RollupBufferSize, "rollup-buffer-size", o.RollupBufferSize, "The size of buffer that caches rollup job")
	fs.Uint16Var(&o.DeleteWorkers, "delete-workers", o.DeleteWorkers, "The number of workers (threads) which consume the delete job")
	fs.Uint32Var(&o.DeleteBufferSize, "delete-buffer-size", o.DeleteBufferSize, "The size of buffer that caches delete job")
	fs.DurationVar(&o.Wait, "graceful-timeout", o.Wait, "the duration for which the server gracefully wait for existing connections to finish - e.g. 15s or 1m")
}

func (o *Options) Config(stopCh <-chan struct{}) (*config.Config, error) {
	c := &config.Config{
		InstanceConfig: instance.Config{
			HeartbeatInterval:   o.HeartbeatInterval,
			RetrieveJobInterval: o.RetrieveJobInterval,
			MaxInstances:        o.MaxInstances,
			RollupWorkers:       o.RollupWorkers,
			RollupBufferSize:    o.RollupBufferSize,
			DeleteWorkers:       o.DeleteWorkers,
			DeleteBufferSize:    o.DeleteBufferSize,
		},
	}

	c.JobGroups = make([]job.Group, len(o.JobGroups))

	for i, v := range o.JobGroups {
		c.JobGroups[i] = job.GroupFromString[v]
	}

	if store, err := logstorage.NewInfluxDB(stopCh, o.InfluxDBUrl, o.InfluxDBToken); err != nil {
		return c, err
	} else {
		c.InstanceConfig.LogStorage = store
	}

	c.InstanceConfig.DataWorkerConfig.Client = influxdb2.NewClientWithOptions(o.InfluxDBUrl,
		o.InfluxDBToken,
		influxdb2.DefaultOptions().SetPrecision(time.Millisecond))

	return c, nil
}
