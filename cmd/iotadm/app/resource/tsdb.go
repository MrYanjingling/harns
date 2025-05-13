package resource

import (
	"lightiot/cmd/iotadm/app/util"
	"path/filepath"
	"text/template"
)

const (
	tsdb                   = "influxdb"
	tsdbSrc                = "/dependencies/influxdb/"
	tsdbDst                = "/influxdb/"
	tsdbSystemd            = "/influxdb/systemd/"
	tsdbSystemdSvc         = "influxdb.service"
	tsdbExecutableFiles    = "/influxdb/bin/"
	tsdbExecutableFile     = "/influxdb/bin/influxd"
	tsdbConfigPath         = "/influxdb/conf"
	tsdbConfig             = "/influxdb/conf/config.yaml"
	TSDBOrg                = "main"
	TSDBUsername           = "harns"
	TSDBPassword           = "IoT@Edge"
	TSDBToken              = "c6jGYUCinwzeTWdeUh32"
	DataRawBucket          = "data-raw"
	RetentionPeriodSeconds = 7 * 24 * 60 * 60
)

var (
	TSDBConfTmpl, _ = template.New("tsdbconf").Parse(`query-concurrency: 0
http-read-timeout: 15s
http-write-timeout: 15s
ui-disabled: true
pprof-disabled: true
metrics-disabled: true
log-level: error
bolt-path: {{.DataPath}}/influxdbv2/influxd.bolt
engine-path: {{.DataPath}}/influxdbv2/engine
reporting-disabled: true
`)
	TSDBSvcTmpl, _ = template.New("tsdbsvc").Parse(`# If you modify this, please also make sure to edit init.sh

[Unit]
Description=Edge IoT time series database
After=network-online.target

[Service]
User={{.User}}
Group={{.Group}}
Environment="INFLUXD_CONFIG_PATH={{.ConfigDir}}"
LimitNOFILE=65536
ExecStart={{.ExecFile}} $INFLUXD_OPTS
ExecReload=/bin/kill -HUP $MAINPID
KillMode=control-group
Restart=always
RestartSec=2s

[Install]
WantedBy=multi-user.target
`)
)

type TSDBConfig struct {
	DataPath  string
	ConfigDir string
	ExecFile  string
	User      string
	Group     string
}

func GetTSDBSvcUnitName() string {
	return tsdb
}

func GetTSDBSrcPath(superPath string) string {
	return filepath.Join(superPath, tsdbSrc)
}

func GetTSDBInstallPath() string {
	return util.GetAbsoluteRuntimePath(tsdbDst)
}

func GetTSDBDataPath() string {
	return util.GetAbsoluteDataPath(tsdbDst)
}

func GetTSDBSystemdPath() string {
	return util.GetAbsoluteRuntimePath(tsdbSystemd)
}

func GetTSDBSystemdSvcPath() string {
	return util.GetAbsoluteRuntimePath(filepath.Join(tsdbSystemd, tsdbSystemdSvc))
}

func GetTSDBExecutableFilesPath() string {
	return util.GetAbsoluteRuntimePath(tsdbExecutableFiles)
}

func GetTSDBExecutableFilePath() string {
	return util.GetAbsoluteRuntimePath(tsdbExecutableFile)
}

func GetTSDBConfigPath() string {
	return util.GetAbsoluteRuntimePath(tsdbConfig)
}

func GetTSDBConfigDir() string {
	return util.GetAbsoluteRuntimePath(tsdbConfigPath)
}
