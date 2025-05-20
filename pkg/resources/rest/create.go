package rest

import (
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"k8s.io/klog/v2"
	"lightiot/pkg/apis"
	"lightiot/pkg/apis/response"
	repo "lightiot/pkg/repository"
	res "lightiot/pkg/resources"
	"net/http"
)

func Create(rc repo.Creator[res.ResourceName, repo.Record]) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer c.Request.Body.Close()
		objs := make([]repo.Object, 0)
		if err := json.NewDecoder(c.Request.Body).Decode(&objs); err != nil {
			klog.V(3).InfoS("Failed to decode", "err", err)
			c.JSON(http.StatusBadRequest, response.NewMultiError(response.ErrMalformedJSON))
			return
		}

		resource := c.Param("resource")

		records := make([]repo.Record, 0, len(objs))
		for _, obj := range objs {
			record := repo.Record(obj)
			records = append(records, record)
		}

		err := rc.Create(res.ResourceName(resource), records)
		if err != nil {
			switch {
			case errors.Is(err, apis.ErrMismatch):
				c.Status(http.StatusPreconditionFailed)
			default:
				var e *response.MultiError
				if errors.As(err, &e) {
					c.JSON(http.StatusBadRequest, err)
				} else if response.IsResponseError(err) {
					c.JSON(http.StatusBadRequest, response.NewMultiError(err))
				} else {
					c.Status(http.StatusInternalServerError)
				}
			}
			return
		}
		c.JSON(http.StatusCreated, objs)
	}
}
