package common

import (
	"flag"
	"lightiot/test/util/httpclient"
	"net/http"
	"net/url"

	"github.com/onsi/ginkgo"
)

var testOption Option

type Option struct {
	IotModelManagerHost        string
	IotRuleManagerHost         string
	IotControlManagerHost      string
	IotNotificationManagerHost string
	IotEventManagerHost        string
}

func Flags(flags *flag.FlagSet) {
	flags.StringVar(&testOption.IotModelManagerHost, "iot-model-manager-host", "http://127.0.0.1:32100", "Host of iot-model-manager.")
	flags.StringVar(&testOption.IotRuleManagerHost, "iot-rule-manager-host", "http://127.0.0.1:32600", "Host of iot-rule-manager.")
	flags.StringVar(&testOption.IotControlManagerHost, "iot-control-manager-host", "http://127.0.0.1:32130", "Host of iot-control-manager.")
	flags.StringVar(&testOption.IotNotificationManagerHost, "iot-notification-manager-host", "http://127.0.0.1:32660", "Host of iot-notification-manager.")
	flags.StringVar(&testOption.IotEventManagerHost, "iot-event-manager-host", "http://127.0.0.1:32500", "Host of iot-notification-manager.")
}

func CreateRuleClient() httpclient.TestClient {
	return createClient(testOption.IotRuleManagerHost, "/api/data/v1")
}

func CreateControlClient() httpclient.TestClient {
	return createClient(testOption.IotControlManagerHost, "/api/control/v1")
}

func CreateNotificationClient() httpclient.TestClient {
	return createClient(testOption.IotNotificationManagerHost, "/api/notification/v1")
}

func CreateModelClient() httpclient.TestClient {
	return createClient(testOption.IotModelManagerHost, "/api/model/v1")
}

func CreateEventClient() httpclient.TestClient {
	return createClient(testOption.IotEventManagerHost, "/api/event/v1")
}

func createClient(urlStr, basePath string) httpclient.TestClient {
	u, err := url.Parse(urlStr)
	if err != nil {
		ginkgo.Fail("bad url:" + err.Error())
	}

	c := httpclient.NewTestClient(u, basePath)
	checkAlive(c, 3)
	return c
}

func checkAlive(c httpclient.TestClient, count int) {
	for i := 0; !doCheckAlive(c); {
		i++
		if i == count {
			ginkgo.Fail("fail to access server")
		}
	}
}

func doCheckAlive(c httpclient.TestClient) bool {
	res, err := c.Get("/", nil, nil)
	if err != nil {
		return false
	}
	if res.StatusCode == http.StatusNotFound {
		return true
	}

	return false
}
