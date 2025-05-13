package handlers

import (
	"github.com/gin-gonic/gin"
	"lightiot/pkg/apis"
	"lightiot/pkg/generic/meta"
	"lightiot/pkg/generic/rest"
	"net/http"
)

func Get(g rest.Getter) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		eTag := c.GetHeader(apis.IfNoneMatch)
		obj, err := g.Get(c.Request.Context(), id, &meta.GetOptions{Query: c.Request.URL.Query()})
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}

		accessor, _ := meta.Accessor(obj)
		if accessor != nil {
			if eTag == accessor.GetVersion() {
				c.Status(http.StatusNotModified)
				return
			}
			c.Header(apis.ETag, accessor.GetVersion())
		}
		c.JSON(http.StatusOK, obj)
	}
}
