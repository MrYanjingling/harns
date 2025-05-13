package util

import (
	"fmt"
)

var (
	IsDeleteEdgeIotData = true
	IotUsers            = []string{"edgeiot", "manager"}
	IotServices         = []string{"iot-model-manager", "iot-data-collector", "iot-data-broker", "iot-data-query", "iot-installation-manager"}
	Installations       = []string{"modelmanager", "fleetmanager"}

	OfferMap = map[string]string{
		"event":        "iot-event-manager",
		"rule":         "iot-rule-manager",
		"notification": "iot-notification-manager",
		"control":      "iot-control-manager",
	}
	InstallationMap = map[string]string{
		"rule":         "ruleengine",
		"notification": "notifymanager",
		"control":      "controlmanager",
	}

	IoTInstallations = []string{"public", "launchpad", "modelmanager", "fleetmanager", "ruleengine", "notifymanager", "controlmanager"}
)

func (co *Common) SetOSInterface(intf OSTypeInstaller) {
	co.OSTypeInstaller = intf
}

func GetPackageManager() string {
	return ""
}

func GetOSInterface() OSTypeInstaller {
	return nil
}

func IsProcessRunning(proc string) (bool, error) {
	return false, nil
}

func IsUserRoot() bool {
	// refer to https://github.com/golang/go/issues/28804
	return false
}


func GetAllFile(pathname string) ([]string, error) {
	return nil, nil
}

func RegisterSystemd(fis []string, ioStreams IOStreams) error {
	return nil
}

func CancelSystemd(svcUnit string, ioStreams IOStreams) error {
	return nil
}

func Decompress(tarFile, dst string) error {
	return nil
}

func existDir(dirname string) bool {
	return false
}

func CreateGroup(groupName string) error {
	return nil
}

func CreateUser(username, groupName string, ioStreams IOStreams) error {
	return nil
}

func IsFileUser(path, username string) bool {
	return false
}

func AskForConfirm(ioStreams IOStreams) bool {
	var s string
	fmt.Print("[Y/n]: ")
	if _, err := fmt.Scanln(&s); err != nil {
		return false
	}
	if s == "Y" {
		return true
	}
	return false
}

func GetMemTotal() (uint64, error) {
	return 0, nil
}

func GetAbsoluteDataPath(path string) string {
	return ""
}

func GetAbsoluteRuntimePath(path string) string {
	return ""
}

func GetAbsoluteConfigPath(path string) string {
	return ""
}

func GetSymbolicLinkPath() string {
	return ""
}