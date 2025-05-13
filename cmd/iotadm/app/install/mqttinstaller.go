package install

import (
	"fmt"
	"lightiot/cmd/iotadm/app/resource"
	"lightiot/cmd/iotadm/app/util"
	"os"
)

type MqttInstTool struct {
	util.Common
	SrcPath string
}

// InstallTools implements ToolsInstaller interface
func (t *MqttInstTool) InstallTools() error {
	_, _ = fmt.Fprintf(t.Out, "Started install MQTT broker\n")
	if err := t.install(); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(t.Out, "Successful to install MQTT broker\n")
	return nil
}

// Check implements ToolsInstaller interface
func (t *MqttInstTool) Check() (resource.ToolDesc, error) {
	desc := resource.ToolDesc{
		Name: util.TOOLMOSQUITTO,
	}

	desc.IsInstalled = t.isToolInstalled()
	if !desc.IsInstalled {
		return desc, nil
	}

	desc.Version = t.getVersion()
	desc.IsRunning, _ = util.IsProcessRunning(util.TOOLMOSQUITTO)
	return desc, nil
}

// TearDown implements ToolsInstaller interface
func (t *MqttInstTool) TearDown() error {
	mqttSvcUnit := newMQTTSvcUnit()
	if util.IsFileUser(resource.GetMQTTInstallPath(), resource.EdgeIoTUser) {
		if err := util.StopSvc(mqttSvcUnit); err != nil {
			_, _ = fmt.Fprintf(t.ErrOut, "Failed to stop MQTT broker\n")
			return err
		}

		if err := util.CancelSystemd(mqttSvcUnit, t.IOStreams); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(t.Out, "Canceled MQTT broker service\n")

		if err := os.RemoveAll(resource.GetMQTTInstallPath()); err != nil {
			_, _ = fmt.Fprintf(t.ErrOut, "Failed to remove MQTT broker intall resources\n")
			return err
		}

		if util.IsDeleteEdgeIotData && util.IsFileUser(resource.GetMQTTDataPath(), resource.EdgeIoTUser) {
			if err := os.RemoveAll(resource.GetMQTTDataPath()); err != nil {
				_, _ = fmt.Fprintf(t.ErrOut, "Failed to remove MQTT broker data\n")
				return err
			}
		}
		_, _ = fmt.Fprintf(t.Out, "Successful to uninstall MQTT broker\n")
	} else {
		_, _ = fmt.Fprintf(t.Out, "Warning! Invalid MQTT install path or permision to remove\n")
	}
	return nil
}

func (t *MqttInstTool) install() error {
	t.SetOSInterface(util.GetOSInterface())
	if version := t.getVersion(); len(version) != 0 {
		_, _ = fmt.Fprintf(t.Out, "Warning! MQTT %s has been installed\n", version)
		return nil
	}

	if err := util.CopyFileWithRecursive(resource.GetMQTTSrcPath(t.SrcPath), resource.GetMQTTInstallPath(), t.IOStreams); err != nil {
		return fmt.Errorf("failed to copy MQTT resource files")
	}
	_, _ = fmt.Fprintf(t.Out, "Copied MQTT resource files\n")

	if stat, _ := os.Stat(resource.GetMQTTDataPath()); stat == nil {
		if err := os.MkdirAll(resource.GetMQTTDataPath(), 0754); err != nil {
			_, _ = fmt.Fprintf(t.ErrOut, "Failed to create MQTT data path\n")
			return err
		}
	}

	mqttSvcUnit := newMQTTSvcUnit()

	if err := util.GenerateFiles(mqttSvcUnit.File, t.IOStreams); err != nil {
		return err
	}

	if err := t.auth(); err != nil {
		return err
	}

	if err := util.RegisterSystemd(mqttSvcUnit, t.IOStreams); err != nil {
		return err
	}
	// run mosquitto
	if err := util.StartSvc(mqttSvcUnit); err != nil {
		_, _ = fmt.Fprintf(t.ErrOut, "Failed to start MQTT broker\n")
		return err
	}
	_, _ = fmt.Fprintf(t.Out, "Started MQTT broker\n")
	return nil
}

func (t *MqttInstTool) auth() error {
	if err := util.ChownFile(resource.EdgeIoTUser, resource.GetMQTTInstallPath(), t.IOStreams); err != nil {
		return err
	}
	if err := util.ChownFile(resource.EdgeIoTUser, resource.GetMQTTDataPath(), t.IOStreams); err != nil {
		return err
	}

	if err := util.ChmodFile(resource.GetMQTTInstallPath(), 0754, t.IOStreams); err != nil {
		return err
	}
	if err := util.AddExecutablePermission(resource.GetMQTTExecutableFilePath(), t.IOStreams); err != nil {
		return err
	}
	return nil
}

func (t *MqttInstTool) isToolInstalled() bool {
	cmd := util.NewCommand("which mosquitto")
	if err := cmd.Exec(); err != nil {
		return false
	}
	return true
}

func (t *MqttInstTool) getVersion() string {
	cmd := util.NewCommand("mosquitto -h | grep \"^mosquitto version\" | awk '{print $3}'")
	if err := cmd.Exec(); err != nil {
		_, _ = fmt.Fprintf(t.ErrOut, "Failed to get MQTT version\n")
		return ""
	}
	return cmd.GetStdOut()
}

func newMQTTSvcUnit() *util.SvcUnit {
	data := &resource.MQTTConfig{
		InstallPath: resource.GetMQTTInstallPath(),
		DataPath:    resource.GetMQTTDataPath(),
		ConfigFile:  resource.GetMQTTConfigPath(),
		ExecFile:    resource.GetMQTTExecutableFilePath(),
		User:        resource.EdgeIoTUser,
		Group:       resource.EdgeIoTGroup,
	}
	files := []util.FileDesc{
		{
			Path:     resource.GetMQTTConfigPath(),
			Template: resource.MqttConfTmpl,
			Data:     data,
		},
		{
			Path:     resource.GetMQTTEdgeIoTConfigPath(),
			Template: resource.MqttEdgeIotConfTmp,
			Data:     data,
		},
		{
			Path:     resource.GetMQTTSystemdSvcPath(),
			Template: resource.MqttSystemdServiceTmpl,
			Data:     data,
		},
	}
	return util.NewSvcUnit(resource.GetMQTTSvcUnitName(), resource.GetMQTTSystemdPath(), resource.DefaultReplicas, files)
}
