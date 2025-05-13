package event

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"k8s.io/klog/v2"
	"lightiot/pkg/apis"
	"lightiot/pkg/apis/response"
	"lightiot/pkg/event/event"
	"lightiot/pkg/event/runtime"
	"lightiot/pkg/generic/endpoints"
	gruntime "lightiot/pkg/generic/runtime"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	EventMgr *event.Manager
}

type RESTProvider struct{}

func (p RESTProvider) NewREST(methods []string, config interface{}) (endpoints.APIGroupVersion, error) {
	c, _ := config.(*Config)
	mgr := map[string]interface{}{
		"eventtypes": c.EventMgr.EventType,
	}
	gv := endpoints.APIGroupVersion{
		Managers:     mgr,
		Methods:      methods,
		GroupVersion: gruntime.GroupVersion{Group: "event", Version: "v1"},
	}
	return gv, nil
}

func InstallHandlers(router *gin.Engine, mgr *event.Manager) {
	v1 := router.Group("/api/event/v1")
	installEventHandlers(v1, mgr)
}

func installEventHandlers(group *gin.RouterGroup, mgr *event.Manager) {
	group.POST("/events", createEvent(mgr))
	group.GET("/events", listEvents(mgr))
	group.GET("/events/:id", getEventById(mgr))
	group.DELETE("/events/:id", deleteEvent(mgr))
}

func createEvent(m *event.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		data, err := io.ReadAll(c.Request.Body)
		if err != nil {
			klog.V(2).InfoS("Failed to read request body", "err", err)
			c.JSON(http.StatusBadRequest, err)
			return
		}

		eTag, id, err := m.CreateEvent(data)

		if err != nil {
			c.JSON(http.StatusBadRequest, err)
			return
		}

		// TODO use different scheme
		c.Header(apis.ETag, eTag)
		c.Header(apis.Location, fmt.Sprintf("http://%s%s/%s", c.Request.Host, c.Request.RequestURI, *id))
		c.Status(http.StatusAccepted)
	}
}

func listEvents(m *event.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit := 2000
		desc := false
		latest := false
		var start, end time.Time
		var selects []string
		var filter map[string]interface{}
		query := c.Request.URL.Query()
		if len(query) > 0 {
			var err error
			f := query.Get(apis.Start)
			start, err = time.Parse(time.RFC3339Nano, f)
			if err != nil {
				klog.V(3).InfoS("Failed to parse start time", "start", f)
			}
			t := query.Get(apis.End)
			end, err = time.Parse(time.RFC3339Nano, t)
			if err != nil {
				klog.V(3).InfoS("Failed to parse end time", "end", t)
			}
			s := query.Get(apis.Select)
			if len(s) > 0 {
				selects = strings.Split(s, ",")
			}
			sort := query.Get(apis.Sort)
			if len(sort) > 0 {
				if sort == "desc" {
					desc = true
				} else if sort != "asc" {
					klog.V(3).InfoS("Invalid sort keyword, the default behavior(asc) is honoured")
				}
			}
			l := query.Get(apis.Limit)
			if len(l) > 0 {
				if nl, err := strconv.Atoi(l); err != nil {
					klog.V(3).InfoS("Failed to parse limit", "err", err)
				} else {
					limit = nl
				}
			}
			sl := query.Get(apis.Latest)
			if len(sl) > 0 {
				if rl, err := strconv.ParseBool(sl); err != nil {
					klog.V(3).InfoS("Failed to parse latest", "err", err)
				} else {
					latest = rl
				}
			}
			v := query.Get(apis.Filter)
			if len(v) > 0 {
				if err := json.Unmarshal([]byte(v), &filter); err != nil {
					c.JSON(http.StatusBadRequest, response.ErrMalformedJSON)
					return
				}
			}
		}
		res, _ := m.ListEvents(start.UTC(), end.UTC(), filter, selects, desc, latest, limit)

		c.JSON(http.StatusOK, &runtime.ResponseModel{Events: res})
	}
}

func getEventById(m *event.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var thingId, typeId string
		query := c.Request.URL.Query()
		if len(query) > 0 {
			thingId = query.Get("thingId")
			typeId = query.Get("typeId")
		}
		re, eTag, err := m.GetEventById(id, thingId, typeId)

		if err != nil {
			c.JSON(http.StatusBadRequest, err)
			return
		}

		if re == nil {
			c.Status(http.StatusNotFound)
			return
		}

		c.Header(apis.ETag, eTag)
		c.JSON(http.StatusOK, re)
	}
}

func deleteEvent(mgr *event.Manager) gin.HandlerFunc {
	return func(context *gin.Context) {
		id := context.Param("id")
		eTag := context.GetHeader(apis.IfMatch)
		if len(eTag) == 0 {
			context.Status(http.StatusPreconditionRequired)
			return
		}
		var typeId string
		query := context.Request.URL.Query()
		if len(query) > 0 {
			typeId = query.Get("typeId")
		}

		err := mgr.DeleteEvent(id, eTag, typeId)
		if err != nil {
			if os.IsNotExist(err) {
				context.Status(http.StatusNotFound)
			} else if errors.Is(err, apis.ErrMismatch) {
				context.Status(http.StatusPreconditionFailed)
			} else {
				context.JSON(http.StatusBadRequest, response.NewMultiError(err))
			}
			return
		}
		context.Status(http.StatusAccepted)
		return
	}
}
