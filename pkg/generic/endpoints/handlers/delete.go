package handlers

import (
	"errors"
	"github.com/gin-gonic/gin"
	"golang.org/x/mod/sumdb"
	"lightiot/pkg/apis"
	"lightiot/pkg/apis/response"
	"lightiot/pkg/generic/meta"
	"lightiot/pkg/generic/rest"
	"net/http"
	"os"
)

func Delete(d rest.Deleter) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		eTag := c.GetHeader(apis.IfMatch)
		if len(eTag) == 0 {
			c.Status(http.StatusPreconditionRequired)
			return
		}
		ctx := c.Request.Context()
		obj, err := d.Delete(ctx, id, &meta.DeleteOptions{Version: eTag, Query: c.Request.URL.Query()})
		if err != nil {
			switch {
			case os.IsNotExist(err):
				c.Status(http.StatusNotFound)
			case errors.Is(err, apis.ErrInternal):
				c.Status(http.StatusInternalServerError)
			case errors.Is(err, apis.ErrMismatch):
				c.Status(http.StatusPreconditionFailed)
			case errors.Is(err, sumdb.ErrWriteConflict):
				c.Status(http.StatusConflict)
			default:
				if response.IsResponseError(err) {
					c.JSON(http.StatusBadRequest, response.NewMultiError(err))
				} else {
					c.Status(http.StatusInternalServerError)
				}
			}
			return
		}
		c.JSON(http.StatusOK, obj)
		return
	}
}
