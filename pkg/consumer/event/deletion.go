package event

import (
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"
	"lightiot/pkg/consumer/common"
	"lightiot/pkg/consumer/job"
	"lightiot/pkg/logstorage/data"
	"lightiot/pkg/logstorage/insert"
	"lightiot/pkg/logstorage/query"
	"lightiot/pkg/util/strutil"
	"strconv"
	"time"
)

func (h *Handler) Delete(j *job.Job) error {
	if len(j.Key) > 0 {
		ttlStr, eventTypeId := strutil.Split(j.Key)
		ttl, err := strconv.Atoi(ttlStr)
		if err != nil {
			klog.V(2).InfoS("Failed to convert ttl string to int", "ttl", ttlStr)
			return err
		}

		td := data.NewTableDefinition(eventTypeId, nil, map[string]interface{}{data.Database: common.EventRaw})
		td.AddColumn(common.ThingId, data.ColumnTypePrimaryKey, data.ColumnDatatypeString)
		ttlDay := h.logStore.GetTTLDays(common.EventRaw)

		shift := time.Duration(ttl-ttlDay) * 24 * time.Hour
		end := time.Now().Add(shift)
		start := end.Add(-time.Duration(ttl) * 24 * time.Hour)

		q := query.NewRange([]*data.TableDefinition{td}, start, end, sets.NewString(common.Id, common.ThingId), nil, 1, false, true, nil)
		eventData, err := h.logStore.List(q)
		if err != nil {
			klog.V(2).InfoS("Failed to get events", "eventTypeId", eventTypeId)
			return err
		}
		events, _ := eventData.([]map[string]interface{})
		if len(events) == 0 {
			return nil
		}

		var things [common.DeleteThreshold]string
		for i := 0; i < len(events) && i < common.DeleteThreshold; i++ {
			things[i] = events[i][common.ThingId].(string)
		}

		for _, t := range things {
			if len(t) != 0 {
				statement := insert.NewDelete(td, start, end, map[string]interface{}{common.ThingId: t})
				if _, err := h.logStore.Delete(statement); err != nil {
					klog.V(2).InfoS("Failed to delete event", "thingId", t, "eventTypeId", eventTypeId, "err", err)
					return err
				}
			}
		}

		if len(events) > common.DeleteThreshold {
			if err := h.jm.DeleteEventType(j.Key, time.Now().Add(common.TimeThreshold*time.Second).UTC()); err != nil {
				klog.V(2).InfoS("Failed to write delete eventType job", "err", err)
				return err
			}
		}
	}
	return nil
}
