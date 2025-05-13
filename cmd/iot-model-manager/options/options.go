package options

import (
	"github.com/spf13/pflag"
	"lightiot/cmd/iot-model-manager/config"
	"lightiot/pkg/consumer/job"
	baseoptions "lightiot/pkg/generic/options"
	"lightiot/pkg/generic/runtime"
	"lightiot/pkg/logstorage"
	"lightiot/pkg/model/agent"
	"lightiot/pkg/model/datapointmapping"
	"lightiot/pkg/model/propertysettype"
	"lightiot/pkg/model/thing"
	"time"
)

type Options struct {
	Port            string        `json:"port"`
	Wait            time.Duration `json:"graceful-timeout"`
	InfluxDBUrl     string        `json:"influxdb-url"`
	InfluxDBToken   string        `json:"influxdb-token"`
	MaxInstances    uint16        `json:"max-consumers"`
	JobDelaySec     int64         `json:"job-delay"`
	CacheRefreshSec uint16        `json:"cache-refresh"`
	baseoptions.BaseOptions
}

const (
	_defaultPort            = "32100"
	_defaultWait            = 15 * time.Second
	_defaultInfluxDBUrl     = "http://127.0.0.1:8086"
	_defaultInfluxDBToken   = ""
	_defaultMaxInstances    = 4
	_defaultJobDelaySec     = 5
	_defaultCacheRefreshSec = 180
)

func NewDefaultOptions() *Options {
	return &Options{
		Port:            _defaultPort,
		Wait:            _defaultWait,
		InfluxDBUrl:     _defaultInfluxDBUrl,
		InfluxDBToken:   _defaultInfluxDBToken,
		MaxInstances:    _defaultMaxInstances,
		JobDelaySec:     _defaultJobDelaySec,
		CacheRefreshSec: _defaultCacheRefreshSec,
		BaseOptions:     baseoptions.NewDefaultBaseOptions(),
	}
}

func (o *Options) AddFlags(fs *pflag.FlagSet) {
	// refer to node port assignment https://rancher.com/docs/rancher/v2.x/en/installation/requirements/ports/#commonly-used-ports
	fs.StringVarP(&o.Port, "port", "P", o.Port, "Port exposed")
	fs.DurationVar(&o.Wait, "graceful-timeout", o.Wait, "The duration for which the server gracefully wait for existing connections to finish - e.g. 15s or 1m")
	fs.StringVar(&o.InfluxDBUrl, "influxdb-url", o.InfluxDBUrl, "The InfluxDB URL")
	fs.StringVar(&o.InfluxDBToken, "influxdb-token", o.InfluxDBToken, "The InfluxDB token")
	fs.Uint16Var(&o.MaxInstances, "max-consumers", o.MaxInstances, "The maximum number instances of iot-consumer. It has to be the same with --max-consumers in iot-consumer. The actual number of instances cannot exceeds --max-consumers")
	fs.Int64Var(&o.JobDelaySec, "job-delay", o.JobDelaySec, "The delay seconds to postpone job execution")
	fs.Uint16Var(&o.CacheRefreshSec, "cache-refresh", o.CacheRefreshSec, "The interval in second between refreshing job cache")
}

func (o *Options) Config(stopCh <-chan struct{}) (*config.Config, error) {
	c := &config.Config{}

	logStorage, err := logstorage.NewInfluxDB(stopCh, o.InfluxDBUrl, o.InfluxDBToken)
	if err != nil {
		return nil, err
	}

	pstCh := make(chan runtime.Object)
	ttCh := make(chan runtime.Object)
	tCh := make(chan runtime.Object)
	atCh := make(chan runtime.Object)
	aCh := make(chan runtime.Object)
	mCh := make(chan runtime.Object)
	jm := job.New(stopCh, o.MaxInstances, o.JobDelaySec, true, o.CacheRefreshSec, logStorage)
	var (
		tm   *thing.Manager
		am   *agent.Manager
		mm   *datapointmapping.Manager
		pstm *propertysettype.Manager
	)
	pstm = propertysettype.NewManager(stopCh, pstCh,
		propertysettype.WithForbidDeletionFunc(func(pstID string) bool { return tm.IsReferredByThingType(pstID) }),
	)
	tm = thing.NewManager(stopCh, thing.WithThingTypeChan(ttCh),
		thing.WithThingChan(tCh),
		thing.WithPropertySetTypeManager(pstm),
		thing.WithJobManager(jm),
		thing.WithDeleteAgentFunc(func(thingId string) error { return am.DeleteAgentByThingId(thingId) }),
		thing.WithDeleteMappingFunc(func(thingId, psName string) { mm.DeleteMappingByThingPSName(thingId, psName) }),
	)
	am = agent.NewManager(stopCh, agent.WithAgentTypeChan(atCh),
		agent.WithAgentChan(aCh),
		agent.WithThingManager(tm),
		agent.WithDeleteMappingByAgentIdFunc(func(agentId string) error { return mm.DeleteMappingByAgentId(agentId) }),
	)
	mm = datapointmapping.NewManager(stopCh, datapointmapping.WithMappingChan(mCh), datapointmapping.WithThingManager(tm), datapointmapping.WithAgentManager(am))

	pstm.InitWrite()
	tm.InitWrite()
	am.InitWrite()
	mm.InitWrite()

	c.ModelConfig.PSTMgr = pstm
	c.ModelConfig.ThingMgr = tm
	c.ModelConfig.AgentMgr = am
	c.ModelConfig.DPMMgr = mm

	return c, nil
}
