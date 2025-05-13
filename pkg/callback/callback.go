package callback

import (
	"lightiot/pkg/exchange"
	"lightiot/pkg/model/runtime"
)

// AgentHandler callback function of agent CRUD
type AgentHandler func(new, old *runtime.Agent, em *exchange.Manager, et runtime.EventType)

// DataPointMappingHandler callback function of datapointmapping CRUD
type DataPointMappingHandler func(m *runtime.DataPointMapping, em *exchange.Manager, et runtime.EventType)
