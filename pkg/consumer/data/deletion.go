package data

import (
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"
	"lightiot/pkg/consumer/common"
	"lightiot/pkg/consumer/job"
	"lightiot/pkg/logstorage/data"
	"lightiot/pkg/logstorage/insert"
	"lightiot/pkg/logstorage/query"
	"time"
)

func (h *Handler) Delete(j *job.Job) error {
	if len(j.Key) > 0 {
		td := data.NewTableDefinition("", nil, map[string]interface{}{data.Database: common.DataRaw})
		td.AddColumn("ti", data.ColumnTypePrimaryKey, data.ColumnDatatypeString)

		end := time.Now()
		ttlSeconds := h.logStore.GetTTLSeconds(common.DataRaw)
		start := end.Add(-time.Duration(ttlSeconds) * time.Second)
		qts := query.NewRange([]*data.TableDefinition{td}, start, end, nil, map[string]interface{}{"ti": j.Key}, 1, false, false, nil)

		propertySets, err := h.logStore.ListTables(qts)
		if err != nil {
			klog.V(2).InfoS("Failed to get thing propertySets", "thingId", j.Key)
			return err
		}
		pss, _ := propertySets.([]string)
		if len(pss) == 0 {
			return nil
		}
		ps := pss[0]

		td.TableName(ps)
		q := query.NewSingle([]*data.TableDefinition{td}, sets.NewString(), map[string]interface{}{"ti": j.Key}, true)
		ttsData, err := h.logStore.Get(q)
		if err != nil {
			klog.V(2).InfoS("Failed to get thing time series", "thingId", j.Key)
			return err
		}

		tts, _ := ttsData.([]map[string]interface{})
		if len(tts) != 0 {
			t := tts[0]["_time"].(time.Time)
			statement := insert.NewDelete(td, t.Add(-24*time.Hour), t, map[string]interface{}{"ti": j.Key})
			if _, err := h.logStore.Delete(statement); err != nil {
				klog.V(2).InfoS("Failed to delete thing timeSeries data", "thingId", j.Key, "err", err)
				return err
			}
		}

		if err := h.jm.DeleteThingTimeSeries(j.Key, time.Now().Add(common.TimeThreshold*time.Second).UTC()); err != nil {
			klog.V(2).InfoS("Failed to write delete thing relation data job", "thingId", j.Key, "err", err)
			return err
		}
	}
	return nil
}
