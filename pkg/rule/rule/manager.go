package rule

import (
	"context"
	"encoding/json"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	"lightiot/pkg/apis"
	"lightiot/pkg/apis/response"
	"lightiot/pkg/client"
	"lightiot/pkg/generic"
	"lightiot/pkg/generic/meta"
	gruntime "lightiot/pkg/generic/runtime"
	model "lightiot/pkg/model/runtime"
	notificationv1 "lightiot/pkg/notification/v1"
	"lightiot/pkg/rule/runtime"
	v1 "lightiot/pkg/rule/v1"
	"lightiot/pkg/rule/validation"
	"lightiot/pkg/storage"
	"lightiot/pkg/util/randutil"
	"lightiot/pkg/util/security"
	"lightiot/pkg/util/uuidutil"
	validutil "lightiot/pkg/util/validation"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Manager struct {
	Rules              *sync.Map // map[id]*runtime.Rule
	rCh                chan gruntime.Object
	tCh                chan gruntime.Object
	ttCh               chan gruntime.Object
	rResCh             chan *storage.PersistResult
	stopCh             <-chan struct{}
	store              *generic.Store
	tStore             *generic.Store
	ttStore            *generic.Store
	modelClient        client.Client
	notificationClient client.Client
	RuleRest           *REST
}

type Option func(*Manager)

func WithRuleChan(rCh chan gruntime.Object) Option {
	return func(m *Manager) {
		m.rCh = rCh
		m.store, _ = generic.NewStore(gruntime.GroupResource{storage.StoreGroupToString[storage.StoreGroupData], storage.Rules}, &runtime.Rule{}, m.rCh)
	}
}

func WithThingChan(tCh chan gruntime.Object) Option {
	return func(m *Manager) {
		m.tCh = tCh
		m.tStore, _ = generic.NewStore(gruntime.GroupResource{storage.StoreGroupToString[storage.StoreGroupModel], storage.Things}, &model.Thing{}, m.tCh)
	}
}

func WithThingTypeChan(ttCh chan gruntime.Object) Option {
	return func(m *Manager) {
		m.ttCh = ttCh
		m.ttStore, _ = generic.NewStore(gruntime.GroupResource{storage.StoreGroupToString[storage.StoreGroupModel], storage.ThingTypes}, &model.ThingType{}, m.ttCh)
	}
}

func WithModelClient(modelMgrClient client.Client) Option {
	return func(m *Manager) {
		m.modelClient = modelMgrClient
	}
}

func WithNotificationClient(notificationMgrUrl client.Client) Option {
	return func(m *Manager) {
		m.notificationClient = notificationMgrUrl
	}
}

func NewManager(stopCh <-chan struct{}, opts ...Option) *Manager {
	m := &Manager{
		Rules:  &sync.Map{},
		stopCh: stopCh,
	}
	for _, opt := range opts {
		opt(m)
	}

	m.RuleRest = &REST{m}
	return m
}

func (m *Manager) Init() {
	m.load()
	m.rResCh = m.store.Start(m.stopCh)
	_, _ = m.tStore.Watch(m.stopCh, "", "")
	_, _ = m.ttStore.Watch(m.stopCh, "", "")
	go m.processModelEvent(m.stopCh)
}

func (m *Manager) Watch() {
	m.load()
	_, _ = m.store.Watch(m.stopCh, "", "")
	go m.processEvent(m.stopCh)
}

func (m *Manager) load() {
	rs, _ := m.store.LoadResource()
	for _, obj := range rs {
		r, _ := obj.(*runtime.Rule)
		m.Rules.Store(r.ID, r)
	}
}

func (m *Manager) CreateRule(obj *v1.Rule) (*runtime.Rule, error) {
	_, operands, _ := validation.Parse(obj.Evaluations[0].Expression)
	propertiesByPropertySet := map[string][]string{}
	for _, operand := range operands {
		res := strings.Split(operand, runtime.OperandUS)
		ps := res[0]
		p := res[1]
		if properties, ok := propertiesByPropertySet[ps]; ok {
			properties = append(properties, p)
		} else {
			propertiesByPropertySet[ps] = []string{p}
		}
	}
	var err error
	var vp *runtime.VirtualParameter
	for k, v := range propertiesByPropertySet {
		if properties, err := m.getThingProperties(obj.ThingId, k); err != nil {
			return nil, err
		} else {
			for _, p := range v {
				if _, ok := properties[p]; !ok {
					klog.V(3).InfoS("Property not found", "property", p, "propertySet", k, "thingId", obj.ThingId)
					return nil, response.ErrPropertyOfThingNotFound(p, k, obj.ThingId)
				}
			}
			if obj.Actions.VirtualParameter != nil && obj.ThingId == obj.Actions.VirtualParameter.ThingId && k == obj.Actions.VirtualParameter.PropertySetName {
				vp, err = m.createVirtualParameterAction(obj.Actions.VirtualParameter, obj.ThingId, k, properties, operands)
			}
			if err != nil {
				return nil, err
			}
		}
	}
	if obj.Actions.VirtualParameter != nil && vp == nil {
		vp, err = m.createVirtualParameterAction(obj.Actions.VirtualParameter, "", "", map[string]*model.Property{}, operands)
		if err != nil {
			return nil, err
		}
	}

	eventAction, _ := m.createEventAction(obj.Actions.Event)
	webhookAction, err := m.createWebhookAction(obj.Actions.Webhook)
	if err != nil {
		return nil, err
	}

	var emailAction, weComAction, weChatAction runtime.Actioner
	ns := &notifyServer{notificationClient: m.notificationClient}
	emailAction, err = m.createEmailAction(obj.Actions.Email, ns)
	if err != nil {
		return nil, err
	}
	weComAction, err = m.createWeComAction(obj.Actions.WeCom, ns)
	if err != nil {
		return nil, err
	}
	weChatAction, err = m.createWeChatAction(obj.Actions.WeChat, ns)
	if err != nil {
		return nil, err
	}

	rr := &runtime.Rule{
		ObjectMeta: meta.ObjectMeta{
			Tenant:  security.GetTenant(),
			Name:    strings.TrimSpace(obj.Name),
			ID:      uuidutil.ShortUUID(),
			Version: strconv.FormatUint(randutil.Uint64n(), 10),
			ModTime: time.Now(),
		},
		Description: obj.Description,
		ThingId:     obj.ThingId,
		RealTime:    obj.RealTime,
		Active:      obj.Active,
		Evaluations: []runtime.Evaluation{
			{
				Template:   obj.Evaluations[0].Template,
				Expression: obj.Evaluations[0].Expression,
			},
		},
		Actions: runtime.Actions{Actions: make(map[runtime.ActionType]runtime.Actioner, 0)},
	}
	if vp != nil {
		rr.Actions.Actions[runtime.ActionTypeVirtualParameter] = vp
	}
	if eventAction != nil {
		rr.Actions.Actions[runtime.ActionTypeEvent] = eventAction
	}
	if webhookAction != nil {
		rr.Actions.Actions[runtime.ActionTypeWebhook] = webhookAction
	}
	if emailAction != nil {
		rr.Actions.Actions[runtime.ActionTypeEmail] = emailAction
	}
	if weComAction != nil {
		rr.Actions.Actions[runtime.ActionTypeWeCom] = weComAction
	}
	if weChatAction != nil {
		rr.Actions.Actions[runtime.ActionTypeWeChat] = weChatAction
	}

	m.rCh <- rr
	res := <-m.rResCh
	if res.Err != nil {
		return nil, err
	}
	m.Rules.Store(rr.ID, rr)
	klog.V(2).InfoS("Created rule", "id", rr.ID)
	return rr, nil
}

func (m *Manager) GetRules(filter *filter) ([]*runtime.Rule, error) {
	rs := make([]*runtime.Rule, 0)
	predicates := parseFilter(filter)
	// descend
	byModTime := func(r1, r2 *runtime.Rule) bool { return r1.ModTime.Before(r2.ModTime) }
	sorter := By(byModTime)

	m.Rules.Range(func(key, value interface{}) bool {
		isMatch := true
		v := value.(*runtime.Rule)
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

func (m *Manager) GetRuleById(id string) (*runtime.Rule, error) {
	r, isExist := m.Rules.Load(id)
	if !isExist {
		return nil, os.ErrNotExist
	}
	return r.(*runtime.Rule), nil
}

func (m *Manager) UpdateRule(id, version string, obj *v1.Rule, old *runtime.Rule) (*runtime.Rule, error) {
	if version != old.Version {
		return nil, apis.ErrMismatch
	}

	_, operands, _ := validation.Parse(obj.Evaluations[0].Expression)
	propertiesByPropertySet := map[string][]string{}
	for _, operand := range operands {
		res := strings.Split(operand, runtime.OperandUS)
		ps := res[0]
		p := res[1]
		if properties, ok := propertiesByPropertySet[ps]; ok {
			properties = append(properties, p)
		} else {
			propertiesByPropertySet[ps] = []string{p}
		}
	}

	var err error
	var vp *runtime.VirtualParameter
	for k, v := range propertiesByPropertySet {
		if properties, err := m.getThingProperties(obj.ThingId, k); err != nil {
			return nil, err
		} else {
			for _, p := range v {
				if _, ok := properties[p]; !ok {
					klog.V(3).InfoS("Property not found", "property", p, "propertySet", k, "thingId", obj.ThingId)
					return nil, response.ErrPropertyOfThingNotFound(p, k, obj.ThingId)
				}
			}
			if obj.Actions.VirtualParameter != nil && obj.ThingId == obj.Actions.VirtualParameter.ThingId && k == obj.Actions.VirtualParameter.PropertySetName {
				vp, err = m.createVirtualParameterAction(obj.Actions.VirtualParameter, obj.ThingId, k, properties, operands)
			}
			if err != nil {
				return nil, err
			}
		}
	}
	if obj.Actions.VirtualParameter != nil && vp == nil {
		vp, err = m.createVirtualParameterAction(obj.Actions.VirtualParameter, "", "", map[string]*model.Property{}, operands)
		if err != nil {
			return nil, err
		}
	}

	eventAction, _ := m.createEventAction(obj.Actions.Event)
	webhookAction, err := m.createWebhookAction(obj.Actions.Webhook)
	if err != nil {
		return nil, err
	}

	var emailAction, weComAction, weChatAction runtime.Actioner
	ns := &notifyServer{notificationClient: m.notificationClient}
	emailAction, err = m.createEmailAction(obj.Actions.Email, ns)
	if err != nil {
		return nil, err
	}
	weComAction, err = m.createWeComAction(obj.Actions.WeCom, ns)
	if err != nil {
		return nil, err
	}
	weChatAction, err = m.createWeChatAction(obj.Actions.WeChat, ns)
	if err != nil {
		return nil, err
	}

	old.Name = obj.Name
	old.ModTime = time.Now()
	old.Description = obj.Description
	old.ThingId = obj.ThingId
	old.RealTime = obj.RealTime
	old.Active = obj.Active
	old.Evaluations[0].Template = obj.Evaluations[0].Template
	old.Evaluations[0].Expression = obj.Evaluations[0].Expression
	old.Actions.Actions = make(map[runtime.ActionType]runtime.Actioner, 0)
	if vp != nil {
		old.Actions.Actions[runtime.ActionTypeVirtualParameter] = vp
	}
	if eventAction != nil {
		old.Actions.Actions[runtime.ActionTypeEvent] = eventAction
	}
	if webhookAction != nil {
		old.Actions.Actions[runtime.ActionTypeWebhook] = webhookAction
	}
	if emailAction != nil {
		old.Actions.Actions[runtime.ActionTypeEmail] = emailAction
	}
	if weComAction != nil {
		old.Actions.Actions[runtime.ActionTypeWeCom] = weComAction
	}
	if weChatAction != nil {
		old.Actions.Actions[runtime.ActionTypeWeChat] = weChatAction
	}

	m.rCh <- old
	res := <-m.rResCh
	if res.Err != nil {
		return nil, res.Err
	}
	saved := res.Saved.(*runtime.Rule)
	des, _ := m.Rules.Load(saved.ID)
	rule := des.(*runtime.Rule)

	*rule = *saved

	return rule, nil
}

func (m *Manager) DeleteRule(ruleId, version string) (*runtime.Rule, error) {
	rv, exist := m.Rules.Load(ruleId)
	if !exist {
		return nil, os.ErrNotExist
	}

	r := rv.(*runtime.Rule)
	if r.Version != version {
		return nil, apis.ErrMismatch
	}
	m.rCh <- &runtime.Rule{ObjectMeta: meta.ObjectMeta{
		ID:      r.ID,
		Version: r.Version,
	}}

	res := <-m.rResCh
	if res.Err != nil {
		return nil, res.Err
	}

	m.Rules.Delete(ruleId)
	klog.V(2).InfoS("Deleted rule", "id", r.ID)
	return r, nil
}

func (m *Manager) getThingProperties(thingId, propertySetName string) (map[string]*model.Property, error) {
	properties := map[string]*model.Property{}
	p := path.Join("/api/model/v1/things", thingId, "propertysets")
	resp, err := m.modelClient.Get(p, nil, nil)
	if err != nil {
		klog.V(1).InfoS("Failed to access iot model manager", "err", err)
		return properties, apis.ErrInternal
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, response.ErrThingNotFound(thingId)
	} else if resp.StatusCode != http.StatusOK {
		klog.V(2).InfoS("Received HTTP response", "code", resp.StatusCode)
		return properties, apis.ErrInternal
	}

	type PropertySets struct {
		PropertySets []*model.PropertySet `json:"propertysets"`
	}
	pss := &PropertySets{}
	if err = json.NewDecoder(resp.Body).Decode(pss); err != nil {
		klog.V(3).InfoS("Failed parse response", "err", err)
		return properties, apis.ErrInternal
	}
	for _, ps := range pss.PropertySets {
		if ps.Name == propertySetName {
			for _, p := range ps.PropertySetType.Properties {
				properties[p.Name] = p
			}
		}
	}
	return properties, nil
}

func (m *Manager) getThings(thingTypeId string) ([]*model.Thing, error) {
	u := path.Join("/api/model/v1/things")
	Things := struct {
		Things []*model.Thing `json:"things"`
	}{
		Things: make([]*model.Thing, 0),
	}
	hasType := struct {
		HasType string `json:"hasType"`
	}{
		HasType: thingTypeId,
	}
	f, _ := json.Marshal(hasType)
	resp, err := m.modelClient.Get(u, nil, &url.Values{"filter": []string{string(f)}})
	if err != nil {
		klog.V(1).InfoS("Failed to access iot model manager", "err", err)
		return nil, apis.ErrInternal
	}
	defer resp.Body.Close()
	if err = json.NewDecoder(resp.Body).Decode(&Things); err != nil {
		klog.V(3).InfoS("Failed parse response", "err", err)
		return nil, apis.ErrInternal
	}
	Things.Things = append(Things.Things, Things.Things...)
	return Things.Things, nil
}

func (m *Manager) createVirtualParameterAction(obj *v1.VirtualParameterAction, thingId, propertySetName string, properties map[string]*model.Property, operands []string) (*runtime.VirtualParameter, error) {
	if obj.ThingId == thingId && obj.PropertySetName == propertySetName {
		for _, operand := range operands {
			if operand == obj.Property.Name {
				return nil, response.ErrVirtualParameterOverlapsOperand(obj.Property.Name)
			}
		}
	} else if vpps, err := m.getThingProperties(obj.ThingId, obj.PropertySetName); err != nil {
		return nil, err
	} else {
		properties = vpps
	}

	if err := validateVirtualParameterAndProperty(obj.ThingId, obj.PropertySetName, properties, &obj.Property); err != nil {
		return nil, err
	}

	return &runtime.VirtualParameter{
		BaseAction:      &runtime.BaseAction{Active: obj.Active},
		ThingId:         obj.ThingId,
		PropertySetName: obj.PropertySetName,
		Property:        *properties[obj.Property.Name],
	}, nil
}

func (m *Manager) createEventAction(ea *v1.EventAction) (*runtime.Event, error) {
	if ea == nil {
		return nil, nil
	}
	rea := &runtime.Event{
		BaseAction: &runtime.BaseAction{
			Active:   ea.Active,
			Interval: ea.Interval,
		},
		Severity:    ea.Severity,
		Description: ea.Description,
	}
	SetDefaults_BaseAction(rea.BaseAction)
	return rea, nil
}

func (m *Manager) createEmailAction(ea *v1.NotifyAction, ns *notifyServer) (*runtime.Notify, error) {
	if ea == nil {
		return nil, nil
	}
	if !ns.IsEmailConfigured() {
		return nil, response.ErrEmailServerUnconfigured
	}
	var invalidAddrs []string
	for _, ad := range ea.Addresses {
		if !validutil.IsEmail(ad) {
			invalidAddrs = append(invalidAddrs, ad)
		}
	}

	if len(invalidAddrs) != 0 {
		return nil, response.ErrEmailAddressInvalid(strings.Join(invalidAddrs, ";"))
	}

	rea := &runtime.Notify{
		BaseAction: &runtime.BaseAction{
			Active:   ea.Active,
			Interval: ea.Interval,
		},
		Description: ea.Description,
		Addresses:   ea.Addresses,
		Severity:    ea.Severity,
	}
	SetDefaults_BaseAction(rea.BaseAction)
	return rea, nil
}

func (m *Manager) createWebhookAction(wa *v1.NotifyAction) (*runtime.Notify, error) {
	if wa == nil {
		return nil, nil
	}
	var invalidAddrs []string
	for _, ad := range wa.Addresses {
		if _, err := url.Parse(ad); err != nil {
			invalidAddrs = append(invalidAddrs, ad)
		}
	}

	if len(invalidAddrs) != 0 {
		return nil, response.ErrWebhookUriInvalid(strings.Join(invalidAddrs, ";"))
	}

	rea := &runtime.Notify{
		BaseAction: &runtime.BaseAction{
			Active:   wa.Active,
			Interval: wa.Interval,
		},
		Description: wa.Description,
		Addresses:   wa.Addresses,
		Severity:    wa.Severity,
	}
	SetDefaults_BaseAction(rea.BaseAction)
	return rea, nil
}

func (m *Manager) createWeComAction(wa *v1.NotifyAction, ns *notifyServer) (*runtime.Notify, error) {
	if wa == nil {
		return nil, nil
	}
	if !ns.IsWeComConfigured() {
		return nil, response.ErrWeComServerUnconfigured
	}
	rea := &runtime.Notify{
		BaseAction: &runtime.BaseAction{
			Active:   wa.Active,
			Interval: wa.Interval,
		},
		Description: wa.Description,
		Addresses:   wa.Addresses,
		Severity:    wa.Severity,
	}
	SetDefaults_BaseAction(rea.BaseAction)
	return rea, nil
}

func (m *Manager) createWeChatAction(wa *v1.NotifyAction, ns *notifyServer) (*runtime.Notify, error) {
	if wa == nil {
		return nil, nil
	}
	if !ns.IsWeChatConfigured() {
		return nil, response.ErrWeChatServerUnconfigured
	}
	rea := &runtime.Notify{
		BaseAction: &runtime.BaseAction{
			Active:   wa.Active,
			Interval: wa.Interval,
		},
		Description: wa.Description,
		Addresses:   wa.Addresses,
		Severity:    wa.Severity,
	}
	SetDefaults_BaseAction(rea.BaseAction)
	return rea, nil
}

func validateVirtualParameterAndProperty(thingId, propertySetName string, properties map[string]*model.Property, vpp *model.Property) error {
	if target, ok := properties[vpp.Name]; ok {
		if target.DataType != vpp.DataType || target.Unit != vpp.Unit || target.Length != vpp.Length {
			return response.ErrVirtualParameterMismatchesProperty(vpp.Name, vpp.DataType, target.DataType, vpp.Unit, target.Unit, vpp.Length, target.Length)
		}
		return nil
	} else {
		return response.ErrPropertyOfThingNotFound(vpp.Name, propertySetName, thingId)
	}
}

func (m *Manager) validateRules(thingTypeId string) error {
	deactivateRules := make([]*runtime.Rule, 0)
	things, err := m.getThings(thingTypeId)
	if err != nil {
		return err
	}
	for _, t := range things {
		rules, _ := m.GetRules(&filter{ThingId: t.ID, Tenant: security.GetTenant()})
		for _, r := range rules {
			// validate
			_, operands, _ := validation.Parse(r.Evaluations[0].Expression)
			propertiesByPropertySet := map[string][]string{}
			for _, operand := range operands {
				res := strings.Split(operand, runtime.OperandUS)
				ps := res[0]
				p := res[1]
				if properties, ok := propertiesByPropertySet[ps]; ok {
					properties = append(properties, p)
				} else {
					propertiesByPropertySet[ps] = []string{p}
				}
			}
			for k, v := range propertiesByPropertySet {
				if properties, err := m.getThingProperties(t.ID, k); err != nil {
					return err
				} else {
					for _, p := range v {
						if _, ok := properties[p]; !ok {
							deactivateRules = append(deactivateRules, r)
						}
					}
				}
			}
		}
	}

	for _, r := range deactivateRules {
		m.deactivateRule(r)
	}

	return nil
}

func (m *Manager) deactivateRule(r *runtime.Rule) {
	cancelContext, cancelFunc := context.WithCancel(context.Background())
	go wait.UntilWithContext(cancelContext, func(ctx context.Context) {
		r.Active = false
		m.rCh <- r
		res := <-m.rResCh
		if res.Err != nil {
			return
		}
		rr := res.Saved.(*runtime.Rule)
		r.ModTime = rr.GetModTime()
		r.Version = rr.Version
		cancelFunc()
	}, 0)
}

func (m *Manager) getThingTypeVersion(thingTypeId string) (string, error) {
	p := path.Join("/api/model/v1/thingtypes", thingTypeId)
	resp, err := m.modelClient.Get(p, nil, nil)
	if err != nil {
		klog.V(1).InfoS("Failed to access iot model manager", "err", err)
		return "", apis.ErrInternal
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return "", os.ErrNotExist
	} else if resp.StatusCode != http.StatusOK {
		klog.V(2).InfoS("Received HTTP response", "code", resp.StatusCode)
		return "", apis.ErrInternal
	}
	ttVersion := resp.Header.Get(apis.ETag)

	return ttVersion, nil

}

func GetChannelTypeByActionType(at runtime.ActionType) notificationv1.ChannelType {
	return actionType2ChannelType[at]
}

var actionType2ChannelType = map[runtime.ActionType]notificationv1.ChannelType{
	runtime.ActionTypeEmail:   notificationv1.ChannelTypeEmail,
	runtime.ActionTypeWebhook: notificationv1.ChannelTypeWebhook,
	runtime.ActionTypeWeCom:   notificationv1.ChannelTypeWeCom,
	runtime.ActionTypeWeChat:  notificationv1.ChannelTypeWeChat,
}

func (m *Manager) processEvent(stopCh <-chan struct{}) {
	for {
		select {
		case obj, _ := <-m.rCh:
			r, _ := obj.(*runtime.Rule)
			if r.ModTime.IsZero() {
				m.Rules.Delete(r.GetID())
			} else {
				m.Rules.Store(r.ID, r)
			}
		case <-stopCh:
			klog.V(2).InfoS("Stopped rule watch event")
			return
		}
	}
}

func (m *Manager) processModelEvent(stopCh <-chan struct{}) {
	for {
		select {
		case obj, _ := <-m.tCh:
			t, _ := obj.(*model.Thing)
			if t.ModTime.IsZero() {
				rules, _ := m.GetRules(&filter{ThingId: t.ID, Tenant: security.GetTenant()})
				for _, r := range rules {
					m.deactivateRule(r)
				}
			}
		case obj, _ := <-m.ttCh:
			tt, _ := obj.(*model.ThingType)
			if !tt.ModTime.IsZero() {
				func(count int) {
					c := 0
					cancelContext, cancelFunc := context.WithCancel(context.Background())
					wait.UntilWithContext(cancelContext, func(ctx context.Context) {
						if c >= count {
							cancelFunc()
							return
						}
						targetVersion, err := m.getThingTypeVersion(tt.ID)
						if err != nil || targetVersion != tt.Version {
							c++
							return
						} else if err = m.validateRules(tt.ID); err == nil {
							cancelFunc()
						}
					}, time.Second)
				}(3)
			}
		case <-stopCh:
			klog.V(2).InfoS("Stopped thing watch event")
			return
		}
	}
}
