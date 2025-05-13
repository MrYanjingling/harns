package resource

import (
	"lightiot/cmd/iotadm/app/util"
	"path/filepath"
	"text/template"
)

const (
	mqtt               = "mosquitto"
	mqttSrc            = "/dependencies/mosquitto/"
	mqttDst            = "/mosquitto/"
	mqttSystemd        = "/mosquitto/systemd/"
	mqttSystemdSvc     = "mosquitto.service"
	mqttExecutableFile = "/mosquitto/sbin/mosquitto"
	mqttConfig         = "/mosquitto/conf/mosquitto.conf"
	mqttEdgeIoTConfig  = "/mosquitto/conf/conf.d/edgeiot-mosquitto.conf"
)

var (
	MqttConfTmpl, _ = template.New("mqttconf").Parse(`include_dir {{.InstallPath}}/conf/conf.d`)

	MqttEdgeIotConfTmp, _ = template.New("mqttiotconf").Parse(`persistence true
persistence_location {{.DataPath}}/data
bind_address 0.0.0.0
allow_anonymous true`)

	MqttSystemdServiceTmpl, _ = template.New("mqttsvc").Parse(`[Unit]
Description=Edge IoT MQTT Broker daemon
ConditionPathExists={{.ConfigFile}}
After=network.target
Requires=network.target

[Service]
Type=forking
User={{.User}}
Group={{.Group}}
RemainAfterExit=no
StartLimitInterval=0
ExecStart={{.ExecFile}} -c {{.ConfigFile}} -d
ExecReload=/bin/kill -HUP $MAINPID
Restart=always
RestartSec=2s

[Install]
WantedBy=multi-user.target
`)
)

type MQTTConfig struct {
	InstallPath string
	DataPath    string
	ConfigFile  string
	ExecFile    string
	User        string
	Group       string
}

func GetMQTTSvcUnitName() string {
	return mqtt
}

func GetMQTTSrcPath(superPath string) string {
	return filepath.Join(superPath, mqttSrc)
}

func GetMQTTInstallPath() string {
	return util.GetAbsoluteRuntimePath(mqttDst)
}

func GetMQTTDataPath() string {
	return util.GetAbsoluteDataPath(mqttDst)
}

func GetMQTTSystemdPath() string {
	return util.GetAbsoluteRuntimePath(mqttSystemd)
}

func GetMQTTSystemdSvcPath() string {
	return util.GetAbsoluteRuntimePath(filepath.Join(mqttSystemd, mqttSystemdSvc))
}

func GetMQTTExecutableFilePath() string {
	return util.GetAbsoluteRuntimePath(mqttExecutableFile)
}

func GetMQTTConfigPath() string {
	return util.GetAbsoluteRuntimePath(mqttConfig)
}

func GetMQTTEdgeIoTConfigPath() string {
	return util.GetAbsoluteRuntimePath(mqttEdgeIoTConfig)
}
