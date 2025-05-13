package command

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
		td := data.NewTableDefinition(j.Key, nil, map[string]interface{}{data.Database: common.CmdRaw})
		td.AddColumn(common.ThingId, data.ColumnTypePrimaryKey, data.ColumnDatatypeString)

		end := time.Now()
		ttlSeconds := h.logStore.GetTTLSeconds(common.CmdRaw)
		start := end.Add(-time.Duration(ttlSeconds) * time.Second)
		q := query.NewRange([]*data.TableDefinition{td}, start, end, sets.NewString(common.Seq, common.ThingId), nil, 1, false, true, nil)
		commandData, err := h.logStore.List(q)
		if err != nil {
			klog.V(2).InfoS("Failed to get commands", "commandTypeId", j.Key)
			return err
		}
		commands, _ := commandData.([]map[string]interface{})
		if len(commands) == 0 {
			return nil
		}

		var things [common.DeleteThreshold]string
		for i := 0; i < len(commands) && i < common.DeleteThreshold; i++ {
			things[i] = commands[i][common.ThingId].(string)
		}

		for _, t := range things {
			if len(t) != 0 {
				statement := insert.NewDelete(td, start, end, map[string]interface{}{common.ThingId: t})
				if _, err := h.logStore.Delete(statement); err != nil {
					klog.V(2).InfoS("Failed to delete command", "thingId", t, "commandTypeId", j.Key, "err", err)
					return err
				}
			}
		}

		if len(commands) > common.DeleteThreshold {
			if err := h.jm.DeleteCommandType(j.Key, time.Now().Add(common.TimeThreshold*time.Second).UTC()); err != nil {
				klog.V(2).InfoS("Failed to write delete commandType job", "err", err)
				return err
			}
		}
	}
	return nil
}
