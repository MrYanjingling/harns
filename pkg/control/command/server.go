package command

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"io"
	"k8s.io/klog/v2"
	"lightiot/pkg/apis"
	"lightiot/pkg/apis/response"
	"lightiot/pkg/control/runtime"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func InstallHandler(group *gin.RouterGroup, cm *Manager) {
	group.POST("/commands", deliverCommand(cm))
	group.GET("/commands", listCommandHistory(cm))
}

func deliverCommand(cm *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var (
			errs          response.MultiError
			thingId       string
			commandTypeId string
		)
		query := c.Request.URL.Query()
		if len(query) > 0 {
			thingId = query.Get("thingId")
			commandTypeId = query.Get("commandTypeId")
		}
		if len(thingId) == 0 {
			errs.Add(response.ErrThingNotFound(""))
		}
		if len(commandTypeId) == 0 {
			errs.Add(response.ErrCommandTypeNotFound(""))
		}
		var obj map[string]interface{}
		if err := c.ShouldBindJSON(&obj); err != nil && err != io.EOF {
			klog.V(3).InfoS("Failed to parse command", "err", err)
			errs.Add(response.ErrMalformedJSON)
		}
		if errs.Len() != 0 {
			c.JSON(http.StatusBadRequest, &errs)
			return
		}

		err := cm.deliverCommand(thingId, commandTypeId, obj)

		if err != nil {
			c.JSON(http.StatusBadRequest, err)
			return
		}

		// TODO use different scheme
		c.Status(http.StatusAccepted)
	}
}

func listCommandHistory(cm *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var (
			thingId, commandTypeId string
			start, end             time.Time
			filter                 map[string]interface{}
		)
		limit := 2000
		desc := false
		latest := false
		query := c.Request.URL.Query()
		if len(query) > 0 {
			thingId = query.Get("thingId")
			if len(thingId) == 0 {
				c.JSON(http.StatusBadRequest, response.NewMultiError(response.ErrThingNotFound("")))
				return
			}
			commandTypeId = query.Get("commandTypeId")
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
			v := query.Get(apis.Filter)
			if len(v) > 0 {
				if len(commandTypeId) == 0 {
					klog.V(3).InfoS("commandTypeId not found, filter is ignored")
				} else if err := json.Unmarshal([]byte(v), &filter); err != nil {
					klog.V(3).InfoS("Failed to parse filter", "err", err)
					c.JSON(http.StatusBadRequest, response.NewMultiError(response.ErrMalformedJSON))
					return
				}
			}
		}
		rcs, _ := cm.listCommandHistory(thingId, commandTypeId, start.UTC(), end.UTC(), filter, desc, latest, limit)
		c.JSON(http.StatusOK, &runtime.ResponseModel{Commands: rcs})
	}
}
