package data

import (
	"lightiot/pkg/apis"
	"lightiot/pkg/apis/response"
	"net/http"

	"github.com/gin-gonic/gin"
)

func InstallRollupHandlers(router *gin.Engine, manager *Manager) {
	v1 := router.Group("/api/data/v1")
	{
		v1.GET("/rollup/:thingId/:propertySetName", getRollup(manager))
	}
}

func getRollup(manager *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		query := c.Request.URL.Query()
		validator := RequestValidator{
			request: &Request{
				thingId:             c.Param("thingId"),
				propertySetName:     c.Param("propertySetName"),
				start:               query.Get(apis.Start),
				end:                 query.Get(apis.End),
				interval:            query.Get(apis.Interval),
				filter:              query.Get(apis.Select),
				limit:               query.Get(apis.Limit),
				weekStartFromSunday: false,
			},
		}
		err := validator.CheckQueries()
		if err != nil {
			c.JSON(http.StatusBadRequest, response.NewMultiError(err))
			return
		}
		pst, selects, err := validator.GetPSTAndSelects(manager)
		if pst == nil {
			c.Status(http.StatusNotFound)
			return
		}
		if selects == nil {
			c.JSON(http.StatusBadRequest, response.NewMultiError(err))
			return
		}
		if len(selects) == 0 {
			c.JSON(http.StatusOK, Response{})
			return
		}

		err = validator.PrepareRequest(manager)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}

		ret := manager.GetRollup(validator.request.thingId,
			validator.request.propertySetName,
			validator.startDate,
			validator.endDate,
			selects,
			validator.interval,
			validator.limit,
			pst)

		c.JSON(http.StatusOK, Response{ret})
	}
}
