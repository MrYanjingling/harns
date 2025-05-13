package rule

import (
	"encoding/json"
	"k8s.io/klog/v2"
	"lightiot/pkg/apis"
	"lightiot/pkg/client"
	notification "lightiot/pkg/notification/runtime"
	"net/http"
)

type notifyServer struct {
	notificationClient client.Client
	finished           bool
	isEmailConfigured  bool
	isWeComConfigured  bool
	isWeChatConfigured bool
}

func (ns *notifyServer) getConfig() error {
	if ns.finished {
		return nil
	}

	resp, err := ns.notificationClient.Get("/api/notification/v1/serverConfigurations", nil, nil)
	if err != nil {
		klog.V(1).InfoS("Failed to request notification serverConfigurations", "err", err)
		return apis.ErrInternal
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		ncs := struct {
			Servers []*notification.Config `json:"serverConfigurations"`
		}{}

		if err = json.NewDecoder(resp.Body).Decode(&ncs); err != nil {
			klog.V(2).InfoS("Failed decode request body", "err", err)
			return apis.ErrInternal
		}
		if len(ncs.Servers) != 0 {
			nc := ncs.Servers[0]
			ns.isEmailConfigured = nc.Email != nil
			ns.isWeComConfigured = nc.WeCom != nil
			ns.isWeChatConfigured = nc.WeChat != nil
			ns.finished = ns.isEmailConfigured && ns.isWeComConfigured && ns.isWeChatConfigured
		}
		return nil
	} else {
		klog.V(2).InfoS("Failed to get notification serverConfigurations", "httpCode", resp.StatusCode)
		return apis.ErrInternal
	}
}

func (ns *notifyServer) IsEmailConfigured() bool {
	_ = ns.getConfig()
	return ns.isEmailConfigured
}

func (ns *notifyServer) IsWeComConfigured() bool {
	_ = ns.getConfig()
	return ns.isWeComConfigured
}

func (ns *notifyServer) IsWeChatConfigured() bool {
	_ = ns.getConfig()
	return ns.isWeChatConfigured
}
