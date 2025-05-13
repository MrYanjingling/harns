package action

import (
	"encoding/json"
	"lightiot/pkg/apis"
	"lightiot/pkg/apis/response"
	"lightiot/pkg/control/runtime"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"k8s.io/klog/v2"
)

func InstallHandler(group *gin.RouterGroup, am *Manager) {
	group.POST("/actions/:thingId/:psName", deliverAction(am))
	group.GET("/actions/:thingId/:psName", listActionHistory(am))
}

func deliverAction(am *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		thingId := c.Param("thingId")
		psName := c.Param("psName")

		var obj []map[string]interface{}
		if err := json.NewDecoder(c.Request.Body).Decode(&obj); err != nil {
			klog.V(3).InfoS("Failed to parse action", "err", err)
			c.JSON(http.StatusBadRequest, response.NewMultiError(response.ErrMalformedJSON))
			return
		}

		err := am.deliverAction(thingId, psName, obj)
		if err != nil {
			c.JSON(http.StatusBadRequest, err)
			return
		}

		// TODO use different scheme
		c.Status(http.StatusAccepted)
	}
}

func listActionHistory(am *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		thingId := c.Param("thingId")
		psName := c.Param("psName")
		if len(thingId) == 0 {
			c.Status(http.StatusNotFound)
			return
		}
		if len(psName) == 0 {
			c.Status(http.StatusNotFound)
			return
		}

		var (
			start, end time.Time
		)
		limit := 2000
		desc := false
		latest := false
		query := c.Request.URL.Query()
		if len(query) > 0 {
			var err error
			s := query.Get(apis.Start)
			start, err = time.Parse(time.RFC3339Nano, s)
			if err != nil {
				klog.V(3).InfoS("Failed to parse start", "err", err)
			}
			e := query.Get(apis.End)
			end, err = time.Parse(time.RFC3339Nano, e)
			if err != nil {
				klog.V(3).InfoS("Failed to parse end", "err", err)
			}
			sort := query.Get(apis.Sort)
			if len(sort) > 0 {
				sl := strings.ToLower(sort)
				if sl == "desc" {
					desc = true
				} else if sl != "asc" {
					klog.V(3).InfoS("Invalid sort keyword, the default behavior(asc) is honoured")
				}
			}
			l := query.Get(apis.Limit)
			if len(l) > 0 {
				if rl, err := strconv.Atoi(l); err != nil {
					klog.V(3).InfoS("Failed to parse limit", "err", err)
				} else {
					limit = rl
				}
			}
			la := query.Get(apis.Latest)
			if len(la) > 0 {
				if rla, err := strconv.ParseBool(la); err != nil {
					klog.V(3).InfoS("Failed to parse latest", "err", err)
				} else {
					latest = rla
				}
			}
		}
		ras, _ := am.ListActionHistory(thingId, psName, start.UTC(), end.UTC(), desc, latest, limit)
		c.JSON(http.StatusOK, &runtime.ResponseModel{Actions: ras})
	}
}
