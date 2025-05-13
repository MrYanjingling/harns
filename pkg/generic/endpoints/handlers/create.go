package handlers

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"k8s.io/klog/v2"
	"lightiot/pkg/apis"
	"lightiot/pkg/apis/response"
	"lightiot/pkg/generic/meta"
	"lightiot/pkg/generic/rest"
	"net/http"
)

func Create(rc rest.Creater) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer c.Request.Body.Close()
		obj := rc.New()
		if err := json.NewDecoder(c.Request.Body).Decode(obj); err != nil {
			klog.V(3).InfoS("Failed to decode", "err", err)
			c.JSON(http.StatusBadRequest, response.NewMultiError(response.ErrMalformedJSON))
			return
		}
		ctx := c.Request.Context()
		created, err := rc.Create(ctx, obj, rest.ValidateFunc(getValidate(rc)), &meta.CreateOptions{Query: c.Request.URL.Query()})
		if err != nil {
			switch {
			case errors.Is(err, apis.ErrMismatch):
				c.Status(http.StatusPreconditionFailed)
			default:
				if response.IsResponseError(err) {
					c.JSON(http.StatusBadRequest, response.NewMultiError(err))
				} else {
					c.Status(http.StatusInternalServerError)
				}
			}
			return
		}
		accessor, _ := meta.Accessor(created)
		if accessor != nil {
			c.Header(apis.ETag, accessor.GetVersion())
		}
		c.JSON(http.StatusCreated, created)
	}
}

func getValidate(mgr interface{}) rest.CreateStrategy {
	v, _ := mgr.(rest.CreateStrategy)
	return v
}
