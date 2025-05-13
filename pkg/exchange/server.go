package exchange

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"k8s.io/klog/v2"
	"lightiot/pkg/apis/response"
	v1 "lightiot/pkg/model/v1"
	"net/http"
	"strconv"
)

func InstallBrokerHandlers(router *gin.Engine, m *Manager) {
	v1 := router.Group("/api/data/v1")
	{
		v1.POST("/exchange", exchange(m))
	}
}

func exchange(m *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		downlink := false
		query := c.Request.URL.Query()
		if len(query) > 0 {
			var err error
			downlink, err = strconv.ParseBool(query.Get("downlink"))
			if err != nil {
				klog.V(5).InfoS("Failed to parse downlink", "value", query.Get("downlink"))
			}
		}

		if !downlink {
			var obj v1.AgentData
			if err := json.NewDecoder(c.Request.Body).Decode(&obj); err != nil {
				klog.V(3).InfoS("Failed to parse agent uplink data", "err", err)
				c.JSON(http.StatusBadRequest, response.ErrMalformedJSON)
				return
			}
			agentId := c.GetHeader("token")
			err := m.exchange(agentId, &obj)
			if err != nil {
				c.JSON(http.StatusBadRequest, response.NewMultiError(err))
				return
			}
		} else {
			var obj v1.ControlData
			if err := json.NewDecoder(c.Request.Body).Decode(&obj); err != nil {
				klog.V(3).InfoS("Failed to parse agent downlink data", "err", err)
				c.JSON(http.StatusBadRequest, response.ErrMalformedJSON)
				return
			}
			ret, err := m.dispatch(&obj)
			if err != nil {
				c.JSON(http.StatusBadRequest, response.NewMultiError(err))
				return
			}
			c.JSON(http.StatusOK, ret)
		}
	}
}
