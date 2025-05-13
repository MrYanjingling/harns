package handlers

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"k8s.io/klog/v2"
	"lightiot/pkg/apis"
	"lightiot/pkg/generic/meta"
	"lightiot/pkg/generic/rest"
	"net/http"
)

func List(l rest.Lister) gin.HandlerFunc {
	return func(c *gin.Context) {
		opts := meta.ListOptions{}
		if f, ok := c.GetQuery(apis.Filter); ok {
			if err := json.Unmarshal([]byte(f), &opts.Filter); err != nil {
				klog.V(3).InfoS("Invalid filter", "err", err)
			}
		}
		opts.Query = c.Request.URL.Query()
		obj, _ := l.List(c.Request.Context(), &opts)
		c.JSON(http.StatusOK, obj)
	}
}
