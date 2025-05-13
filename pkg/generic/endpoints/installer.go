package endpoints

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"
	"lightiot/pkg/generic/endpoints/handlers"
	"lightiot/pkg/generic/rest"
	"net/http"
	gpath "path"
	"sort"
	"strings"
	"time"
)

type APIInstaller struct {
	group             *APIGroupVersion
	prefix            string
	minRequestTimeout time.Duration
}

func (a *APIInstaller) Install(group *gin.RouterGroup) []error {
	var (
		errors []error
		i      int
	)
	paths := make([]string, len(a.group.Managers))
	for path := range a.group.Managers {
		paths[i] = path
		i++
	}
	sort.Strings(paths)
	for _, path := range paths {
		err := a.registerHandlers(group, path, a.group.Methods, a.group.Managers[path])
		if err != nil {
			errors = append(errors, fmt.Errorf("error in registering resource: %s, %v", path, err))
		}
	}
	return errors
}

func (a *APIInstaller) registerHandlers(group *gin.RouterGroup, path string, methods []string, mgr interface{}) error {
	resource, subresource, err := splitSubresource(path)
	if err != nil {
		return err
	}
	creater, isCreater := mgr.(rest.Creater)
	lister, isLister := mgr.(rest.Lister)
	getter, isGetter := mgr.(rest.Getter)
	updater, isUpdater := mgr.(rest.Updater)
	patcher, isPatcher := mgr.(rest.Patcher)
	deleter, isDeleter := mgr.(rest.Deleter)

	for _, method := range methods {
		switch method {
		case http.MethodGet:
			if isLister {
				group.GET(resource, handlers.List(lister))
			}
			if isGetter {
				if len(subresource) == 0 {
					group.GET(resource+"/:id", handlers.Get(getter))
				} else {
					group.GET(resource+"/:id/"+subresource, handlers.Get(getter))
				}
			}
		case http.MethodPost:
			if isCreater {
				group.POST(resource, handlers.Create(creater))
			}
		case http.MethodPut:
			if isUpdater {
				if len(subresource) == 0 {
					group.PUT(resource+"/:id", handlers.Update(updater))
				} else {
					group.PUT(resource+"/:id/"+subresource, handlers.Update(updater))
				}
			}
		case http.MethodPatch:
			if isPatcher {
				if len(subresource) == 0 {
					group.PATCH(resource+"/:id", handlers.Patch(patcher, sets.NewString(string(types.JSONPatchType), string(types.MergePatchType))))
				}
			}
		case http.MethodDelete:
			if isDeleter {
				group.DELETE(resource+"/:id", handlers.Delete(deleter))
			}
		default:
			klog.V(2).InfoS("Unsupported rest handler", "method", method, "path", gpath.Join(a.prefix, path))
		}
	}
	return nil
}

func splitSubresource(path string) (string, string, error) {
	var resource, subresource string
	switch parts := strings.Split(path, "/"); len(parts) {
	case 2:
		resource, subresource = parts[0], parts[1]
	case 1:
		resource = parts[0]
	default:
		// TODO: support deeper paths
		return "", "", fmt.Errorf("api_installer allows only one or two segment paths (resource or resource/subresource)")
	}
	return resource, subresource, nil
}
