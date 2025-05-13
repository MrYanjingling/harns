package message

import (
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"
	"lightiot/pkg/apis/response"
	"lightiot/pkg/client"
	"lightiot/pkg/generic/meta"
	gruntime "lightiot/pkg/generic/runtime"
	"lightiot/pkg/notification/config"
	"lightiot/pkg/notification/messagetemplate"
	"lightiot/pkg/notification/notify"
	"lightiot/pkg/notification/notify/email"
	"lightiot/pkg/notification/notify/wechat"
	"lightiot/pkg/notification/notify/wecom"
	"lightiot/pkg/notification/recipient"
	"lightiot/pkg/notification/runtime"
	"lightiot/pkg/notification/template"
	v1 "lightiot/pkg/notification/v1"
	"sync"
	"time"
)

type Manager struct {
	Messages     *sync.Map // id -> *runtime.Message
	mCh          chan gruntime.Object
	stopCh       <-chan struct{}
	serverMgr    *config.Manager
	recipientMgr *recipient.Manager
	msgTmplMgr   *messagetemplate.Manager
	tmplMgr      *template.Manager
	weChatClient client.Client
	weComClient  client.Client
	notifiers    map[v1.ChannelType]notify.Notifier
}

func NewManager(stopChan <-chan struct{},
	mCh chan gruntime.Object,
	serverMgr *config.Manager,
	recipientMgr *recipient.Manager,
	msgTmplMgr *messagetemplate.Manager,
	tmplMgr *template.Manager,
	weChatClient client.Client,
	weComClient client.Client,
) *Manager {
	return &Manager{
		Messages:     &sync.Map{},
		mCh:          mCh,
		stopCh:       stopChan,
		serverMgr:    serverMgr,
		recipientMgr: recipientMgr,
		msgTmplMgr:   msgTmplMgr,
		tmplMgr:      tmplMgr,
		weChatClient: weChatClient,
		weComClient:  weComClient,
		notifiers:    map[v1.ChannelType]notify.Notifier{},
	}
}

func (m *Manager) Init() {
	m.notifiers[v1.ChannelTypeEmail] = email.New(m.serverMgr)
	m.notifiers[v1.ChannelTypeWeCom] = wecom.New(m.serverMgr, m.weComClient)
	m.notifiers[v1.ChannelTypeWeChat] = wechat.New(m.serverMgr, m.weChatClient)
}

func (m *Manager) SendMessage(obj *v1.Message) error {
	configValidate := func(serverMgr *config.Manager) error { return nil }
	if cv, ok := configValidator[obj.Channel]; ok {
		configValidate = cv
	}
	if err := configValidate(m.serverMgr); err != nil {
		return err
	}
	mt, err := m.parseMessageType(obj)
	if err != nil {
		return err
	}

	rm := &runtime.Message{
		ObjectMeta: meta.ObjectMeta{ModTime: time.Now()},
		Message: &v1.Message{
			Channel:    obj.Channel,
			Subject:    obj.Subject,
			From:       obj.From,
			Payload:    obj.Payload,
			To:         obj.To,
			TemplateId: obj.TemplateId,
		},
		MsgTmpls: make([]*runtime.MessageTemplate, 0),
	}

	if mt == msgTypeSimple {
		rm.MsgTmpls = append(rm.MsgTmpls, simpleMessageTmpl)

		if len(rm.From) == 0 {
			s, _ := m.serverMgr.GetConfig()
			rm.From = s.Email.Smtp.Username
		}
	}

	if mt == msgTypeTemplate {
		tos := sets.String{}
		t, _ := m.tmplMgr.GetTmplById(obj.TemplateId)
		for _, r := range t.Recipients {
			if rr, err := m.recipientMgr.GetRecipientById(r.Id); err == nil {
				for _, ri := range rr.Detail {
					if ri.Type == obj.Channel {
						tos.Insert(ri.Address)
					}
				}
			}
		}
		rm.To = append(rm.To, tos.UnsortedList()...)
		for _, mtId := range t.MessageTemplates {
			if msgTmpl, err := m.msgTmplMgr.GetMessageTmplById(mtId, true); err == nil {
				rm.MsgTmpls = append(rm.MsgTmpls, msgTmpl)
			}
		}
		for _, msgTmpl := range rm.MsgTmpls {
			if msgTmpl.Placeholder == nil {
				continue
			}
			for _, item := range msgTmpl.Placeholder.Placeholders {
				if _, ok := rm.Payload[item.Name]; !ok && item.DefaultValue != nil{
					rm.Payload[item.Name] = item.DefaultValue
				}
			}
		}
		if len(rm.Subject) == 0 {
			rm.Subject = t.Subject
		}
		if len(rm.From) == 0 {
			rm.From = t.From
		}
		if len(rm.From) == 0 {
			s, _ := m.serverMgr.GetConfig()
			rm.From = s.Email.Smtp.Username
		}
	}

	if n, ok := m.notifiers[rm.Channel]; ok {
		go func(n notify.Notifier) {
			_, err := n.Notify(m.stopCh, rm)
			if err != nil {
				klog.V(2).InfoS("Failed to notify", "err", err, "channelType", v1.ChannelTypeToString[rm.Channel])
			}
		}(n)
	}
	return nil
}

func (m *Manager) parseMessageType(obj *v1.Message) (msgType, error) {
	var mt msgType

	if len(obj.To) != 0 {
		if _, ok := obj.Payload[simpleMessagePayloadFieldContent]; ok {
			mt = msgTypeSimple
			return mt, nil
		} else {
			return mt, response.ErrSimpleMessageContentNotFound
		}
	}

	if len(obj.TemplateId) != 0 {
		mt = msgTypeTemplate
		if _, err := m.tmplMgr.GetTmplById(obj.TemplateId); err != nil {
			return mt, response.ErrTemplateNotFound(obj.TemplateId)
		}
	}

	return mt, nil
}

var configValidator = map[v1.ChannelType]func(serverMgr *config.Manager) error{
	v1.ChannelTypeEmail: func(serverMgr *config.Manager) error {
		if !serverMgr.IsEmailConfigured() {
			return response.ErrEmailServerUnconfigured
		} else {
			return nil
		}
	},
	v1.ChannelTypeWeCom: func(serverMgr *config.Manager) error {
		if !serverMgr.IsWeComConfigured() {
			return response.ErrWeComServerUnconfigured
		} else {
			return nil
		}
	},
	v1.ChannelTypeWeChat: func(serverMgr *config.Manager) error {
		if !serverMgr.IsWeChatConfigured() {
			return response.ErrWeChatServerUnconfigured
		} else {
			return nil
		}
	},
}
