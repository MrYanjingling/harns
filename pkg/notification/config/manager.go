package config

import (
	"k8s.io/klog/v2"
	"lightiot/pkg/apis"
	"lightiot/pkg/apis/response"
	"lightiot/pkg/generic"
	"lightiot/pkg/generic/meta"
	gruntime "lightiot/pkg/generic/runtime"
	"lightiot/pkg/notification/runtime"
	v1 "lightiot/pkg/notification/v1"
	"lightiot/pkg/storage"
	"lightiot/pkg/util/aesutil"
	"lightiot/pkg/util/randutil"
	"lightiot/pkg/util/security"
	"lightiot/pkg/util/uuidutil"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Manager struct {
	servers *sync.Map // tenant -> *runtime.Config
	cCh     chan gruntime.Object
	cResCh  chan *storage.PersistResult
	store   *generic.Store

	stopCh <-chan struct{}
}

func NewManager(stopChan <-chan struct{}, cCh chan gruntime.Object) *Manager {
	store, _ := generic.NewStore(gruntime.GroupResource{storage.StoreGroupToString[storage.StoreGroupNotify], storage.Servers}, &runtime.Config{}, cCh)
	return &Manager{
		servers: &sync.Map{},
		cCh:     cCh,
		stopCh:  stopChan,
		store:   store,
	}
}

func (m *Manager) Init() {
	m.load()
	m.cResCh = m.store.Start(m.stopCh)
}

func (m *Manager) load() {
	cs, _ := m.store.LoadResource()
	for _, obj := range cs {
		c, _ := obj.(*runtime.Config)
		m.servers.Store(c.Tenant, c)
	}
}

func (m *Manager) createConfigs(obj *v1.Config) (*runtime.Config, error) {
	if _, ok := m.servers.Load(security.GetTenant()); ok {
		return nil, response.ErrResourceExists("Config")
	}

	rc := &runtime.Config{
		ObjectMeta: meta.ObjectMeta{
			Tenant:  security.GetTenant(),
			ID:      uuidutil.UUID(),
			Version: strconv.FormatUint(randutil.Uint64n(), 10),
			ModTime: time.Now(),
		},
	}
	if obj.Email != nil {
		rc.Config.Email = &v1.EmailConfig{
			Smtp: v1.SMTPConfig{
				Hostname: obj.Email.Smtp.Hostname,
				Port:     obj.Email.Smtp.Port,
				Username: obj.Email.Smtp.Username,
				Password: v1.Secret(aesutil.EncryptCBC(string(obj.Email.Smtp.Password), GetAesKey(security.GetTenant()))),
			},
		}
	}
	if obj.WeCom != nil {
		rc.Config.WeCom = &v1.WeComConfig{
			CorpId:     obj.WeCom.CorpId,
			CorpSecret: v1.Secret(aesutil.EncryptCBC(string(obj.WeCom.CorpSecret), GetAesKey(security.GetTenant()))),
			AgentId:    obj.WeCom.AgentId,
		}
	}
	if obj.WeChat != nil {
		rc.Config.WeChat = &v1.WeChatConfig{
			AppId:     obj.WeChat.AppId,
			AppSecret: v1.Secret(aesutil.EncryptCBC(string(obj.WeChat.AppSecret), GetAesKey(security.GetTenant()))),
		}
	}

	m.cCh <- rc
	res := <-m.cResCh
	if res.Err != nil {
		return nil, apis.ErrInternal
	}
	m.servers.Store(rc.Tenant, res.Saved)

	klog.V(2).InfoS("Created serverConfig", "id", rc.ID)
	return rc, nil
}

func (m *Manager) GetConfig() (*runtime.Config, error) {
	if c, ok := m.servers.Load(security.GetTenant()); ok {
		return c.(*runtime.Config), nil
	} else {
		return nil, os.ErrNotExist
	}
}

func (m *Manager) updateConfigs(id, eTag string, obj *v1.Config, old *runtime.Config) (*runtime.Config, error) {
	if old.Version != eTag {
		return nil, apis.ErrMismatch
	}

	old.ModTime = time.Now()

	if obj.Email != nil {
		if old.Config.Email != nil {
			old.Config.Email.Smtp.Hostname = obj.Email.Smtp.Hostname
			old.Config.Email.Smtp.Port = obj.Email.Smtp.Port
			old.Config.Email.Smtp.Username = obj.Email.Smtp.Username
			// The reason we don't handle it on Secret.UnmarshalJSON() is that the new password may be "<secret>"
			if obj.Email.Smtp.Password != v1.SecretPlaceholder {
				old.Config.Email.Smtp.Password = v1.Secret(aesutil.EncryptCBC(string(obj.Email.Smtp.Password), GetAesKey(security.GetTenant())))
			}
		} else {
			old.Config.Email = &v1.EmailConfig{
				Smtp: v1.SMTPConfig{
					Hostname: obj.Email.Smtp.Hostname,
					Port:     obj.Email.Smtp.Port,
					Username: obj.Email.Smtp.Username,
					Password: v1.Secret(aesutil.EncryptCBC(string(obj.Email.Smtp.Password), GetAesKey(security.GetTenant()))),
				},
			}
		}
	} else {
		old.Config.Email = obj.Email
	}

	if obj.WeCom != nil {
		if old.Config.WeCom != nil {
			old.Config.WeCom.CorpId = obj.WeCom.CorpId
			old.Config.WeCom.AgentId = obj.WeCom.AgentId
			if obj.WeCom.CorpSecret != v1.SecretPlaceholder {
				old.Config.WeCom.CorpSecret = v1.Secret(aesutil.EncryptCBC(string(obj.WeCom.CorpSecret), GetAesKey(security.GetTenant())))
			}
		} else {
			old.Config.WeCom = &v1.WeComConfig{
				CorpId:     obj.WeCom.CorpId,
				CorpSecret: v1.Secret(aesutil.EncryptCBC(string(obj.WeCom.CorpSecret), GetAesKey(security.GetTenant()))),
				AgentId:    obj.WeCom.AgentId,
			}
		}
	} else {
		old.Config.WeCom = obj.WeCom
	}

	if obj.WeChat != nil {
		if old.Config.WeChat != nil {
			old.Config.WeChat.AppId = obj.WeChat.AppId
			if obj.WeChat.AppSecret != v1.SecretPlaceholder {
				old.WeChat.AppSecret = v1.Secret(aesutil.EncryptCBC(string(obj.WeChat.AppSecret), GetAesKey(security.GetTenant())))
			}
		} else {
			old.Config.WeChat = &v1.WeChatConfig{
				AppId:     obj.WeChat.AppId,
				AppSecret: v1.Secret(aesutil.EncryptCBC(string(obj.WeChat.AppSecret), GetAesKey(security.GetTenant()))),
			}
		}
	} else {
		old.Config.WeChat = obj.WeChat
	}

	m.cCh <- old
	res := <-m.cResCh
	if res.Err != nil {
		return nil, apis.ErrInternal
	}
	saved := res.Saved.(*runtime.Config)
	des, _ := m.servers.Load(old.Tenant)
	c := des.(*runtime.Config)
	*c = *saved
	return saved, nil
}

func (m *Manager) IsEmailConfigured() bool {
	if c, ok := m.servers.Load(security.GetTenant()); !ok {
		return false
	} else {
		rc := c.(*runtime.Config)
		return rc.Email != nil
	}
}

func (m *Manager) IsWeComConfigured() bool {
	if c, ok := m.servers.Load(security.GetTenant()); !ok {
		return false
	} else {
		rc := c.(*runtime.Config)
		return rc.WeCom != nil
	}
}

func (m *Manager) IsWeChatConfigured() bool {
	if c, ok := m.servers.Load(security.GetTenant()); !ok {
		return false
	} else {
		rc := c.(*runtime.Config)
		return rc.WeChat != nil
	}
}

func GetAesKey(tenant string) string {
	var sb strings.Builder
	pos := len(tenant)
	if pos >= len(keyTmpl) {
		pos = 0
	}
	sb.WriteString(keyTmpl[:pos])
	sb.WriteString(tenant)
	sb.WriteString(keyTmpl[pos:])
	return sb.String()
}
