package messagetemplate

import (
	"fmt"
	htmlTmpl "html/template"
	"k8s.io/klog/v2"
	"lightiot/pkg/apis"
	"lightiot/pkg/apis/response"
	"lightiot/pkg/generic"
	"lightiot/pkg/generic/meta"
	gruntime "lightiot/pkg/generic/runtime"
	"lightiot/pkg/notification/runtime"
	v1 "lightiot/pkg/notification/v1"
	"lightiot/pkg/storage"
	"lightiot/pkg/util/randutil"
	"lightiot/pkg/util/security"
	"lightiot/pkg/util/uuidutil"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	textTmpl "text/template"
	"time"
)

type UpdateTemplateByMessageTemplateIdFunc func(mtId string) error

type Manager struct {
	msgTmpls           *sync.Map // id -> *runtime.MessageTemplate
	mtCh               chan gruntime.Object
	mtResCh            chan *storage.PersistResult
	store              *generic.Store
	stopCh             <-chan struct{}
	updateTemplateFunc UpdateTemplateByMessageTemplateIdFunc
	forbidDeletionFunc runtime.IsReferredByTemplateFunc
}

func NewManager(stopChan <-chan struct{}, mtCh chan gruntime.Object, opts ...Option) *Manager {
	store, _ := generic.NewStore(gruntime.GroupResource{storage.StoreGroupToString[storage.StoreGroupNotify], storage.MessageTmpls}, &runtime.MessageTemplate{}, mtCh)
	m := &Manager{
		msgTmpls: &sync.Map{},
		mtCh:     mtCh,
		stopCh:   stopChan,
		store:    store,
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

type Option func(m *Manager)

func WithUpdateTemplateFunc(f UpdateTemplateByMessageTemplateIdFunc) Option {
	return func(m *Manager) {
		m.updateTemplateFunc = f
	}
}

func WithForbidDeletionFunc(f runtime.IsReferredByTemplateFunc) Option {
	return func(m *Manager) {
		m.forbidDeletionFunc = f
	}
}

func (m *Manager) Init() {
	m.load()
	m.mtResCh = m.store.Start(m.stopCh)
}

func (m *Manager) load() {
	mts, _ := m.store.LoadResource()
	for _, obj := range mts {
		mt, _ := obj.(*runtime.MessageTemplate)
		tmpl := parseContent(mt.Type, mt.Content)
		if mt.Type == v1.ContentTypeText {
			mt.ParsedTmpl.TextTmpl = tmpl.text
		} else if mt.Type == v1.ContentTypeHtml {
			mt.ParsedTmpl.HtmlTmpl = tmpl.html
		}
		m.msgTmpls.Store(mt.ID, mt)
	}
}

type tmpl struct {
	text *textTmpl.Template
	html *htmlTmpl.Template
	phs  []runtime.PlaceholderItem
}

func (m *Manager) CreateMessageTmpls(obj *v1.MessageTmpl) (*runtime.MessageTemplate, error) {
	if ret, _ := m.ListMessageTmpls(Filter{security.GetTenant(), obj.Name}); len(ret) != 0 {
		return nil, response.ErrResourceExists(obj.Name)
	}

	tmpl := parseContent(obj.Type, obj.Content)

	version := strconv.FormatUint(randutil.Uint64n(), 10)

	ph := &runtime.TemplatePlaceholder{
		ETag:         version,
		Placeholders: tmpl.phs,
	}

	rmt := &runtime.MessageTemplate{
		ObjectMeta: meta.ObjectMeta{
			Tenant:  security.GetTenant(),
			Name:    obj.Name,
			ID:      uuidutil.ShortUUID(),
			Version: version,
			ModTime: time.Now(),
		},
		Type:        obj.Type,
		Content:     obj.Content,
		Placeholder: ph,
	}

	if rmt.Type == v1.ContentTypeText {
		rmt.ParsedTmpl.TextTmpl = tmpl.text
	} else if rmt.Type == v1.ContentTypeHtml {
		rmt.ParsedTmpl.HtmlTmpl = tmpl.html
	}

	m.mtCh <- rmt
	res := <-m.mtResCh
	if res.Err != nil {
		return nil, apis.ErrInternal
	}
	m.msgTmpls.Store(rmt.ID, res.Saved)

	klog.V(2).InfoS("Created messageTemplate", "id", rmt.ID)

	ret := *rmt
	ret.Placeholder = nil
	return &ret, nil
}

func (m *Manager) ListMessageTmpls(filter Filter) ([]*runtime.MessageTemplate, error) {
	mts := make([]*runtime.MessageTemplate, 0)
	predicates := parseFilter(filter)

	// descend
	byModTime := func(mt1, mt2 *runtime.MessageTemplate) bool { return mt1.ModTime.Before(mt2.ModTime) }
	sorter := By(byModTime)

	m.msgTmpls.Range(func(key, value interface{}) bool {
		isMatch := true
		v := value.(*runtime.MessageTemplate)
		for _, p := range predicates {
			if !p(v) {
				isMatch = false
				break
			}
		}
		if isMatch {
			ret := *v
			ret.Placeholder = nil
			mts = sorter.Insert(mts, &ret)
		}
		return true
	})

	return mts, nil
}

func (m *Manager) GetMessageTmplById(id string, exploded bool) (*runtime.MessageTemplate, error) {
	mt, isExist := m.msgTmpls.Load(id)
	if !isExist {
		return nil, os.ErrNotExist
	}

	ret := mt.(*runtime.MessageTemplate)
	if !exploded {
		tmp := *ret
		tmp.Placeholder = nil
		ret = &tmp
	}

	return ret, nil
}

func (m *Manager) DeleteMsgTmpl(mtId, eTag string, orphanRemoval bool) (*runtime.MessageTemplate, error) {
	mtv, exist := m.msgTmpls.Load(mtId)
	if !exist {
		return nil, os.ErrNotExist
	}

	mt := mtv.(*runtime.MessageTemplate)
	if mt.Version != eTag {
		return nil, apis.ErrMismatch
	}

	if !orphanRemoval && m.forbidDeletionFunc(mtId) {
		return nil, response.ErrAssociateResourceExist(mtId)
	}

	m.mtCh <- mt
	res := <-m.mtResCh
	if res.Err != nil {
		return nil, res.Err
	}
	m.msgTmpls.Delete(mtId)
	if orphanRemoval {
		_ = m.updateTemplateFunc(mt.ID)
	}

	klog.V(2).InfoS("Deleted messageTemplate", "id", mt.ID)
	return mt, nil
}

func (m *Manager) getPlaceholders(id string) (*runtime.TemplatePlaceholder, error) {
	mt, isExist := m.msgTmpls.Load(id)
	if !isExist {
		return nil, os.ErrNotExist
	}
	return mt.(*runtime.MessageTemplate).Placeholder, nil
}

func (m *Manager) updatePlaceholders(id, version string, obj, old *runtime.TemplatePlaceholder) (*runtime.TemplatePlaceholder, error) {
	origin, isExist := m.msgTmpls.Load(id)
	if !isExist {
		return nil, os.ErrNotExist
	}
	mt, _ := origin.(*runtime.MessageTemplate)

	if version != old.ETag {
		return nil, apis.ErrMismatch
	}

	for _, in := range obj.Placeholders {
		for i, item := range old.Placeholders {
			if item.Name == in.Name {
				old.Placeholders[i].DefaultValue = in.DefaultValue
				break
			}
		}
	}

	updated := *mt
	updated.Placeholder = old

	m.mtCh <- &updated
	res := <-m.mtResCh
	if res.Err != nil {
		return nil, res.Err
	}
	saved := res.Saved.(*runtime.MessageTemplate)
	mt.ModTime = time.Now()
	mt.Placeholder = old
	mt.Version = saved.Version
	mt.Placeholder.ETag = mt.Version

	return mt.Placeholder, nil
}

var parser = map[v1.ContentType]func(string) tmpl{
	v1.ContentTypeText: func(content string) tmpl {
		rgx := regexp.MustCompile(runtime.PlaceholderRegExp)
		placeholders := make([]runtime.PlaceholderItem, 0)
		reps := rgx.ReplaceAllStringFunc(content, func(s string) string {
			ph := strings.TrimSpace(s[2 : len(s)-2])
			placeholders = append(placeholders, runtime.PlaceholderItem{Name: ph})
			repl := fmt.Sprintf("{{.Payload.%s}}", ph)
			return repl
		})
		t, _ := textTmpl.New("").Parse(reps)
		tmpl := tmpl{
			text: t,
			phs:  placeholders,
		}
		return tmpl
	},
	v1.ContentTypeHtml: func(content string) tmpl {
		rgx := regexp.MustCompile(runtime.PlaceholderRegExp)
		placeholders := make([]runtime.PlaceholderItem, 0)
		reps := rgx.ReplaceAllStringFunc(content, func(s string) string {
			ph := strings.TrimSpace(s[2 : len(s)-2])
			placeholders = append(placeholders, runtime.PlaceholderItem{Name: ph})
			repl := fmt.Sprintf("{{.Payload.%s}}", ph)
			return repl
		})
		t, _ := htmlTmpl.New("").Parse(reps)
		tmpl := tmpl{
			html: t,
			phs:  placeholders,
		}
		return tmpl
	},
	v1.ContentTypeMarkdown: func(address string) tmpl {
		return tmpl{}
	},
}

func parseContent(ct v1.ContentType, content string) tmpl {
	return parser[ct](content)
}
