package rule

import (
	"lightiot/pkg/generic/endpoints"
	"lightiot/pkg/generic/runtime"
	"lightiot/pkg/rule/rule"
)

type Config struct {
	RuleMgr *rule.Manager
}

type RESTProvider struct{}

func (p RESTProvider) NewREST(methods []string, config interface{}) (endpoints.APIGroupVersion, error) {
	c, _ := config.(*Config)
	mgr := map[string]interface{}{
		"rules": c.RuleMgr.RuleRest,
	}
	gv := endpoints.APIGroupVersion{
		Managers:     mgr,
		Methods:      methods,
		GroupVersion: runtime.GroupVersion{"data", "v1"},
	}
	return gv, nil
}
