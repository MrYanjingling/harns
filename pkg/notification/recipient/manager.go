package recipient

import (
	"k8s.io/klog/v2"
	"lightiot/pkg/apis"
	"lightiot/pkg/apis/response"
	"lightiot/pkg/generic"
	"lightiot/pkg/generic/meta"
	gruntime "lightiot/pkg/generic/runtime"
	"lightiot/pkg/notification/config"
	"lightiot/pkg/notification/runtime"
	v1 "lightiot/pkg/notification/v1"
	"lightiot/pkg/storage"
	"lightiot/pkg/util/randutil"
	"lightiot/pkg/util/security"
	"lightiot/pkg/util/uuidutil"
	"os"
	"strconv"
	"sync"
	"time"
)

type UpdateTemplateByRecipientIdFunc func(rId string) error

type Manager struct {
	recipients         *sync.Map // id -> *runtime.Recipient
	rCh                chan gruntime.Object
	rResCh             chan *storage.PersistResult
	store              *generic.Store
	stopCh             <-chan struct{}
	serverMgr          *config.Manager
	updateTemplateFunc UpdateTemplateByRecipientIdFunc
	forbidDeletionFunc runtime.IsReferredByTemplateFunc
}

type Option func(*Manager)

func WithUpdateTemplateFunc(f UpdateTemplateByRecipientIdFunc) Option {
	return func(m *Manager) {
		m.updateTemplateFunc = f
	}
}

func WithForbidDeletionFunc(f runtime.IsReferredByTemplateFunc) Option {
	return func(m *Manager) {
		m.forbidDeletionFunc = f
	}
}

func NewManager(stopChan <-chan struct{}, rCh chan gruntime.Object, serverMgr *config.Manager, opts ...Option) *Manager {
	store, _ := generic.NewStore(gruntime.GroupResource{storage.StoreGroupToString[storage.StoreGroupNotify], storage.Recipients}, &runtime.Recipient{}, rCh)
	m := &Manager{
		recipients: &sync.Map{},
		rCh:        rCh,
		stopCh:     stopChan,
		store:      store,
		serverMgr:  serverMgr,
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

func (m *Manager) Init() {
	m.load()
	m.rResCh = m.store.Start(m.stopCh)
}

func (m *Manager) load() {
	rs, _ := m.store.LoadResource()
	for _, obj := range rs {
		r, _ := obj.(*runtime.Recipient)
		m.recipients.Store(r.ID, r)
	}
}

func (m *Manager) createRecipient(obj *v1.Recipient) (*runtime.Recipient, error) {
	if ret, _ := m.listRecipients(Filter{security.GetTenant(), obj.Name}); len(ret) != 0 {
		return nil, response.ErrResourceExists(obj.Name)
	}

	channels := make(map[v1.ChannelType]struct{}, 0)
	for _, i := range obj.Detail {
		channels[i.Type] = struct{}{}
	}

	for k, _ := range channels {
		if err := m.validateServer(k); err != nil {
			return nil, err
		}
	}

	rr := &runtime.Recipient{
		ObjectMeta: meta.ObjectMeta{
			Tenant:  security.GetTenant(),
			Name:    obj.Name,
			ID:      uuidutil.ShortUUID(),
			Version: strconv.FormatUint(randutil.Uint64n(), 10),
			ModTime: time.Now(),
		},
		Detail: make([]runtime.RecipientIem, len(obj.Detail)),
	}

	for i, item := range obj.Detail {
		rr.Detail[i].Type = item.Type
		rr.Detail[i].Address = item.Address
	}

	m.rCh <- rr
	res := <-m.rResCh
	if res.Err != nil {
		return nil, apis.ErrInternal
	}
	m.recipients.Store(rr.ID, res.Saved)

	klog.V(2).InfoS("Created recipient", "id", rr.ID)
	return rr, nil
}

func (m *Manager) listRecipients(filter Filter) ([]*runtime.Recipient, error) {
	rs := make([]*runtime.Recipient, 0)
	predicates := parseFilter(filter)

	// descend
	byModTime := func(r1, r2 *runtime.Recipient) bool { return r1.ModTime.Before(r2.ModTime) }
	sorter := By(byModTime)

	m.recipients.Range(func(key, value interface{}) bool {
		isMatch := true
		v := value.(*runtime.Recipient)
		for _, p := range predicates {
			if !p(v) {
				isMatch = false
				break
			}
		}
		if isMatch {
			rs = sorter.Insert(rs, v)
		}
		return true
	})

	return rs, nil
}

func (m *Manager) GetRecipientById(id string) (*runtime.Recipient, error) {
	r, isExist := m.recipients.Load(id)
	if !isExist {
		return nil, os.ErrNotExist
	}
	return r.(*runtime.Recipient), nil
}

func (m *Manager) validateServer(channel v1.ChannelType) error {
	switch channel {
	case v1.ChannelTypeEmail:
		if !m.serverMgr.IsEmailConfigured() {
			return response.ErrEmailServerUnconfigured
		}
	case v1.ChannelTypeWeCom:
		if !m.serverMgr.IsWeComConfigured() {
			return response.ErrWeComServerUnconfigured
		}
	case v1.ChannelTypeWeChat:
		if !m.serverMgr.IsWeChatConfigured() {
			return response.ErrWeChatServerUnconfigured
		}
	default:
		return nil
	}
	return nil
}

func (m *Manager) DeleteRecipient(recipientId, eTag string, orphanRemoval bool) (*runtime.Recipient, error) {
	rv, exist := m.recipients.Load(recipientId)
	if !exist {
		return nil, os.ErrNotExist
	}

	r := rv.(*runtime.Recipient)
	if r.Version != eTag {
		return nil, apis.ErrMismatch
	}

	if !orphanRemoval && m.forbidDeletionFunc(recipientId) {
		return nil, response.ErrAssociateResourceExist(recipientId)
	}

	m.rCh <- &runtime.Recipient{ObjectMeta: meta.ObjectMeta{ID: recipientId, Version: eTag}}
	res := <-m.rResCh
	if res.Err != nil {
		return nil, res.Err
	}
	m.recipients.Delete(recipientId)

	if orphanRemoval {
		_ = m.updateTemplateFunc(recipientId)
	}

	klog.V(2).InfoS("Deleted recipient", "id", r.ID)
	return r, nil
}
