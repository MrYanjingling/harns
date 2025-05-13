package endpoints

import (
	"github.com/gin-gonic/gin"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"lightiot/pkg/generic/runtime"
	"path"
	"time"
)

type APIGroupVersion struct {
	Managers          map[string]interface{}
	Root              string
	Methods           []string
	MinRequestTimeout time.Duration
	GroupVersion      runtime.GroupVersion
}

func (a *APIGroupVersion) InstallREST(router *gin.Engine) error {
	prefix := path.Join(a.Root, a.GroupVersion.Group, a.GroupVersion.Version)
	installer := &APIInstaller{
		group:             a,
		prefix:            prefix,
		minRequestTimeout: a.MinRequestTimeout,
	}

	errs := installer.Install(router.Group(prefix))

	return utilerrors.NewAggregate(errs)
}
