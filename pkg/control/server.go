package control

import (
	"github.com/gin-gonic/gin"
	"lightiot/pkg/control/action"
	"lightiot/pkg/control/command"
	"lightiot/pkg/generic/endpoints"
	"lightiot/pkg/generic/runtime"
)

func InstallHandlers(router *gin.Engine, cm *command.Manager, am *action.Manager) {
	v1 := router.Group("/api/control/v1")
	command.InstallHandler(v1, cm)
	action.InstallHandler(v1, am)
}

type Config struct {
	CMDMgr *command.Manager
	ACTMgr *action.Manager
}

type RESTProvider struct{}

func (p RESTProvider) NewREST(methods []string, config interface{}) (endpoints.APIGroupVersion, error) {
	c, _ := config.(*Config)
	mgr := map[string]interface{}{
		"commandTypes": c.CMDMgr.CommandType,
	}
	gv := endpoints.APIGroupVersion{
		Managers:     mgr,
		Methods:      methods,
		GroupVersion: runtime.GroupVersion{Group: "control", Version: "v1"},
	}
	return gv, nil
}
