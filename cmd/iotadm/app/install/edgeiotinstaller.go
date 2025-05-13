package install

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/domain"
	"io"
	"lightiot/cmd/iotadm/app/resource"
	"lightiot/cmd/iotadm/app/util"
	"lightiot/pkg/client"
	ins "lightiot/pkg/installation"
	v1 "lightiot/pkg/installation/v1"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/util/sets"
)

type EdgeIotInstTool struct {
	util.Common
	SrcPath        string
	Services       []string
	Client         client.Client
	freePort       *resource.FreePort
	bidirection    bool
	enableRollup   bool
	enableRule     bool
	influxdbClient influxdb2.Client

	installMgrReady bool
	apiSvcUrls      map[string]string
}

// InstallTools implements ToolsInstaller interface
func (t *EdgeIotInstTool) InstallTools() error {
	_, _ = fmt.Fprintf(t.Out, "Started install EdgeIoT\n")
	if err := t.createBuckets(); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(t.Out, "Successful to initialize EdgeIoT databases\n")

	if err := t.install(); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(t.Out, "Successful to install EdgeIoT\n")
	return nil
}

// Check implements ToolsInstaller interface
func (t *EdgeIotInstTool) Check() (resource.ToolDesc, error) {
	desc := resource.ToolDesc{
		Name: util.TOOLEDGIOT,
	}
	return desc, nil
}

// TearDown implements ToolsInstaller interface
func (t *EdgeIotInstTool) TearDown() error {
	if util.IsFileUser(resource.GetEdgeIoTRuntimePath(), resource.EdgeIoTUser) {
		// stop services of iot
		svcUnits := t.getSystemdUnits()
		if err := t.stopEdgeIot(svcUnits); err != nil {
			_, _ = fmt.Fprintf(t.ErrOut, "Failed to stop EdgeIoT\n")
		}

		for _, svcUnit := range svcUnits {
			if err := util.CancelSystemd(svcUnit, t.IOStreams); err != nil {
				return err
			}
		}
		_, _ = fmt.Fprintf(t.Out, "Canceled EdgeIoT services\n")

		fmt.Print("If delete all data of EdgeIoT? ")
		if confirm := util.AskForConfirm(t.IOStreams); confirm {
			util.IsDeleteEdgeIotData = true
			if err := os.RemoveAll(resource.GetEdgeIoTDataPath()); err != nil {
				_, _ = fmt.Fprintf(t.ErrOut, "Failed to remove EdgeIoT data\n")
			}
		} else {
			util.IsDeleteEdgeIotData = false
		}

		if err := os.RemoveAll(resource.GetEdgeIoTConfigPath()); err != nil {
			_, _ = fmt.Fprintf(t.ErrOut, "Failed to remove EdgeIoT configs\n")
		}
		if err := os.RemoveAll(resource.GetEdgeIoTApiFilesPath()); err != nil {
			_, _ = fmt.Fprintf(t.ErrOut, "Failed to remove EdgeIoT apis\n")
		}

		apis := sets.String{}
		uis := sets.String{}
		for _, v := range resource.EdgeIoTServices {
			for _, a := range v.APIs {
				apis.Insert(a.Name)
			}
			for _, u := range v.UIs {
				uis.Insert(u)
			}
		}
		appFIs, _ := util.GetAllFiles(resource.GetEdgeIoTExecutableFilesPath(), t.IOStreams)
		has3rdSvc := false
		for _, fi := range appFIs {
			name := filepath.Base(fi)
			if apis.Has(name) {
				if err := os.Remove(fi); err != nil {
					_, _ = fmt.Fprintf(t.ErrOut, "Failed to remove %s\n", fi)
				}
			} else {
				has3rdSvc = true
			}
		}
		if !has3rdSvc {
			if err := os.RemoveAll(resource.GetEdgeIoTExecutableFilesPath()); err != nil {
				_, _ = fmt.Fprintf(t.ErrOut, "Failed to remove EdgeIoT executable file path\n")
			}
		}

		uiFIs, _ := util.GetAllFiles(resource.GetEdgeIoTAppFilesPath(), t.IOStreams)
		has3rdSvc = false
		for _, fi := range uiFIs {
			name := filepath.Base(fi)
			if uis.Has(name) {
				if err := os.Remove(fi); err != nil {
					_, _ = fmt.Fprintf(t.ErrOut, "Failed to remove %s\n", fi)
				}
			} else {
				has3rdSvc = true
			}
		}
		if !has3rdSvc {
			if err := os.RemoveAll(resource.GetEdgeIoTAppFilesPath()); err != nil {
				_, _ = fmt.Fprintf(t.ErrOut, "Failed to remove EdgeIoT api file path\n")
			}
		}
		_, _ = fmt.Fprintf(t.Out, "Successful to uninstall EdgeIoT\n")
	} else {
		_, _ = fmt.Fprintf(t.Out, "Warning! Invalid EdgeIoT install path or permision to remove\n")
	}
	return nil
}

func (t *EdgeIotInstTool) install() error {
	t.SetOSInterface(util.GetOSInterface())
	if err := t.InstallGateway(); err != nil {
		_, _ = fmt.Fprintf(t.ErrOut, "Failed to open gateway port %d\n", util.GatewayTCPPort)
		// TODO: need a graceful method
		// return err
	}

	_ = t.stopEdgeIot(t.getSystemdUnits())

	for _, dir := range []string{
		resource.GetEdgeIoTConfigPath(),
		resource.GetEdgeIoTConfigdPath(),
		resource.GetEdgeIoTDataPath(),
		resource.GetEdgeIoTRuntimePath(),
		resource.GetEdgeIoTExecutableFilesPath(),
	} {
		if err := os.MkdirAll(dir, 0754); err != nil {
			_, _ = fmt.Fprintf(t.ErrOut, "Failed to create path %s\n", dir)
			return err
		}

		if err := util.ChownFile(resource.EdgeIoTUser, dir, t.IOStreams); err != nil {
			return err
		}
	}

	var (
		svcs []*resource.Service
		eis  map[int][]*resource.EdgeInstallation
		err  error
	)
	if svcs, err = t.copyResources(); err != nil {
		_, _ = fmt.Fprintf(t.ErrOut, "Failed to copy EdgeIoT resource file because %s\n", err)
		return fmt.Errorf("failed to copy EdgeIoT resource files")
	}
	_, _ = fmt.Fprintf(t.Out, "Copied EdgeIoT resource files\n")

	eis = t.convert2Installation(svcs)

	priorities := make([]int, 0)
	for k := range eis {
		priorities = append(priorities, k)
	}
	sort.Ints(priorities)

	for _, priority := range priorities {
		if priority == 0 {
			if err = t.waitGatewayReady(); err != nil {
				return err
			}
		}
		for _, ei := range eis[priority] {
			if ei.Name == resource.UIOsBar {
				// os-bar is special, enable it later
				continue
			}
			if err := t.provisionInstallation(priority, ei); err != nil {
				return err
			}
		}
	}

	if err := t.auth(); err != nil {
		return err
	}

	t.registerOsBar()

	return nil
}

func (t *EdgeIotInstTool) registerOsBar() {
	if resp, err := t.Client.Post("/api/installation/v1/installations",
		http.Header{"Content-Type": []string{"application/json"}},
		nil,
		v1.Installation{
			Name:        resource.UIOsBar,
			DisplayName: resource.UIOsBar,
			ProviderId:  "main",
			Version:     "v1",
			Type:        v1.InstallationTypeAPI,
			Start:       time.Now(),
			End:         time.Now().Add(365 * 24 * time.Hour),
			Components: []*v1.Component{
				{
					Name: resource.UIOsBar,
					Uri:  resource.UIOsBar,
					Endpoints: []v1.Endpoint{
						{
							Path:   filepath.Join(resource.GetEdgeIoTAppFilesPath(), resource.UIOsBar, "header.js"),
							Action: v1.EndpointActionGet,
						},
					},
				},
			},
		}); err != nil {
		_, _ = fmt.Fprintf(t.ErrOut, "Failed to register os-bar\n")
	} else {
		defer client.Drain(resp)
	}
	_, _ = fmt.Fprintf(t.Out, "Registered os-bar\n")
}

func (t *EdgeIotInstTool) convert2Installation(svcs []*resource.Service) (eis map[int][]*resource.EdgeInstallation) {
	eis = map[int][]*resource.EdgeInstallation{
		resource.EdgeIoTUIPriority: make([]*resource.EdgeInstallation, 0),
	}
	for _, svc := range svcs {
		ei, ok := eis[svc.Priority]
		if !ok {
			ei = make([]*resource.EdgeInstallation, 0)
		}
		for _, api := range svc.APIs {
			ei = append(ei, &resource.EdgeInstallation{
				Name:     api.Name,
				Type:     v1.InstallationTypeAPI,
				Replicas: api.Replicas,
				Resource: filepath.Join(resource.GetEdgeIoTExecutableFilesPath(), api.Name),
				StartCommand: &resource.Command{
					Command: filepath.Join(resource.GetEdgeIoTExecutableFilesPath(), api.Name),
				},
			})
			eis[svc.Priority] = ei
		}

		for _, ui := range svc.UIs {
			var defaultOsBarType v1.EnableOsBar = true
			if ui == resource.UIHelp || ui == resource.UIEvent {
				defaultOsBarType = false
			}
			eis[resource.EdgeIoTUIPriority] = append(eis[resource.EdgeIoTUIPriority], &resource.EdgeInstallation{
				Name:        ui,
				DisplayName: ui,
				Type:        v1.InstallationTypeUI,
				Replicas:    1,
				Resource:    filepath.Join(resource.GetEdgeIoTAppFilesPath(), ui),
				OsBar:       &defaultOsBarType,
			})
		}
	}
	return
}

func (t *EdgeIotInstTool) provisionInstallation(priority int, ei *resource.EdgeInstallation) error {
	_, _ = fmt.Fprintf(t.Out, "Started provision %s\n", ei.Name)
	header := http.Header{"Content-Type": []string{"application/json"}}
	var port int
	if ei.Type == v1.InstallationTypeAPI {
		port = t.freePort.Next()
		startCmd := bytes.Buffer{}
		startCmd.WriteString(ei.StartCommand.Command)
		for k, v := range ei.StartCommand.Args {
			startCmd.WriteString(fmt.Sprintf(" %s=%v", k, v))
		}
		if f, ok := apiExtraArgs[ei.Name]; ok {
			for _, v := range f(t) {
				startCmd.WriteString(fmt.Sprintf(" %s", v))
			}
		}
		if priority >= 0 && priority < resource.EdgeIoTUIPriority {
			if ei.Name != resource.APIJobConsumer {
				startCmd.WriteString(fmt.Sprintf(" --port=%d", port))
			}
			// debug for svc
			// startCmd += fmt.Sprintf(" -v=%d", 5)
			startCmd.WriteString(fmt.Sprintf(" -v=%d", 2))
			startCmd.WriteString(fmt.Sprintf(" --logging-format=%s", "json"))
		}

		if resource.DependOnTSDBAPIs.Has(ei.Name) {
			startCmd.WriteString(fmt.Sprintf(" --influxdb-token=%s", resource.TSDBToken))
		}

		if t.enableRule && ei.Name == resource.APIDataCollector {
			startCmd.WriteString(" --enable-rule")
		}

		if t.bidirection && ei.Name == resource.APIDataBroker {
			startCmd.WriteString(" --bi-direction")
		}

		if t.enableRollup && ei.Name == resource.APIDataCollector {
			startCmd.WriteString(" --enable-rollup")
		}

		if !t.enableRollup && ei.Name == resource.APIJobConsumer {
			ei.Replicas = resource.DefaultReplicas
			startCmd.WriteString(" --job-groups=del")
			startCmd.WriteString(" --retrieve-job-interval=120")
		}

		var svcFileData interface{}
		if ei.Name == resource.APIGateway {
			svcFileData = resource.EdgeIoTConfig{
				StartCmd: apiExtraArgs[ei.Name](t)[0],
				User:     "root",
				Group:    "root",
			}
		} else {
			svcFileData = resource.EdgeIoTConfig{
				StartCmd: startCmd.String(),
				User:     resource.EdgeIoTUser,
				Group:    resource.EdgeIoTGroup,
			}
		}
		svcUnit := ei.ConvertEdgeIotSvcUnit(svcFileData)

		if err := util.GenerateFiles(svcUnit.File, t.IOStreams); err != nil {
			return err
		}

		if err := util.ChownFile(resource.EdgeIoTUser, filepath.Join(resource.GetEdgeIoTConfigPath(), svcUnit.SvcFileName), t.IOStreams); err != nil {
			return err
		}

		if err := util.RegisterSystemd(svcUnit, t.IOStreams); err != nil {
			return err
		}
		if err := util.StartSvc(svcUnit); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(t.Out, "Started EdgeIoT service %s\n", ei.Name)
	}

	if priority >= 0 {
		i := ei.Convert()

		if ei.Type == v1.InstallationTypeAPI {
			uri := fmt.Sprintf("http://127.0.0.1:%d", port)
			i.Components[0].Uri = uri
			t.apiSvcUrls[ei.Name] = uri
		} else if ei.Type == v1.InstallationTypeUI {
			if ei.Name == resource.UILaunchpad {
				i.Components[0].Uri = "/"
			} else {
				i.Components[0].Uri = fmt.Sprintf("/%s", ei.Name)
			}
		}
		if !t.installMgrReady {
			if err := t.waitInstallationManagerReady(); err != nil {
				return err
			}
		}

		filter := struct {
			Name string              `json:"name"`
			Type v1.InstallationType `json:"type"`
		}{
			Name: i.Name,
			Type: i.Type,
		}
		fb, _ := json.Marshal(filter)

		getResp, err := t.Client.Get("/api/installation/v1/installations", nil, &url.Values{"filter": []string{string(fb)}})

		if err != nil {
			_, _ = fmt.Fprintf(t.ErrOut, "Failed to get installation %s, %s\n", i.Name, v1.InstallationTypeToString[i.Type])
			return err
		}
		defer client.Drain(getResp)
		if getResp.StatusCode == http.StatusOK {
			type insRes struct {
				Installations []ins.Installation
			}
			var res insRes
			if err = json.NewDecoder(getResp.Body).Decode(&res); err != nil {
				_, _ = fmt.Fprintf(t.ErrOut, "Failed to parse installations\n")
			}

			if len(res.Installations) != 0 {
				is := res.Installations[0]
				deleteResp, err := t.Client.Delete("/api/installation/v1/installations/" + is.ID)
				if err != nil {
					_, _ = fmt.Fprintf(t.ErrOut, "Failed to delete installation %s\n", is.Name)
				}
				defer client.Drain(deleteResp)
				if deleteResp.StatusCode == http.StatusOK {
					for _, c := range is.Components {
						if c.Name != i.Components[0].Name {
							rc := &v1.Component{
								Name:      c.Name,
								Uri:       c.Uri,
								Endpoints: make([]v1.Endpoint, len(c.Endpoints)),
							}
							for i, ep := range c.Endpoints {
								rc.Endpoints[i] = v1.Endpoint{
									Path:   ep.Path,
									Action: ep.Action,
									OsBar:  ep.OsBar,
								}
							}
							i.Components = append(i.Components, rc)
						}
					}
				}
				_, _ = fmt.Fprintf(t.Out, "Updated installation %s\n", is.Name)
			}

			resp, err := t.Client.Post("/api/installation/v1/installations", header, nil, i)
			if err != nil {
				_, _ = fmt.Fprintf(t.ErrOut, "Failed to create installation %s\n", i.Name)
				return err
			}
			defer client.Drain(resp)
			if ei.Type == v1.InstallationTypeUI {
				var installationRes ins.Installation
				if resp.StatusCode == http.StatusCreated {
					if err = json.NewDecoder(resp.Body).Decode(&installationRes); err != nil {
						_, _ = fmt.Fprintf(t.ErrOut, "Failed to parse installation\n")
						return err
					}
				} else {
					errMsg, _ := io.ReadAll(resp.Body)
					_, _ = fmt.Fprintf(t.ErrOut, "Failed to create installation %s\n", string(errMsg))
				}

				if len(installationRes.ID) > 0 {
					images, _ := util.GetAllFiles(fmt.Sprintf("%s/installations/%s/images", t.SrcPath, i.Name), t.IOStreams)
					for _, img := range images {
						if err = t.uploadImg(img, installationRes); err != nil {
							return err
						}
					}
				}

			}
		}
	}
	_, _ = fmt.Fprintf(t.Out, "Successful to provision EdgeIoT service %s\n", ei.Name)
	return nil
}

func (t *EdgeIotInstTool) copyResources() ([]*resource.Service, error) {
	if err := util.CopyFileWithRecursive(filepath.Join(t.SrcPath, "api"), resource.GetEdgeIoTApiFilesPath(), t.IOStreams); err != nil {
		return nil, err
	}
	var svcs []*resource.Service
	binSrcPath := filepath.Join(t.SrcPath, "bin")
	appSrcPath := filepath.Join(t.SrcPath, "app")
	binDstPath := resource.GetEdgeIoTExecutableFilesPath()
	appDstPath := resource.GetEdgeIoTAppFilesPath()
	for _, svcname := range t.Services {
		for _, svc := range resource.EdgeIoTServices {
			if svc.Name == svcname {
				svcs = append(svcs, &svc)
				for _, api := range svc.APIs {
					if err := util.CopyFile(filepath.Join(binSrcPath, api.Name), filepath.Join(binDstPath, api.Name)); err != nil {
						return nil, err
					}
					if err := util.ChownFile(resource.EdgeIoTUser, filepath.Join(binDstPath, api.Name), t.IOStreams); err != nil {
						return nil, err
					}
					if err := util.AddExecutablePermission(filepath.Join(binDstPath, api.Name), t.IOStreams); err != nil {
						return nil, err
					}
				}
				for _, app := range svc.UIs {
					if err := util.CopyFileWithRecursive(filepath.Join(appSrcPath, app), filepath.Join(appDstPath, app), t.IOStreams); err != nil {
						return nil, err
					}
				}
				break
			}
		}
	}
	return svcs, nil
}

func (t *EdgeIotInstTool) auth() error {
	for _, path := range []string{
		resource.GetEdgeIoTConfigPath(),
		resource.GetEdgeIoTDataPath(),
		resource.GetEdgeIoTRuntimePath(),
	} {
		if err := util.ChownFile(resource.EdgeIoTUser, path, t.IOStreams); err != nil {
			return err
		}
	}
	if err := util.AddExecutablePermission(resource.GetEdgeIoTExecutableFilesPath(), t.IOStreams); err != nil {
		return err
	}
	return nil
}

func (t *EdgeIotInstTool) stopEdgeIot(svcUnits []*util.SvcUnit) error {
	var units string
	for _, svcUnit := range svcUnits {
		units += svcUnit.SvcFullName() + " "
	}
	command := util.NewCommand(fmt.Sprintf("systemctl stop %s", units))
	return command.Exec()
}

func (t *EdgeIotInstTool) getSystemdUnits() []*util.SvcUnit {
	svcUnits := make([]*util.SvcUnit, 0)
	if fis, err := util.GetAllFiles(resource.GetEdgeIoTConfigPath(), t.IOStreams); err != nil {
		return svcUnits
	} else {
		for _, fi := range fis {
			fiName := filepath.Base(fi)
			svcName := fiName[0:strings.LastIndex(fiName, filepath.Ext(fiName))]
			if strings.HasSuffix(svcName, "@") {
				command := util.NewCommand(fmt.Sprintf(`systemctl --type=service | grep %s`, svcName[0:len(svcName)-1]))
				if err := command.Exec(); err != nil {
					_, _ = fmt.Fprintf(t.ErrOut, "Failed to list %s service instances\n", svcName[0:len(svcName)-1])
					continue
				} else {
					processCount := strings.Split(command.GetStdOut(), "\n")
					svcUnits = append(svcUnits, util.NewSvcUnit(svcName[0:len(svcName)-1], resource.GetEdgeIoTConfigPath(), len(processCount), nil))
				}
			} else {
				svcUnits = append(svcUnits, util.NewSvcUnit(svcName, resource.GetEdgeIoTConfigPath(), resource.DefaultReplicas, nil))
			}
		}
	}

	return svcUnits
}

func (t *EdgeIotInstTool) waitInstallationManagerReady() error {
	_, _ = fmt.Fprintf(t.Out, "Waited installation manager ready\n")
	timer := time.NewTimer(time.Second * 60)
	defer timer.Stop()
	header := http.Header{"Content-Type": []string{"application/json"}}
	for {
		select {
		case <-timer.C:
			return fmt.Errorf("intallation manager not ready")
		default:
		}
		func() {
			resp, err := t.Client.Get("/api/installation/v1/installations", header, nil)
			if err != nil {
				return
			}
			defer client.Drain(resp)
			if resp.StatusCode == http.StatusOK {
				t.installMgrReady = true
				_, _ = fmt.Fprintf(t.Out, "intallation manager ready\n")
			}
		}()
		if t.installMgrReady {
			break
		}
	}
	return nil
}

func (t *EdgeIotInstTool) waitGatewayReady() error {
	_, _ = fmt.Fprintf(t.Out, "Waited harns-gateway ready\n")
	ready := false
	timer := time.NewTimer(time.Second * 60)
	u, _ := url.Parse(gatewayReadyUrl)
	tp := http.DefaultTransport.(*http.Transport)
	c := client.NewClient(u, 60*time.Second, tp)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			return fmt.Errorf("harns-gateway not ready")
		default:
		}
		func() {
			resp, err := c.Get("/ready", nil, nil)
			if err != nil {
				return
			}
			defer client.Drain(resp)
			if resp.StatusCode == http.StatusOK {
				if ret, err := io.ReadAll(resp.Body); err != nil {
					_, _ = fmt.Fprintf(t.ErrOut, "Failed to read response body\n")
					return
				} else {
					if string(ret[:len(ret)-1]) == "LIVE" {
						ready = true
						_, _ = fmt.Fprintf(t.Out, "harns-gateway ready\n")
						return
					}
				}
			}
		}()
		if ready {
			break
		}
	}
	return nil
}

func (t *EdgeIotInstTool) uploadImg(img string, i ins.Installation) error {
	return resource.UploadImage(img, i, t.IOStreams)
}

func (t *EdgeIotInstTool) createBuckets() error {
	var buckets []resource.Bucket
	for _, svc := range t.Services {
		if bucket, ok := resource.EdgeIoTBuckets[svc]; ok {
			buckets = append(buckets, bucket...)
		}
	}
	existBuckets, err := t.influxdbClient.BucketsAPI().GetBuckets(context.Background())
	ebs := sets.String{}
	if err != nil {
		_, _ = fmt.Fprintf(t.ErrOut, "Failed to get existed databases\n")
	} else if existBuckets != nil {
		for _, eb := range *existBuckets {
			ebs.Insert(eb.Name)
		}
	}

	if len(buckets) > 0 {
		org, err := t.influxdbClient.OrganizationsAPI().FindOrganizationByName(context.Background(), resource.TSDBOrg)
		if err != nil {
			_, _ = fmt.Fprintf(t.ErrOut, "Failed to get org %s\n", resource.TSDBOrg)
		}
		for _, bucket := range buckets {
			if ebs.Has(bucket.Name) {
				continue
			}
			if err := t.createBucket(bucket, org); err != nil {
				return err
			}
		}
	}
	return nil
}

const (
	oneHour  = 60 * 60
	oneDay   = 24 * oneHour
	oneWeek  = 7 * oneDay
	oneMonth = 30 * oneDay
)

func (t *EdgeIotInstTool) createBucket(bucket resource.Bucket, org *domain.Organization) error {
	// https://docs.influxdata.com/influxdb/v1.8/concepts/schema_and_data_layout/#shard-group-duration-recommendations
	duration := bucket.TTL * oneDay
	var sgd int64 // shard group duration
	if duration == 0 {
		sgd = int64(52 * oneWeek)
	} else if duration <= oneDay {
		sgd = int64(6 * oneHour)
	} else if duration > oneDay && duration <= oneWeek {
		sgd = int64(oneDay)
	} else if duration > oneWeek && duration <= 3*oneMonth {
		sgd = int64(oneWeek)
	} else {
		sgd = int64(oneMonth)
	}

	rr := domain.RetentionRule{
		EverySeconds:              duration,
		ShardGroupDurationSeconds: &sgd,
		Type:                      "expire",
	}

	if _, err := t.influxdbClient.BucketsAPI().CreateBucketWithName(context.Background(), org, bucket.Name, rr); err != nil {
		_, _ = fmt.Fprintf(t.ErrOut, "Failed to init database %s\n", bucket.Name)
		return err
	}
	_, _ = fmt.Fprintf(t.Out, "Successful to create database %s\n", bucket.Name)
	return nil
}

func (t *EdgeIotInstTool) getAPISvcUrl(name string) string {
	if u, ok := t.apiSvcUrls[name]; ok {
		return u
	}
	// use gateway
	return "http://127.0.0.1"
}

type extraArgs func(*EdgeIotInstTool) []string

var gatewayReadyUrl string // Ugly code

var apiExtraArgs = map[string]extraArgs{
	resource.APIGateway: func(eit *EdgeIotInstTool) []string {
		return []string{
			"/bin/bash -c '/usr/local/edgeiot/bin/harns-gateway --config-path /etc/edgeiot/conf.d/gateway.yaml  | tee'",
		}
	},
	resource.APIGatewayProxy: func(eit *EdgeIotInstTool) []string {
		p1 := eit.freePort.Next()
		p2 := eit.freePort.Next()

		gatewayReadyUrl = fmt.Sprintf("http://127.0.0.1:%d", p1)

		return []string{
			fmt.Sprintf("--envoy-admin_port=%d", p1),
			fmt.Sprintf("--web-server_port=%d", p2),
		}
	},
	resource.APIInstallation: func(eit *EdgeIotInstTool) []string {
		p := eit.freePort.Next()
		return []string{
			fmt.Sprintf("--registry-url=http://127.0.0.1:%d", p),
			fmt.Sprintf("--port=%d", p),
			fmt.Sprintf("-v=%d", 2),
		}
	},
	resource.APIDataCollector: func(eit *EdgeIotInstTool) []string {
		return []string{
			fmt.Sprintf("--iot-event-manager-url=%s", eit.getAPISvcUrl(resource.APIEvent)),
			fmt.Sprintf("--iot-notification-manager-url=%s", eit.getAPISvcUrl(resource.APINotify)),
		}
	},
	resource.APIDataBroker: func(eit *EdgeIotInstTool) []string {
		return []string{
			fmt.Sprintf("--iot-event-manager-url=%s", eit.getAPISvcUrl(resource.APIEvent)),
			fmt.Sprintf("--iot-data-collector-url=%s", eit.getAPISvcUrl(resource.APIDataCollector)),
		}
	},
	resource.APIRule: func(eit *EdgeIotInstTool) []string {
		return []string{
			fmt.Sprintf("--iot-model-manager-url=%s", eit.getAPISvcUrl(resource.APIModel)),
			fmt.Sprintf("--iot-notification-manager-url=%s", eit.getAPISvcUrl(resource.APINotify)),
		}
	},
	resource.APIControl: func(eit *EdgeIotInstTool) []string {
		return []string{
			fmt.Sprintf("--iot-data-broker-url=%s", eit.getAPISvcUrl(resource.APIDataBroker)),
		}
	},
}
