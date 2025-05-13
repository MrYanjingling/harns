package install

import (
	"context"
	"encoding/json"
	"fmt"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"lightiot/cmd/iotadm/app/resource"
	"lightiot/cmd/iotadm/app/util"
	"lightiot/pkg/client"
	"net/http"
	"os"
	"time"
)

type TimeSeriesDBInstTool struct {
	util.Common
	SrcPath        string
	InfluxdbClient client.Client
}

// InstallTools implements ToolsInstaller interface
func (t *TimeSeriesDBInstTool) InstallTools() error {
	_, _ = fmt.Fprintf(t.Out, "Started install TSDB\n")
	if err := t.install(); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(t.Out, "Successful to install TSDB\n")
	return nil

}

// Check implements ToolsInstaller interface
func (t *TimeSeriesDBInstTool) Check() (resource.ToolDesc, error) {
	desc := resource.ToolDesc{
		Name: util.TOOLINFLUXDB,
	}

	desc.Version = t.getVersion()
	if desc.Version != "" {
		desc.IsInstalled = true
	} else {
		desc.IsInstalled = false
		return desc, nil
	}

	desc.IsRunning, _ = util.IsProcessRunning(util.TOOLINFLUXDBSERVEREXEC)
	return desc, nil
}

// TearDown implements ToolsInstaller interface
func (t *TimeSeriesDBInstTool) TearDown() error {
	if util.IsFileUser(resource.GetTSDBInstallPath(), resource.EdgeIoTUser) {
		tsdbSvcUnit := newTSDBSvcUnit()
		if err := util.StopSvc(tsdbSvcUnit); err != nil {
			_, _ = fmt.Fprintf(t.ErrOut, "Failed to stop TSDB\n")
			return err
		}

		if err := util.CancelSystemd(tsdbSvcUnit, t.IOStreams); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(t.Out, "Canceled TSDB service\n")

		if err := os.RemoveAll(resource.GetTSDBInstallPath()); err != nil {
			_, _ = fmt.Fprintf(t.ErrOut, "Failed to remove TSDB intall resources\n")
			return err
		}

		if util.IsDeleteEdgeIotData && util.IsFileUser(resource.GetTSDBDataPath(), resource.EdgeIoTUser) {
			if err := os.RemoveAll(resource.GetTSDBDataPath()); err != nil {
				_, _ = fmt.Fprintf(t.ErrOut, "Failed to remove TSDB data\n")
				return err
			}
		}
		_, _ = fmt.Fprintf(t.Out, "Successful to uninstall TSDB\n")
	} else {
		_, _ = fmt.Fprintf(t.Out, "Warning! Invalid TSDB install path or permision to remove\n")
	}
	return nil
}

func (t *TimeSeriesDBInstTool) install() error {
	t.SetOSInterface(util.GetOSInterface())
	if version := t.getVersion(); version != "" {
		_, _ = fmt.Fprintf(t.Out, "Warning! TSDB %s has been installed\n", version)
		return nil
	}

	if stat, _ := os.Stat(resource.GetTSDBDataPath()); stat == nil {
		if err := os.MkdirAll(resource.GetTSDBDataPath(), 0754); err != nil {
			_, _ = fmt.Fprintf(t.ErrOut, "Failed to create TSDB data path\n")
			return err
		}
	}

	if err := util.CopyFileWithRecursive(resource.GetTSDBSrcPath(t.SrcPath), resource.GetTSDBInstallPath(), t.IOStreams); err != nil {
		return fmt.Errorf("failed to copy TSDB resource files")
	}
	_, _ = fmt.Fprintf(t.Out, "Copied TSDB resource files\n")

	tsdbSvcUnit := newTSDBSvcUnit()
	if err := util.GenerateFiles(tsdbSvcUnit.File, t.IOStreams); err != nil {
		return err
	}

	if err := t.auth(); err != nil {
		return err
	}

	if err := util.RegisterSystemd(tsdbSvcUnit, t.IOStreams); err != nil {
		return err
	}
	// run
	if err := util.StartSvc(tsdbSvcUnit); err != nil {
		_, _ = fmt.Fprintf(t.ErrOut, "Failed to start TSDB\n")
		return err
	}
	if err := t.waitReady(); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(t.Out, "Started TSDB\n")

	if err := t.setup(); err != nil {
		_, _ = fmt.Fprintf(t.ErrOut, "Failed to setup TSDB\n")
		return err
	}
	_, _ = fmt.Fprintf(t.Out, "Setuped TSDB\n")
	return nil
}

func (t *TimeSeriesDBInstTool) getVersion() string {
	cmd := util.NewCommand("influxd version | awk '{print $2}'")
	err := cmd.Exec()
	if err != nil {
		_, _ = fmt.Fprintf(t.ErrOut, "Failed to get TSDB version\n")
		return ""
	}
	return cmd.GetStdOut()
}

func (t *TimeSeriesDBInstTool) auth() error {
	if err := util.ChownFile(resource.EdgeIoTUser, resource.GetTSDBInstallPath(), t.IOStreams); err != nil {
		return err
	}

	if err := util.ChownFile(resource.EdgeIoTUser, resource.GetTSDBDataPath(), t.IOStreams); err != nil {
		return err
	}

	if err := util.ChmodFile(resource.GetTSDBInstallPath(), 0754, t.IOStreams); err != nil {
		return err
	}

	if err := util.AddExecutablePermission(resource.GetTSDBExecutableFilesPath(), t.IOStreams); err != nil {
		return err
	}
	return nil
}

func (t *TimeSeriesDBInstTool) waitReady() error {
	_, _ = fmt.Fprintf(t.Out, "Waited TSDB ready\n")
	influxdbClient := influxdb2.NewClient("http://127.0.0.1:8086", "")
	ready := false
	timer := time.NewTimer(time.Second * 60)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			return fmt.Errorf("TSDB not ready")
		default:
		}
		func() {
			resp, err := influxdbClient.Ready(context.Background())
			if err != nil {
				return
			} else if resp {
				ready = true
				_, _ = fmt.Fprintf(t.Out, "TSDB ready\n")
			}
		}()
		if ready {
			break
		}
	}
	return nil
}

func (t *TimeSeriesDBInstTool) setup() error {
	getResp, err := t.InfluxdbClient.Get("/api/v2/setup", http.Header{}, nil)
	if err != nil {
		return fmt.Errorf("failed to access TSDB")
	}
	defer client.Drain(getResp)
	var setup = &struct {
		Allowed bool `json:"allowed"`
	}{}
	if getResp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to get setup status")
	} else if err = json.NewDecoder(getResp.Body).Decode(setup); err != nil {
		return fmt.Errorf("failed to parse setup status")
	} else if !setup.Allowed {
		return nil
	}

	defaultTSDBSetupConfig := &struct {
		Org                    string `json:"org"`
		Username               string `json:"username"`
		Password               string `json:"password"`
		Token                  string `json:"token"`
		Bucket                 string `json:"bucket"`
		RetentionPeriodSeconds int64  `json:"retentionPeriodSeconds"`
	}{
		resource.TSDBOrg,
		resource.TSDBUsername,
		resource.TSDBPassword,
		resource.TSDBToken,
		resource.DataRawBucket,
		int64(resource.RetentionPeriodSeconds),
	}
	header := http.Header{"Content-Type": []string{"application/json"}}
	resp, err := t.InfluxdbClient.Post("/api/v2/setup", header, nil, defaultTSDBSetupConfig)
	if err != nil {
		return fmt.Errorf("failed to access TSDB")
	}
	defer client.Drain(resp)
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to setup TSDB")
	}
	return nil
}

func newTSDBSvcUnit() *util.SvcUnit {
	data := &resource.TSDBConfig{
		DataPath:  resource.GetTSDBDataPath(),
		ConfigDir: resource.GetTSDBConfigDir(),
		ExecFile:  resource.GetTSDBExecutableFilePath(),
		User:      resource.EdgeIoTUser,
		Group:     resource.EdgeIoTGroup,
	}
	files := []util.FileDesc{
		{
			Path:     resource.GetTSDBConfigPath(),
			Template: resource.TSDBConfTmpl,
			Data:     data,
		},
		{
			Path:     resource.GetTSDBSystemdSvcPath(),
			Template: resource.TSDBSvcTmpl,
			Data:     data,
		},
	}
	return util.NewSvcUnit(resource.GetTSDBSvcUnitName(), resource.GetTSDBSystemdPath(), resource.DefaultReplicas, files)
}
