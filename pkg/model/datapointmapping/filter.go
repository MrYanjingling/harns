package datapointmapping

import (
	"lightiot/pkg/model/runtime"
)

type Filter struct {
	AgentId         string
	ThingId         string
	PropertySetName string
	PropertyName    string
}

type predicate func(dpm *runtime.DataPointMapping) bool

func parseFilter(filter Filter) []predicate {
	var predicates []predicate
	// agentId
	if len(filter.AgentId) > 0 {
		p := func(dpm *runtime.DataPointMapping) bool {
			if filter.AgentId == dpm.AgentId {
				return true
			}
			return false
		}
		predicates = append(predicates, p)
	}
	// thingId
	if len(filter.ThingId) > 0 {
		p := func(dpm *runtime.DataPointMapping) bool {
			if filter.ThingId == dpm.ThingId {
				return true
			}
			return false
		}
		predicates = append(predicates, p)
	}
	// PropertySetName
	if len(filter.PropertySetName) > 0 {
		p := func(dpm *runtime.DataPointMapping) bool {
			if filter.PropertySetName == dpm.PropertySetName {
				return true
			}
			return false
		}
		predicates = append(predicates, p)
	}
	// PropertyName
	if len(filter.PropertyName) > 0 {
		p := func(dpm *runtime.DataPointMapping) bool {
			if filter.PropertyName == dpm.PropertyName {
				return true
			}
			return false
		}
		predicates = append(predicates, p)
	}

	return predicates
}
