package data

import (
	"encoding/json"
	"lightiot/pkg/apis"
	"lightiot/pkg/apis/response"
	"lightiot/pkg/data/storage"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"k8s.io/klog/v2"
)

func InstallCollectHandlers(router *gin.Engine, store *storage.Store) {
	v1 := router.Group("/api/data/v1")
	{
		v1.PUT("/timeseries/:thingId/:propertySetName", saveOrUpdateTimeSeries(store))
		v1.DELETE("/timeseries/:thingId/:propertySetName", deleteTimeSeries(store))
	}
}

func InstallQueryHandlers(router *gin.Engine, store *storage.Store) {
	v1 := router.Group("/api/data/v1")
	{
		v1.GET("/timeseries/:thingId/:propertySetName", getTimeSeries(store))
	}
}

func saveOrUpdateTimeSeries(store *storage.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var obj []map[string]interface{}
		if err := json.NewDecoder(c.Request.Body).Decode(&obj); err != nil {
			klog.V(3).InfoS("Failed to parse time series", "err", err)
			c.JSON(http.StatusBadRequest, response.ErrMalformedJSON)
			return
		}
		sync, _ := strconv.ParseBool(c.GetHeader("sync"))
		thingId := c.Param("thingId")
		psName := c.Param("propertySetName")
		err := store.SaveOrUpdateTimeSeries(thingId, psName, obj, sync)
		if err != nil {
			c.JSON(http.StatusBadRequest, response.NewMultiError(err))
			return
		}
		c.Status(http.StatusAccepted)
	}
}

func getTimeSeries(store *storage.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		thingId := c.Param("thingId")
		if len(thingId) == 0 {
			c.Status(http.StatusNotFound)
			return
		}
		psName := c.Param("propertySetName")
		if len(psName) == 0 {
			c.Status(http.StatusNotFound)
			return
		}

		var (
			start, end time.Time
			selects    []string
		)
		limit := 2000
		desc := false
		latest := false

		query := c.Request.URL.Query()
		if len(query) > 0 {
			var err error
			la := query.Get(apis.Latest)
			if len(la) > 0 {
				if rla, err := strconv.ParseBool(la); err != nil {
					klog.V(3).InfoS("Failed to parse latest", "err", err)
				} else {
					latest = rla
				}
			}
			sel := query.Get(apis.Select)
			if len(sel) > 0 {
				selects = strings.Split(sel, ",")
			}

			// if query latest, the following query parameters are ignored
			if !latest {
				s := query.Get(apis.Start)
				start, err = time.Parse(time.RFC3339Nano, s)
				if err != nil {
					klog.V(3).InfoS("Failed to parse start", "err", err)
					c.JSON(http.StatusBadRequest, response.ErrTimeInvalid(s))
					return
				}
				e := query.Get(apis.End)
				end, err = time.Parse(time.RFC3339Nano, e)
				if err != nil {
					klog.V(3).InfoS("Failed to parse end", "err", err)
					c.JSON(http.StatusBadRequest, response.ErrTimeInvalid(e))
					return
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
			}
		}

		data, err := store.GetTimeSeries(thingId, psName, start.UTC(), end.UTC(), selects, desc, latest, limit)

		if err != nil {
			c.JSON(http.StatusBadRequest, response.NewMultiError(err))
			return
		}

		c.JSON(http.StatusOK, data)
	}
}

func deleteTimeSeries(store *storage.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		thingId := c.Param("thingId")
		psName := c.Param("propertySetName")
		query := c.Request.URL.Query()
		var (
			s, e time.Time
			err  error
		)
		f := query.Get(apis.Start)
		s, err = time.Parse(time.RFC3339Nano, f)
		if err != nil {
			klog.V(3).InfoS("Failed to parse start", "err", err)
			c.JSON(http.StatusBadRequest, response.ErrTimeInvalid(f))
			return
		}
		t := query.Get(apis.End)
		e, err = time.Parse(time.RFC3339Nano, t)
		if err != nil {
			klog.V(3).InfoS("Failed to parse end", "err", err)
			c.JSON(http.StatusBadRequest, response.ErrTimeInvalid(t))
			return
		}

		err = store.DeleteTimeSeries(thingId, psName, s.UTC(), e.UTC())

		if err != nil {
			c.JSON(http.StatusBadRequest, response.NewMultiError(err))
			return
		}
		c.Status(http.StatusAccepted)
		return
	}
}
