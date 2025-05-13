package query

import (
	"k8s.io/apimachinery/pkg/util/sets"
	"lightiot/pkg/logstorage/data"
	"time"
)

// Single used for latest query or query by ID
type Single struct {
	TableDefinitions
	end     *time.Time
	selects sets.String
	keys    map[string]interface{}
	latest  bool
}

func NewSingle(tds []*data.TableDefinition, selects sets.String, keys map[string]interface{}, latest bool) *Single {
	return &Single{
		TableDefinitions: tds,
		selects:          selects,
		keys:             keys,
		latest:           latest,
	}
}

func NewSingleWithEndTime(tds []*data.TableDefinition, end time.Time, selects sets.String, keys map[string]interface{}, latest bool) *Single {
	return &Single{
		TableDefinitions: tds,
		end:              &end,
		selects:          selects,
		keys:             keys,
		latest:           latest,
	}
}

func (s *Single) GetEnd() time.Time {
	if s.end == nil {
		return time.Now()
	}
	return *s.end
}

func (s *Single) GetKeys() map[string]interface{} {
	return s.keys
}

func (s *Single) GetSelects() sets.String {
	return s.selects
}

func (s *Single) IsLatest() bool {
	return s.latest
}
