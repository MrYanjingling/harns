package realtimecomputation

import (
	"errors"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"
	"lightiot/pkg/generic"
	gruntime "lightiot/pkg/generic/runtime"
	"lightiot/pkg/rule/runtime"
	"lightiot/pkg/rule/validation"
	"lightiot/pkg/storage"
	"lightiot/pkg/util/timeutil"
	"path"
	"strings"
	"sync"
)

type Manager struct {
	Rules        *sync.Map // tingId/propertySetName -> []*runtime.Expression
	enabledRules *sync.Map // ruleID -> struct{}{}
	store        *generic.Store
	stopCh       <-chan struct{}
	rCh          chan gruntime.Object
}

type Option func(*Manager)

func WithRuleChan(rCh chan gruntime.Object) Option {
	return func(m *Manager) {
		m.rCh = rCh
		m.store, _ = generic.NewStore(gruntime.GroupResource{storage.StoreGroupToString[storage.StoreGroupData], storage.Rules}, &runtime.Rule{}, m.rCh)
	}
}

func NewManager(stopCh <-chan struct{}, opts ...Option) *Manager {
	m := &Manager{
		Rules:        &sync.Map{},
		enabledRules: &sync.Map{},
		stopCh:       stopCh,
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

func (m *Manager) Init() {
	rs, _ := m.store.LoadResource()
	for _, obj := range rs {
		_ = m.onRuleReceived(obj.(*runtime.Rule))
	}
	_, _ = m.store.Watch(m.stopCh, "", "")
	go m.processEvent(m.stopCh)
}

func (m *Manager) onRuleReceived(r *runtime.Rule) error {
	// here, it is different with rule manager
	// we only consider the active and real-time computation rule
	if !r.Active || !r.RealTime || !hasActiveAction(r) {
		klog.V(2).InfoS("Ignored rule", "id", r.ID, "name", r.Name)
		return nil
	}

	isLegal := true
	expr, operands, err := validation.Parse(r.Evaluations[0].Expression)
	if err != nil {
		klog.V(2).InfoS("Failed to parse rule", "err", err)
		return err
	}

	propertiesByPropertySet := map[string][]string{}
	for _, operand := range operands {
		res := strings.Split(operand, runtime.OperandUS)
		if len(res) != 2 {
			klog.V(3).InfoS("Invalid operand", "operand", operand)
			isLegal = false
			break
		}
		ps := res[0]
		p := res[1]
		if properties, ok := propertiesByPropertySet[ps]; ok {
			propertiesByPropertySet[ps] = append(properties, p)
		} else {
			propertiesByPropertySet[ps] = []string{p}
		}
	}

	if !isLegal {
		klog.V(2).InfoS("Ignored illegal rule", "expression", r.Evaluations[0].Expression)
		return errors.New("illegal rule")
	}

	for k, v := range propertiesByPropertySet {
		key := path.Join(r.ThingId, k)
		minInterval := timeutil.MaxDuration
		if vp, ok := r.Actions.Actions[runtime.ActionTypeVirtualParameter]; ok && vp.IsActive() {
			minInterval = 0
		} else {
			for _, a := range r.Actions.Actions {
				if a.IsActive() {
					if minInterval > a.GetInterval() {
						minInterval = a.GetInterval()
					}
				}
			}
		}

		obj, _ := m.Rules.LoadOrStore(key, []*runtime.Expression{})
		exprs := obj.([]*runtime.Expression)

		exprs = append(exprs, &runtime.Expression{
			RuleId:      r.ID,
			RuleName:    r.Name,
			Expr:        expr,
			Properties:  v,
			Actions:     r.Actions,
			MinInterval: minInterval,
		})
		m.Rules.Store(key, exprs)
	}

	m.enabledRules.Store(r.ID, struct{}{})
	return nil
}

func (m *Manager) onRuleDeleted(r *runtime.Rule) error {
	if _, ok := m.enabledRules.Load(r.ID); ok {
		keys := m.getKeysByRuleID(r.ID)
		for key := range keys {
			if obj, ok := m.Rules.Load(key); ok {
				exprs := obj.([]*runtime.Expression)
				for index := 0; index < len(exprs); index++ {
					if exprs[index].RuleId == r.ID {
						exprs[len(exprs)-1], exprs[index] = exprs[index], exprs[len(exprs)-1]
						exprs[len(exprs)-1] = nil
						exprs = exprs[:len(exprs)-1]
						index--
					}
				}
				if len(exprs) == 0 {
					m.Rules.Delete(key)
				} else {
					m.Rules.Store(key, exprs)
				}
			}
		}
		m.enabledRules.Delete(r.ID)
	}
	return nil
}

func (m *Manager) onRuleUpdated(new *runtime.Rule) error {
	if !new.Active || !new.RealTime || !hasActiveAction(new) {
		return m.onRuleDeleted(new)
	}

	isLegal := true
	expr, operands, err := validation.Parse(new.Evaluations[0].Expression)
	if err != nil {
		klog.V(2).InfoS("Failed to parse rule", "err", err)
		return m.onRuleDeleted(new)
	}
	propertiesByPropertySet := map[string][]string{}
	for _, operand := range operands {
		res := strings.Split(operand, runtime.OperandUS)
		if len(res) != 2 {
			klog.V(3).InfoS("Invalid operand", "operand", operand)
			isLegal = false
			break
		}
		ps := res[0]
		p := res[1]
		if properties, ok := propertiesByPropertySet[ps]; ok {
			propertiesByPropertySet[ps] = append(properties, p)
		} else {
			propertiesByPropertySet[ps] = []string{p}
		}
	}

	if !isLegal {
		klog.V(2).InfoS("Ignored illegal rule", "expression", new.Evaluations[0].Expression)
		return m.onRuleDeleted(new)
	}

	var newExprs []*runtime.Expression
	newKeys := sets.String{}
	for k, v := range propertiesByPropertySet {
		newKeys.Insert(path.Join(new.ThingId, k))
		minInterval := timeutil.MaxDuration
		if vp, ok := new.Actions.Actions[runtime.ActionTypeVirtualParameter]; ok && vp.IsActive() {
			minInterval = 0
		} else {
			for _, a := range new.Actions.Actions {
				if a.IsActive() {
					if minInterval > a.GetInterval() {
						minInterval = a.GetInterval()
					}
				}
			}
		}

		newExprs = append(newExprs, &runtime.Expression{
			RuleId:      new.ID,
			RuleName:    new.Name,
			Expr:        expr,
			Properties:  v,
			Actions:     new.Actions,
			MinInterval: minInterval,
		})
	}

	keys := m.getKeysByRuleID(new.ID)
	insert, update, del := generic.DifferenceAndIntersectionStrings(newKeys.UnsortedList(), keys.UnsortedList())
	for _, key := range insert {
		obj, _ := m.Rules.LoadOrStore(key, []*runtime.Expression{})
		exprs := obj.([]*runtime.Expression)
		exprs = append(exprs, newExprs...)
		m.Rules.Store(key, exprs)
	}

	for _, key := range update {
		obj, _ := m.Rules.Load(key)
		exprs := obj.([]*runtime.Expression)
		for index := 0; index < len(exprs); index++ {
			if exprs[index].RuleId == new.ID {
				exprs[len(exprs)-1], exprs[index] = exprs[index], exprs[len(exprs)-1]
				exprs[len(exprs)-1] = nil
				exprs = exprs[:len(exprs)-1]
				index--
			}
		}
		exprs = append(exprs, newExprs...)
		m.Rules.Store(key, exprs)
	}

	for _, key := range del {
		obj, _ := m.Rules.Load(key)
		exprs := obj.([]*runtime.Expression)
		for index := 0; index < len(exprs); index++ {
			if exprs[index].RuleId == new.ID {
				exprs[len(exprs)-1], exprs[index] = exprs[index], exprs[len(exprs)-1]
				exprs[len(exprs)-1] = nil
				exprs = exprs[:len(exprs)-1]
				index--
			}
		}
		if len(exprs) == 0 {
			m.Rules.Delete(key)
		} else {
			m.Rules.Store(key, exprs)
		}
	}

	return nil
}

func (m *Manager) getKeysByRuleID(ruleID string) sets.String {
	keys := sets.String{}
	m.Rules.Range(func(key, value interface{}) bool {
		exprs, _ := value.([]*runtime.Expression)
		for _, expr := range exprs {
			if expr.RuleId == ruleID {
				keys.Insert(key.(string))
				break
			}
		}
		return true
	})
	return keys
}

func (m *Manager) processEvent(stopCh <-chan struct{}) {
	for {
		select {
		case obj, _ := <-m.rCh:
			r, _ := obj.(*runtime.Rule)
			if r.ModTime.IsZero() {
				klog.V(3).InfoS("Deleted rule", "id", r.ID)
				_ = m.onRuleDeleted(r)
			} else if _, ok := m.enabledRules.Load(r.ID); ok {
				klog.V(3).InfoS("Updated rule", "id", r.ID)
				_ = m.onRuleUpdated(r)
			} else {
				klog.V(3).InfoS("Created rule", "id", r.ID)
				_ = m.onRuleReceived(r)
			}
		case <-stopCh:
			klog.V(2).InfoS("Stopped rule watch event")
			return
		}
	}
}

func hasActiveAction(r *runtime.Rule) bool {
	for _, a := range r.Actions.Actions {
		if a.IsActive() {
			return true
		}
	}
	return false
}
