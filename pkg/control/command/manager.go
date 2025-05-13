package command

import (
	"fmt"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"
	"lightiot/pkg/apis"
	"lightiot/pkg/apis/response"
	"lightiot/pkg/client"
	"lightiot/pkg/consumer/job"
	"lightiot/pkg/control/runtime"
	"lightiot/pkg/control/storage"
	"lightiot/pkg/control/v1"
	"lightiot/pkg/generic"
	"lightiot/pkg/generic/meta"
	gruntime "lightiot/pkg/generic/runtime"
	model "lightiot/pkg/model/runtime"
	"lightiot/pkg/model/thing"
	fstorage "lightiot/pkg/storage"
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
	cmdTypes *sync.Map // thingTypeId.id --> *runtime.CommandType

	stopCh <-chan struct{}
	// TODO consider cache
	// 1. https://github.com/dgraph-io/ristretto
	// 2. https://github.com/hashicorp/golang-lru
	// modelClient client.Client
	brokerClient client.Client
	modelMgr     *thing.Manager
	logStore     *storage.Store

	store   *generic.Store
	ctCh    chan gruntime.Object
	ctResCh chan *fstorage.PersistResult

	config *runtime.Config
	jm     *job.Manager

	CommandType *TypeREST
}

func NewManager(stopCh <-chan struct{},
	store *storage.Store,
	ctStore *generic.Store,
	ctCh chan gruntime.Object,
	brokerClient client.Client,
	modelMgr *thing.Manager,
	config *runtime.Config,
	jm *job.Manager) *Manager {
	m := &Manager{
		cmdTypes:     &sync.Map{},
		ctCh:         ctCh,
		stopCh:       stopCh,
		brokerClient: brokerClient,
		modelMgr:     modelMgr,
		logStore:     store,
		store:        ctStore,
		config:       config,
		jm:           jm,
	}

	m.CommandType = &TypeREST{m}
	return m
}

func (m *Manager) Init() {
	m.load()
	m.ctResCh = m.store.Start(m.stopCh)
}

func (m *Manager) load() {
	cts, _ := m.store.LoadResource()
	for _, obj := range cts {
		ct := obj.(*runtime.CommandType)
		m.cmdTypes.Store(fmt.Sprintf("%s.%s", ct.ThingTypeId, ct.TypeId), ct)
	}
	if len(cts) > 0 {
		m.reindexCommandType()
	}
}

func (m *Manager) createCommandType(thingTypeId string, obj *v1.CommandType) (*runtime.CommandType, error) {
	var tt *model.ThingType
	if tt, _ = m.modelMgr.GetThingTypeById(thingTypeId, false); tt == nil {
		return nil, response.ErrThingTypeNotFound(thingTypeId)
	}

	rtt := m.modelMgr.GetRootThingType(thingTypeId)
	if rtt == nil || rtt.ID != runtime.ThingTypeBaseAgent {
		pid := thingTypeId
		if rtt != nil {
			pid = tt.ID
		}
		klog.V(3).InfoS("Root thingType", "root", pid, "thingType", thingTypeId)
		return nil, response.ErrRootThingTypeNotBaseAgent
	}

	id := fmt.Sprintf("%s.%s", security.GetTenant(), obj.Name)
	if _, err := m.getCommandTypeById(thingTypeId, id, false); err == nil {
		return nil, response.ErrResourceExists(id)
	}

	var pct *runtime.CommandType
	ptt := tt.ParentTypeId
	for len(ptt) != 0 && ptt != runtime.ThingTypeBaseAgent {
		if pct, _ = m.getCommandTypeById(ptt, id, true); pct != nil {
			break
		}
		if tt, _ := m.modelMgr.GetThingTypeById(ptt, false); tt != nil {
			ptt = tt.ParentTypeId
		} else {
			return nil, response.ErrParentTypeNotFound(ptt)
		}
	}

	rct := &runtime.CommandType{
		ObjectMeta: meta.ObjectMeta{
			Tenant:  security.GetTenant(),
			Name:    obj.Name,
			ID:      uuidutil.UUID(),
			Version: strconv.FormatUint(randutil.Uint64n(), 10),
			ModTime: time.Now(),
		},
		TypeId:       id,
		Description:  obj.Description,
		Ack:          obj.Ack,
		User:         "",
		ThingTypeId:  thingTypeId,
		Options:      make([]*runtime.Option, 0),
		OptionByName: make(map[string]*runtime.Option, 0),
	}

	for _, o := range obj.Options {
		if pct != nil {
			if _, ok := pct.OptionByName[o.Name]; ok {
				return nil, response.ErrCommandOptionExists(o.Name)
			}
		}
		if _, ok := rct.OptionByName[o.Name]; ok {
			return nil, response.ErrCommandOptionExists(o.Name)
		}
		ro := &runtime.Option{
			Name:        o.Name,
			Description: o.Description,
			Required:    o.Required,
			Filterable:  o.Required,
			Datatype:    o.Datatype,
			Values:      o.Values,
			Default:     o.Default,
			Min:         o.Min,
			Max:         o.Max,
		}
		rct.Options = append(rct.Options, ro)
		rct.OptionByName[ro.Name] = ro
	}

	if pct != nil {
		for _, po := range pct.Options {
			rct.OptionByName[po.Name] = po
		}
	}

	m.ctCh <- rct
	res := <-m.ctResCh
	if res.Err != nil {
		return nil, res.Err
	}
	saved := res.Saved.(*runtime.CommandType)
	m.cmdTypes.Store(fmt.Sprintf("%s.%s", saved.ThingTypeId, saved.TypeId), saved)
	klog.V(2).InfoS("Created commandType", "id", saved.ID)

	m.derivedChainCommandType(saved)
	return saved, nil
}

func (m *Manager) listCommandTypes(filter typeFilter, exploded bool) ([]*runtime.CommandType, error) {
	rcts := make([]*runtime.CommandType, 0)
	filter.Tenant = security.GetTenant()
	predicates := parseTypeFilter(filter)

	// descend
	byModTime := func(ct1, ct2 *runtime.CommandType) bool { return ct1.ModTime.Before(ct2.ModTime) }
	sorter := ByType(byModTime)

	m.cmdTypes.Range(func(key, value interface{}) bool {
		isMatch := true
		v := value.(*runtime.CommandType)
		for _, p := range predicates {
			if !p(v) {
				isMatch = false
				break
			}
		}
		if isMatch {
			rcts = sorter.Insert(rcts, v)
		}
		return true
	})

	if exploded {
		for i, _ := range rcts {
			rcts[i] = m.explodedCmdType(rcts[i])
		}
	}

	return rcts, nil
}

func (m *Manager) getCommandTypeById(thingTypeId, id string, exploded bool) (*runtime.CommandType, error) {
	ct, isExist := m.cmdTypes.Load(fmt.Sprintf("%s.%s", thingTypeId, id))
	if !isExist {
		return nil, os.ErrNotExist
	}
	ret, _ := ct.(*runtime.CommandType)
	if exploded {
		ret = m.explodedCmdType(ret)
	}
	return ret, nil
}

func (m *Manager) updateCommandType(id string, version string, obj *v1.CommandType, new *runtime.CommandType) (gruntime.Object, error) {
	if version != new.Version {
		return nil, apis.ErrMismatch
	}
	new.ModTime = time.Now()
	new.Ack = obj.Ack
	new.Description = obj.Description

	ops := sets.NewString()
	for _, option := range new.Options {
		ops.Insert(option.Name)
	}
	for _, o := range obj.Options {
		if option, ok := new.OptionByName[o.Name]; !ok {
			ro := &runtime.Option{
				Name:        o.Name,
				Description: o.Description,
				Required:    o.Required,
				Filterable:  o.Filterable,
				Datatype:    o.Datatype,
				Values:      o.Values,
				Default:     o.Default,
				Min:         o.Min,
				Max:         o.Max,
			}
			new.Options = append(new.Options, ro)
			new.OptionByName[o.Name] = ro
		} else if ops.Has(option.Name) {
			option.Min = o.Min
			option.Max = o.Max
			option.Description = o.Description
			option.Default = o.Default
			option.Required = o.Required
		}
	}

	m.ctCh <- new
	res := <-m.ctResCh
	if res.Err != nil {
		return nil, res.Err
	}

	old, err := m.getCommandTypeById(new.ThingTypeId, id, false)
	if err != nil {
		return nil, err
	}
	old.Description = new.Description
	old.Ack = new.Ack
	old.ModTime = new.ModTime

	oops := sets.NewString()
	for _, option := range old.Options {
		oops.Insert(option.Name)
	}

	for _, o := range new.Options {
		if option, ok := old.OptionByName[o.Name]; !ok {
			old.Options = append(old.Options, o)
			old.OptionByName[o.Name] = o
		} else if oops.Has(o.Name) {
			option.Min = o.Min
			option.Max = o.Max
			option.Description = o.Description
			option.Default = o.Default
			option.Required = o.Required
		}
	}

	m.derivedChainCommandType(old)
	old.Version = new.Version
	return old, nil
}

func (m *Manager) deleteCommandType(thingTypeId, id string, eTag string) (*runtime.CommandType, error) {
	// todo should treed the thingType
	ctt, _ := m.modelMgr.GetThingTypes(&thing.ThingTypeFilter{ParentTypeId: thingTypeId}, false)
	if len(ctt) > 0 {
		for _, tt := range ctt {
			if cct, _ := m.getCommandTypeById(tt.ID, id, false); cct != nil {
				return nil, response.ErrChildTypeExist(id)
			}
		}
	}

	key := fmt.Sprintf("%s.%s", thingTypeId, id)
	ctv, isExist := m.cmdTypes.Load(key)
	if !isExist {
		return nil, os.ErrNotExist
	}

	ct := ctv.(*runtime.CommandType)
	if ct.Version != eTag {
		return nil, apis.ErrMismatch
	}

	_ = m.jm.DeleteCommandType(id, time.Now().UTC())

	m.cmdTypes.Delete(key)

	m.ctCh <- &runtime.CommandType{
		ObjectMeta: meta.ObjectMeta{
			ID:      ct.ID,
			Version: ct.Version,
		},
	}
	res := <-m.ctResCh
	if res.Err != nil {
		return nil, res.Err
	}

	klog.V(2).InfoS("Deleted commandType", "id", ct.ID)
	return ct, nil
}

func (m *Manager) DeleteCommandTypes(thingTypeId string) error {
	m.cmdTypes.Range(func(key, value interface{}) bool {
		if index := strings.Index(key.(string), thingTypeId); index == 0 {
			ct := value.(*runtime.CommandType)
			// todo maybe not need write this job.Because deleted thing has do this operation
			_ = m.jm.DeleteCommandType(ct.ID, time.Now().UTC())
			m.cmdTypes.Delete(key)
			m.ctCh <- &runtime.CommandType{
				ObjectMeta: meta.ObjectMeta{
					ID:      ct.ID,
					Version: ct.Version,
				},
			}
			res := <-m.ctResCh
			if res.Err != nil {
				klog.V(3).InfoS("Failed to delete commandType", "id", ct.ID, "err", res.Err)
			}

			klog.V(2).InfoS("Deleted commandType", "id", ct.ID)
		}
		return true
	})
	return nil
}

func (m *Manager) explodedCmdType(ct *runtime.CommandType) *runtime.CommandType {
	rct := &runtime.CommandType{
		ObjectMeta: meta.ObjectMeta{
			Tenant:  ct.Tenant,
			Name:    ct.Name,
			ID:      ct.ID,
			Version: ct.Version,
			ModTime: ct.ModTime,
		},
		TypeId:       ct.TypeId,
		Description:  ct.Description,
		User:         ct.User,
		ThingTypeId:  ct.ThingTypeId,
		Options:      make([]*runtime.Option, 0, len(ct.OptionByName)),
		OptionByName: make(map[string]*runtime.Option, len(ct.OptionByName)),
	}
	for k, option := range ct.OptionByName {
		rct.OptionByName[k] = option
		rct.Options = append(rct.Options, option)
	}

	return rct
}

func (m *Manager) reindexCommandType() {
	m.cmdTypes.Range(func(key, value interface{}) bool {
		v := value.(*runtime.CommandType)
		if len(v.Options) > 0 {
			v.OptionByName = make(map[string]*runtime.Option, len(v.Options))
			for _, o := range v.Options {
				v.OptionByName[o.Name] = o
			}
			m.inheritOptions(v)
		}
		return true
	})
}

func (m *Manager) listCommandHistory(thingId, commandTypeId string, start, end time.Time, filter map[string]interface{}, desc, latest bool, limit int) (interface{}, error) {
	thing, _ := m.modelMgr.GetThingById(thingId)
	if thing == nil {
		return nil, nil
	}
	thingType, _ := m.modelMgr.GetThingTypeById(thing.Type.ID, false)
	if thingType == nil {
		return nil, nil
	}
	var commandTypes []*runtime.CommandType
	tf := typeFilter{ThingTypeId: thing.Type.ID}
	if len(commandTypeId) != 0 {
		tf.Id = commandTypeId
	}
	commandTypes, _ = m.listCommandTypes(tf, true)
	if len(commandTypes) == 0 {
		return nil, nil
	}

	if len(commandTypeId) == 0 {
		// filter is ignored, because commandTypeId is not specified
		return m.logStore.ListCommands(thingId, commandTypes, start, end, desc, latest, limit)
	} else {
		// should only one commandType
		ct := commandTypes[0]
		legalFilter := make(map[string]interface{})
		for k, v := range filter {
			if o, ok := ct.OptionByName[k]; ok && o.Filterable {
				legalFilter[k] = v
			} else {
				klog.V(3).InfoS("Not filterable", "option", k)
			}
		}
		return m.logStore.ListCommandsWithFilter(thingId, ct, legalFilter, start, end, desc, latest, limit)
	}
}

func (m *Manager) derivedChainCommandType(ct *runtime.CommandType) {
	tt := ct.ThingTypeId
	if ctts := m.getChildThingType(tt); len(ctts) > 0 {
		for _, ctt := range ctts {
			if cct, err := m.getCommandTypeById(ctt.ID, ct.TypeId, false); !os.IsNotExist(err) {
				for _, option := range ct.Options {
					if _, ok := cct.OptionByName[option.Name]; !ok {
						cct.OptionByName[option.Name] = option
					}
				}
			}
		}
	}
}

func (m *Manager) inheritOptions(ct *runtime.CommandType) {
	options := ct.OptionByName
	tt, _ := m.modelMgr.GetThingTypeById(ct.ThingTypeId, false)
	ttId := tt.ParentTypeId
	for len(ttId) > 0 && ttId != runtime.ThingTypeBaseAgent {
		tt, err := m.modelMgr.GetThingTypeById(ttId, false)
		if err != nil {
			klog.V(4).InfoS("ThingType not found", "id", ttId)
			return
		}
		if pct, err := m.getCommandTypeById(ttId, ct.TypeId, false); !os.IsNotExist(err) {
			for _, v := range pct.Options {
				if _, ok := options[v.Name]; !ok {
					options[v.Name] = v
				}
			}
		}
		ttId = tt.ParentTypeId
	}
}

func (m *Manager) getChildThingType(thingTypeId string) []*model.ThingType {
	childs, _ := m.modelMgr.GetThingTypes(&thing.ThingTypeFilter{Tenant: security.GetTenant(), ParentTypeId: thingTypeId}, false)
	if len(childs) != 0 {
		for i := 0; i < len(childs); i++ {
			if c, _ := m.modelMgr.GetThingTypes(&thing.ThingTypeFilter{Tenant: security.GetTenant(), ParentTypeId: childs[i].ID}, false); len(c) > 0 {
				childs = append(childs, c...)
			}
		}
	}
	return childs
}
