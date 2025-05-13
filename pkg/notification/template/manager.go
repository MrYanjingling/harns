package template

import (
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	"lightiot/pkg/apis"
	"lightiot/pkg/apis/response"
	"lightiot/pkg/generic"
	"lightiot/pkg/generic/meta"
	gruntime "lightiot/pkg/generic/runtime"
	"lightiot/pkg/notification/messagetemplate"
	"lightiot/pkg/notification/recipient"
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

type Manager struct {
	tmpls  *sync.Map // id -> *runtime.Template
	tCh    chan gruntime.Object
	tResCh chan *storage.PersistResult
	store  *generic.Store
	stopCh <-chan struct{}

	recipientMgr *recipient.Manager
	msgTmplMgr   *messagetemplate.Manager
}

func NewManager(stopChan <-chan struct{},
	tCh chan gruntime.Object,
	recipientMgr *recipient.Manager,
	msgTmplMgr *messagetemplate.Manager,
) *Manager {
	store, _ := generic.NewStore(gruntime.GroupResource{storage.StoreGroupToString[storage.StoreGroupNotify], storage.Templates}, &runtime.Template{}, tCh)
	return &Manager{
		tmpls:        &sync.Map{},
		tCh:          tCh,
		stopCh:       stopChan,
		store:        store,
		recipientMgr: recipientMgr,
		msgTmplMgr:   msgTmplMgr,
	}
}

func (m *Manager) Init() {
	m.load()
	m.tResCh = m.store.Start(m.stopCh)
}

func (m *Manager) load() {
	ts, _ := m.store.LoadResource()
	for _, obj := range ts {
		t, _ := obj.(*runtime.Template)
		m.tmpls.Store(t.ID, t)
	}
}

func (m *Manager) CreateTmpl(obj *v1.Template) (*runtime.Template, error) {
	if ret, _ := m.ListTmpls(Filter{security.GetTenant(), obj.Name}); len(ret) != 0 {
		return nil, response.ErrResourceExists(obj.Name)
	}

	var invalidRecipients []string
	for _, r := range obj.Recipients {
		if _, err := m.recipientMgr.GetRecipientById(r.Id); err != nil {
			invalidRecipients = append(invalidRecipients, r.Id)
		}
	}
	if len(invalidRecipients) > 0 {
		return nil, response.ErrRecipientNotFound(invalidRecipients)
	}

	var invalidMsgTmpls []string
	for _, mtId := range obj.MessageTemplates {
		if _, err := m.msgTmplMgr.GetMessageTmplById(mtId, true); err != nil {
			invalidMsgTmpls = append(invalidMsgTmpls, mtId)
		}
	}
	if len(invalidMsgTmpls) > 0 {
		return nil, response.ErrMessageTmplNotFound(invalidMsgTmpls)
	}

	rt := &runtime.Template{
		ObjectMeta: meta.ObjectMeta{
			Tenant:  security.GetTenant(),
			Name:    obj.Name,
			ID:      uuidutil.ShortUUID(),
			Version: strconv.FormatUint(randutil.Uint64n(), 10),
			ModTime: time.Now(),
		},
		Subject:          obj.Subject,
		From:             obj.From,
		Recipients:       make([]runtime.TmplRecipient, len(obj.Recipients)),
		MessageTemplates: make([]string, len(obj.MessageTemplates)),
	}

	for i, tr := range obj.Recipients {
		rt.Recipients[i] = runtime.TmplRecipient{
			Id: tr.Id,
		}
	}

	for i, mt := range obj.MessageTemplates {
		rt.MessageTemplates[i] = mt
	}

	m.tCh <- rt
	res := <-m.tResCh
	if res.Err != nil {
		return nil, apis.ErrInternal
	}
	m.tmpls.Store(rt.ID, res.Saved)

	klog.V(2).InfoS("Created template", "id", rt.ID)

	return rt, nil
}

func (m *Manager) ListTmpls(filter Filter) ([]*runtime.Template, error) {
	ts := make([]*runtime.Template, 0)
	predicates := parseFilter(filter)

	// descend
	byModTime := func(mt1, mt2 *runtime.Template) bool { return mt1.ModTime.Before(mt2.ModTime) }
	sorter := By(byModTime)

	m.tmpls.Range(func(key, value interface{}) bool {
		isMatch := true
		v := value.(*runtime.Template)
		for _, p := range predicates {
			if !p(v) {
				isMatch = false
				break
			}
		}
		if isMatch {
			ts = sorter.Insert(ts, v)
		}
		return true
	})

	return ts, nil
}

func (m *Manager) GetTmplById(id string) (*runtime.Template, error) {
	r, isExist := m.tmpls.Load(id)
	if !isExist {
		return nil, os.ErrNotExist
	}
	return r.(*runtime.Template), nil
}

func (m *Manager) DeleteTmpl(tmplId, eTag string) (*runtime.Template, error) {
	tv, exist := m.tmpls.Load(tmplId)
	if !exist {
		return nil, os.ErrNotExist
	}

	t := tv.(*runtime.Template)
	if t.Version != eTag {
		return nil, apis.ErrMismatch
	}

	m.tCh <- &runtime.Template{ObjectMeta: meta.ObjectMeta{ID: tmplId, Version: eTag}}
	res := <-m.tResCh
	if res.Err != nil {
		return nil, res.Err
	}
	m.tmpls.Delete(tmplId)

	klog.V(2).InfoS("Deleted template", "id", t.ID)
	return t, nil
}

// UpdateTemplateByMessageTmplDeletion while a message template is deleted, the templates using the message template will be updated.
// The template will be deleted if it does not contain a valid message template after update.
func (m *Manager) UpdateTemplateByMessageTmplDeletion(mtId string) error {
	var tmpls []*runtime.Template
	m.tmpls.Range(func(key, value interface{}) bool {
		tmpl, _ := value.(*runtime.Template)
		for _, id := range tmpl.MessageTemplates {
			if id == mtId {
				tmpls = append(tmpls, tmpl)
				break
			}
		}
		return true
	})

	for _, tmpl := range tmpls {
		i := 0
		for _, id := range tmpl.MessageTemplates {
			if id != mtId {
				tmpl.MessageTemplates[i] = id
				i++
			}
		}
		tmpl.MessageTemplates = tmpl.MessageTemplates[:i]

		if len(tmpl.MessageTemplates) == 0 {
			m.tCh <- &runtime.Template{ObjectMeta: meta.ObjectMeta{ID: tmpl.ID}}
			res := <-m.tResCh
			if res.Err != nil {
				klog.V(2).InfoS("Failed to delete template", "id", tmpl.ID, "err", res.Err)
			} else {
				m.tmpls.Delete(tmpl.ID)
			}
		} else {
			func(t *runtime.Template) {
				stop := make(chan struct{})
				go func() {
					wait.Until(func() {
						m.tCh <- t
						res := <-m.tResCh
						if res.Err == nil {
							saved, _ := res.Saved.(*runtime.Template)
							saved.ModTime = time.Now()
							m.tmpls.Store(t.ID, saved)
							close(stop)
						}
					}, 0, stop)
				}()
			}(tmpl)
		}
	}

	return nil
}

// UpdateTemplateByRecipientDeletion while a recipient is deleted, the templates using the recipient will be updated.
// The template will be deleted if it does not contain a valid recipient after update.
func (m *Manager) UpdateTemplateByRecipientDeletion(recipientID string) error {
	var tmpls []*runtime.Template
	m.tmpls.Range(func(key, value interface{}) bool {
		tmpl, _ := value.(*runtime.Template)
		for _, r := range tmpl.Recipients {
			if r.Id == recipientID {
				tmpls = append(tmpls, tmpl)
				break
			}
		}
		return true
	})
	for _, tmpl := range tmpls {
		i := 0
		for _, r := range tmpl.Recipients {
			if r.Id != recipientID {
				tmpl.Recipients[i] = r
				i++
			}
		}
		tmpl.Recipients = tmpl.Recipients[:i]

		if len(tmpl.Recipients) == 0 {
			m.tCh <- &runtime.Template{ObjectMeta: meta.ObjectMeta{ID: tmpl.ID}}
			res := <-m.tResCh
			if res.Err != nil {
				klog.V(2).InfoS("Failed to delete template", "id", tmpl.ID, "err", res.Err)
			} else {
				m.tmpls.Delete(tmpl.ID)
			}
		} else {
			func(t *runtime.Template) {
				stop := make(chan struct{})
				go func() {
					wait.Until(func() {
						m.tCh <- t
						res := <-m.tResCh
						if res.Err == nil {
							saved, _ := res.Saved.(*runtime.Template)
							saved.ModTime = time.Now()
							m.tmpls.Store(t.ID, saved)
							close(stop)
						}
					}, 0, stop)
				}()
			}(tmpl)
		}
	}

	return nil
}

func (m *Manager) IsRecipientReferredByTemplate(recipientID string) bool {
	var referred bool
	m.tmpls.Range(func(key, value interface{}) bool {
		v := value.(*runtime.Template)
		for _, r := range v.Recipients {
			if r.Id == recipientID {
				referred = true
				break
			}
		}
		return !referred
	})
	return referred
}

func (m *Manager) IsMessageTemplateReferredByTemplate(mtId string) bool {
	var referred bool
	m.tmpls.Range(func(key, value interface{}) bool {
		v := value.(*runtime.Template)
		for _, mt := range v.MessageTemplates {
			if mt == mtId {
				referred = true
				break
			}
		}
		return !referred
	})
	return referred
}
