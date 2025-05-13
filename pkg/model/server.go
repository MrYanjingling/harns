package model

import (
	"lightiot/pkg/generic/endpoints"
	"lightiot/pkg/generic/runtime"
	"lightiot/pkg/model/agent"
	"lightiot/pkg/model/datapointmapping"
	"lightiot/pkg/model/propertysettype"
	"lightiot/pkg/model/thing"
)

type Config struct {
	PSTMgr   *propertysettype.Manager
	ThingMgr *thing.Manager
	AgentMgr *agent.Manager
	DPMMgr   *datapointmapping.Manager
}

type RESTProvider struct{}

func (p RESTProvider) NewREST(methods []string, config interface{}) (endpoints.APIGroupVersion, error) {
	c, _ := config.(*Config)
	mgr := map[string]interface{}{
		"things":                 c.ThingMgr.Thing,
		"things/characteristics": c.ThingMgr.Characteristic,
		"things/propertysets":    c.ThingMgr.PropertySet,
		"propertysettypes":       c.PSTMgr.PST,
		"thingtypes":             c.ThingMgr.ThingType,
		"agenttypes":             c.AgentMgr.AgentType,
		"agents":                 c.AgentMgr.Agent,
		"datapointmappings":      c.DPMMgr.Mapping,
	}
	gv := endpoints.APIGroupVersion{
		Managers:     mgr,
		Methods:      methods,
		GroupVersion: runtime.GroupVersion{"model", "v1"},
	}
	return gv, nil
}
