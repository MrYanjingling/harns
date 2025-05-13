package handlers

import (
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"k8s.io/klog/v2"
	"lightiot/pkg/apis"
	"lightiot/pkg/apis/response"
	"lightiot/pkg/generic/meta"
	"lightiot/pkg/generic/rest"
	"net/http"
	"os"
)

func Update(u rest.Updater) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer c.Request.Body.Close()
		eTag := c.GetHeader(apis.IfMatch)
		if len(eTag) == 0 {
			c.Status(http.StatusPreconditionRequired)
			return
		}
		obj := u.New()
		if err := json.NewDecoder(c.Request.Body).Decode(obj); err != nil {
			klog.V(3).InfoS("Failed to decode", "err", err)
			c.JSON(http.StatusBadRequest, response.NewMultiError(response.ErrMalformedJSON))
			return
		}
		id := c.Param("id")
		ctx := c.Request.Context()
		updated, err := u.Update(ctx, id, obj, rest.ValidateUpdateFunc(getValidateUpdate(u)), &meta.UpdateOptions{Version: eTag, Query: c.Request.URL.Query()})
		if err != nil {
			switch {
			case os.IsNotExist(err):
				c.Status(http.StatusNotFound)
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

		accessor, _ := meta.Accessor(updated)
		if accessor != nil {
			c.Header(apis.ETag, accessor.GetVersion())
		}
		c.JSON(http.StatusOK, updated)
	}
}

func getValidateUpdate(mgr interface{}) rest.UpdateStrategy {
	v, _ := mgr.(rest.UpdateStrategy)
	return v
}
