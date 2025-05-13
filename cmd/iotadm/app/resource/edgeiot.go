package resource

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	consumer "lightiot/cmd/iot-consumer/app"
	control "lightiot/cmd/iot-control-manager/app"
	databroker "lightiot/cmd/iot-data-broker/app"
	datacollector "lightiot/cmd/iot-data-collector/app"
	dataquery "lightiot/cmd/iot-data-query/app"
	datarollup "lightiot/cmd/iot-data-rollup/app"
	event "lightiot/cmd/iot-event-manager/app"
	installation "lightiot/cmd/iot-installation-manager/app"
	model "lightiot/cmd/iot-model-manager/app"
	notification "lightiot/cmd/iot-notification-manager/app"
	rule "lightiot/cmd/iot-rule-manager/app"
	"lightiot/cmd/iotadm/app/util"
	"lightiot/pkg/client"
	ins "lightiot/pkg/installation"
	v1 "lightiot/pkg/installation/v1"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"text/template"

	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	EdgeIoTUser       = "edgeiot"
	EdgeIoTGroup      = "edgeiot"
	EdgeIoTUIPriority = 100
	DefaultReplicas   = 1
	MaxReplicas       = 4

	// base services
	EdgeIoTGatewayProxy    = "gateway-proxy"
	EdgeIoTGateway         = "gateway"
	EdgeIoTStaticWebServer = "static-web-server"
	EdgeIoTInstallation    = "installation"
	EdgeIoTModel           = "model"
	EdgeIoTTimeSeries      = "time-series"
	EdgeIotJob             = "job"

	// offering services
	EdgeIotEvent   = "event"
	EdgeIoTNotify  = "notification"
	EdgeIoTRule    = "rule"
	EdgeIoTControl = "control"
	EdgeIotRollup  = "rollup"

	// APIs
	APIGatewayProxy    = "iot-gateway-proxy"
	APIStaticWebServer = "static-web-server"
	APIGateway         = "harns-gateway"
	APIInstallation    = installation.ComponentInstallation
	APIModel           = model.ComponentModel
	APIDataCollector   = datacollector.ComponentDataCollector
	APIDataBroker      = databroker.ComponentDataBroker
	APIDataQuery       = dataquery.ComponentDataQuery
	APIRule            = rule.ComponentRule
	APINotify          = notification.ComponentNotification
	APIEvent           = event.ComponentEvent
	APIDataRollup      = datarollup.ComponentDataRollup
	APIJobConsumer     = consumer.ComponentConsumer
	APIControl         = control.ComponentControl

	APIInstallModel   = "model"
	APIInstallData    = "data"
	APIInstallEvent   = "event"
	APIInstallNotify  = "notification"
	APIInstallControl = "control"

	// UIs
	UIHelp      = "help"
	UILaunchpad = "launchpad"
	UIOsBar     = "osbar"
	UIModel     = "modelmanager"
	UIData      = "fleetmanager"
	UIEvent     = "eventmanager"
	UIRule      = "ruleengine"
	UINotify    = "notifymanager"
	UIControl   = "controlmanager"

	// installer
	edgeIoTConfig          = "/edgeiot/"
	edgeIoTConfigd         = "/edgeiot/conf.d"
	edgeIoTData            = "/edgeiot/"
	edgeIoTRuntime         = "/edgeiot/"
	edgeIoTExecutableFiles = "/edgeiot/bin"
	edIoTApps              = "/edgeiot/app"
	edIoTApis              = "/edgeiot/api"

	// bucket
	// !!! IMPORTANT !!!
	// For new bucket name, it should NOT exceed 9 bytes.
	// If it exceeds, please evaluate whether update ValidateTypeName in lightiot/pkg/model/validation or not
	dbEventRaw  = "event-raw"
	dbCmdRaw    = "cmd-raw"
	dbActionRaw = "act-raw"
	dbDataRup   = "data-rup"
	dbJob       = "jobs"
	dbConsumer  = "consumers"
)

type EdgeIoTConfig struct {
	StartCmd string
	User     string
	Group    string
}

var (
	EdgeIoTSvcTmpl, _ = template.New("edgeiotsvc").Parse(`[Unit]
Description=EdgeIoT service daemon 

[Service]
User={{.User}}
Group={{.Group}}
ExecStart={{.StartCmd}}
ExecStop=/bin/kill -s TERM $MAINPID
Restart=always
RestartSec=2s
PrivateTmp=true

[Install]
WantedBy=multi-user.target
`)

	apiSvcTmpl, _ = template.New("apisvc").Parse(`[Unit]
Description=EdgeIoT 3rd service daemon 

[Service]
User={{.User}}
Group={{.Group}}
ExecStart={{.StartCmd}}
ExecStop=/bin/kill -s TERM $MAINPID
Restart=always
RestartSec=2s
PrivateTmp=true

[Install]
WantedBy=multi-user.target
`)
)

type Service struct {
	Name     string
	Priority int // valid range is (-100, 100)
	APIs     []APIService
	UIs      []string
}

type APIService struct {
	Name     string
	Replicas int
}

var EdgeIoTServices = []Service{
	{EdgeIoTInstallation, -99, []APIService{{APIInstallation, DefaultReplicas}}, nil},
	{EdgeIoTGatewayProxy, -90, []APIService{{APIGatewayProxy, DefaultReplicas}}, nil},
	{EdgeIoTStaticWebServer, -80, []APIService{{APIStaticWebServer, DefaultReplicas}}, nil},
	{EdgeIoTGateway, -70, []APIService{{APIGateway, DefaultReplicas}}, []string{UILaunchpad, UIOsBar, UIHelp}},
	// all svcs' priority should less than 0 except the above 4 svcs
	{EdgeIoTModel, 0, []APIService{{APIModel, DefaultReplicas}}, []string{UIModel}},
	{EdgeIotEvent, 10, []APIService{{APIEvent, DefaultReplicas}}, []string{UIEvent}},
	{EdgeIoTNotify, 20, []APIService{{APINotify, DefaultReplicas}}, []string{UINotify}},
	{EdgeIoTTimeSeries, 30, []APIService{{APIDataCollector, DefaultReplicas}, {APIDataBroker, DefaultReplicas}, {APIDataQuery, DefaultReplicas}}, []string{UIData}},
	{EdgeIotJob, 40, []APIService{{APIJobConsumer, 2}}, nil},
	{EdgeIotRollup, 50, []APIService{{APIDataRollup, DefaultReplicas}}, nil},
	{EdgeIoTRule, 60, []APIService{{APIRule, DefaultReplicas}}, []string{UIRule}},
	{EdgeIoTControl, 70, []APIService{{APIControl, DefaultReplicas}}, []string{UIControl}},
}

var (
	DependOnTSDBAPIs = sets.NewString(APIDataCollector, APIDataQuery, APIEvent, APIControl, APIJobConsumer, APIDataRollup, APIModel)
	EdgeIotAPIs      = sets.NewString(EdgeIoTModel, EdgeIotEvent, EdgeIoTNotify, EdgeIoTControl, EdgeIoTInstallation, APIInstallData, APIJobConsumer)
	EdgeIotUIs       = sets.NewString(UIData, UIControl, UIModel, UIRule, UIOsBar, UILaunchpad, UIHelp, UINotify, UIEvent)
	EdgeIoTOfferings = sets.NewString(EdgeIotEvent, EdgeIoTRule, EdgeIoTNotify, EdgeIoTControl, EdgeIotRollup)
	EdgeIoTBaseSvcs  = sets.NewString(
		EdgeIoTGatewayProxy,
		EdgeIoTGateway,
		EdgeIoTStaticWebServer,
		EdgeIoTInstallation,
		EdgeIoTModel,
		EdgeIoTTimeSeries,
		EdgeIotJob,
		ToolTypeToString[ToolMQTT],
		ToolTypeToString[ToolTimeSeriesDB],
	)
	EdgeIoTAllSvcs = sets.NewString(EdgeIoTBaseSvcs.UnsortedList()...).Insert(EdgeIoTOfferings.UnsortedList()...)
)

// EdgeIoTAPIEndpoints doesn't include APIInstallation which registers itself when first start
var EdgeIoTAPIEndpoints = map[string]Endpoint{
	APIModel:         {"", []v1.EndpointAction{v1.EndpointActionGet, v1.EndpointActionPost, v1.EndpointActionDelete, v1.EndpointActionPut, v1.EndpointActionPatch}},
	APIDataCollector: {"/timeseries", []v1.EndpointAction{v1.EndpointActionDelete, v1.EndpointActionPut}},
	APIDataQuery:     {"/timeseries", []v1.EndpointAction{v1.EndpointActionGet}},
	APIDataRollup:    {"/rollup", []v1.EndpointAction{v1.EndpointActionGet}},
	APIDataBroker:    {"/exchange", []v1.EndpointAction{v1.EndpointActionPost}},
	APIEvent:         {"", []v1.EndpointAction{v1.EndpointActionGet, v1.EndpointActionPost, v1.EndpointActionDelete, v1.EndpointActionPut, v1.EndpointActionPatch}},
	APIRule:          {"/rules", []v1.EndpointAction{v1.EndpointActionGet, v1.EndpointActionPost, v1.EndpointActionDelete, v1.EndpointActionPut, v1.EndpointActionPatch}},
	APINotify:        {"", []v1.EndpointAction{v1.EndpointActionGet, v1.EndpointActionPost, v1.EndpointActionPut, v1.EndpointActionDelete}},
	APIControl:       {"", []v1.EndpointAction{v1.EndpointActionGet, v1.EndpointActionPost, v1.EndpointActionPut, v1.EndpointActionPatch, v1.EndpointActionDelete}},
}

var EdgeIoTAPIComponents = map[string][]Endpoint{
	APIInstallModel: {
		EdgeIoTAPIEndpoints[APIModel],
	},
	APIInstallData: {
		EdgeIoTAPIEndpoints[APIDataCollector],
		EdgeIoTAPIEndpoints[APIDataQuery],
		EdgeIoTAPIEndpoints[APIDataBroker],
		EdgeIoTAPIEndpoints[APIDataRollup],
		EdgeIoTAPIEndpoints[APIRule],
	},
	APIInstallEvent: {
		EdgeIoTAPIEndpoints[APIEvent],
	},
	APIInstallNotify: {
		EdgeIoTAPIEndpoints[APINotify],
	},
	APIInstallControl: {
		EdgeIoTAPIEndpoints[APIControl],
	},
}

var EdgeIoTAPIInstallations = map[string]string{
	APIModel:         APIInstallModel,
	APIDataCollector: APIInstallData,
	APIDataQuery:     APIInstallData,
	APIDataBroker:    APIInstallData,
	APIDataRollup:    APIInstallData,
	APIRule:          APIInstallData,
	APIEvent:         APIInstallEvent,
	APINotify:        APIInstallNotify,
	APIControl:       APIInstallControl,
}

var EdgeIoTUIInstallations = map[string]string{
	UIModel:     UIModel,
	UIHelp:      UIHelp,
	UILaunchpad: UILaunchpad,
	UIOsBar:     UIOsBar,
	UIData:      UIData,
	UIEvent:     UIData,
	UIRule:      UIRule,
	UINotify:    UINotify,
	UIControl:   UIControl,
}

var EdgeIoTUIDisplayNames = map[string]string{
	UIModel:   "模型管理",
	UIData:    "Fleet Manager",
	UIRule:    "规则引擎",
	UINotify:  "通知服务",
	UIControl: "控制服务",
}

type Bucket struct {
	Name string
	TTL  int // unit is day
}

var EdgeIoTBuckets = map[string][]Bucket{
	EdgeIotEvent:   {{dbEventRaw, 7}},
	EdgeIoTControl: {{dbCmdRaw, 30}, {dbActionRaw, 30}},
	EdgeIotJob:     {{dbJob, 7}, {dbConsumer, 7}},
	EdgeIotRollup:  {{dbDataRup, 90}},
}

func GetEdgeIoTExecutableFilesPath() string {
	return util.GetAbsoluteRuntimePath(edgeIoTExecutableFiles)
}

func GetEdgeIoTRuntimePath() string {
	return util.GetAbsoluteRuntimePath(edgeIoTRuntime)
}

func GetEdgeIoTAppFilesPath() string {
	return util.GetAbsoluteRuntimePath(edIoTApps)
}

func GetEdgeIoTApiFilesPath() string {
	return util.GetAbsoluteRuntimePath(edIoTApis)
}

func GetEdgeIoTConfigPath() string {
	return util.GetAbsoluteConfigPath(edgeIoTConfig)
}

func GetEdgeIoTConfigdPath() string {
	return util.GetAbsoluteConfigPath(edgeIoTConfigd)
}

func GetEdgeIoTDataPath() string {
	return util.GetAbsoluteDataPath(edgeIoTData)
}

type insRes struct {
	Installations []ins.Installation
}

func GetInstallation(harnsClient client.Client, filter interface{}, ioStreams util.IOStreams) *ins.Installation {
	fb, _ := json.Marshal(filter)

	getResp, err := harnsClient.Get("/api/installation/v1/installations", nil, &url.Values{"filter": []string{string(fb)}})

	if err != nil {
		_, _ = fmt.Fprintf(ioStreams.ErrOut, "Failed to access iot installation manager\n")
		return nil
	}
	defer client.Drain(getResp)
	if getResp.StatusCode == http.StatusOK {
		var res insRes
		if err = json.NewDecoder(getResp.Body).Decode(&res); err != nil {
			_, _ = fmt.Fprintf(ioStreams.ErrOut, "Failed to parse installations\n")
			return nil
		}
		if len(res.Installations) != 0 {
			return &res.Installations[0]
		}
	}
	return nil
}

func GetUiImages(harnsClient client.Client, installationId string, ioStreams util.IOStreams) ([]ins.Image, error) {
	resp, err := harnsClient.Get(fmt.Sprintf("/api/installation/v1/installations/%s/images", installationId), http.Header{}, nil)
	if err != nil {
		_, _ = fmt.Fprintf(ioStreams.ErrOut, "Failed to access iot installation manager\n")
		return nil, err
	}
	defer client.Drain(resp)
	if resp.StatusCode != http.StatusOK {
		_, _ = fmt.Fprintf(ioStreams.ErrOut, "Failed to get ui images\n")
		return nil, err
	}
	images := &struct {
		Images []ins.Image `json:"image"`
	}{
		make([]ins.Image, 0, 0),
	}
	if err = json.NewDecoder(resp.Body).Decode(images); err != nil {
		_, _ = fmt.Fprintf(ioStreams.ErrOut, "Failed to parse ui image\n")
		return nil, err
	}
	return images.Images, nil
}

func UploadImage(img string, i ins.Installation, ioStreams util.IOStreams) error {
	imgName := filepath.Base(img)
	bodyBuffer := &bytes.Buffer{}
	writer := multipart.NewWriter(bodyBuffer)
	fw, err := writer.CreateFormFile("file", imgName)
	if err != nil {
		_, _ = fmt.Fprintf(ioStreams.ErrOut, "Failed to create mulitpart file\n")
		return err
	}
	file, err := os.Open(img)
	if err != nil {
		_, _ = fmt.Fprintf(ioStreams.ErrOut, "Failed to open image file\n")
		return err
	}
	defer file.Close()
	if _, err = io.Copy(fw, file); err != nil {
		_, _ = fmt.Fprintf(ioStreams.ErrOut, "Failed to copy image file\n")
		return err
	}

	contentType := writer.FormDataContentType()
	if err = writer.Close(); err != nil {
		_, _ = fmt.Fprintf(ioStreams.ErrOut, "Failed to assemble mulitpart file\n")
		return err
	}

	header := http.Header{"Content-Type": []string{contentType}}
	imgPath := fmt.Sprintf("/api/installation/v1/installations/%s/images", i.ID)
	data := map[string]string{
		"name": i.Name,
	}
	imgPath = buildUrl(imgPath, data)

	req, err := http.NewRequest(http.MethodPut, "http://127.0.0.1"+imgPath, bodyBuffer)
	if err != nil {
		_, _ = fmt.Fprintf(ioStreams.ErrOut, "Failed to create upload image resquest\n")
		return err
	}
	req.Header = header
	c := &http.Client{}
	resp, err := c.Do(req)
	if err != nil {
		_, _ = fmt.Fprintf(ioStreams.ErrOut, "Failed to upload image\n")
		return err
	}
	defer client.Drain(resp)
	if resp.StatusCode != http.StatusOK {
		if ret, err := io.ReadAll(resp.Body); err != nil {
			return err
		} else {
			_, _ = fmt.Fprintf(ioStreams.ErrOut, "Failed to upload image because %s\n", string(ret))
		}
		return fmt.Errorf("upload image failed")
	}
	_, _ = fmt.Fprintf(ioStreams.Out, "Uploaded image %s for installation %s\n", img, i.Name)
	return nil
}

func DeleteUiImages(harnsClient client.Client, imgName, installationId string, ioStreams util.IOStreams) error {
	resp, err := harnsClient.Delete(fmt.Sprintf("/api/installation/v1/installations/%s/images/%s", installationId, imgName))
	if err != nil {
		_, _ = fmt.Fprintf(ioStreams.ErrOut, "Failed to access iot installation manager\n")
		return err
	}
	defer client.Drain(resp)
	if resp.StatusCode != http.StatusNoContent {
		_, _ = fmt.Fprintf(ioStreams.ErrOut, "Failed to delete ui images\n")
		return err
	}
	return nil
}

func buildUrl(path string, queryParams map[string]string) string {
	u, _ := url.Parse(path)
	q := u.Query()
	for k, v := range queryParams {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()
	return u.String()
}
