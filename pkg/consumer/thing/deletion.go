package thing

import (
	"k8s.io/klog/v2"
	"lightiot/pkg/consumer/common"
	"lightiot/pkg/consumer/job"
	"lightiot/pkg/logstorage/data"
	"lightiot/pkg/logstorage/insert"
	"lightiot/pkg/logstorage/query"
	"time"
)

func (h *Handler) Delete(j *job.Job) (err error) {
	if len(j.Key) > 0 {
		if err = h.jm.DeleteThingAction(j.Key, time.Now().UTC()); err != nil {
			return err
		}
		if err = h.jm.DeleteThingCommand(j.Key, time.Now().Add(time.Second).UTC()); err != nil {
			return err
		}
		if err = h.jm.DeleteThingEvent(j.Key, time.Now().Add(2*time.Second).UTC()); err != nil {
			return err
		}
		if err = h.jm.DeleteThingTimeSeries(j.Key, time.Now().Add(3*time.Second).UTC()); err != nil {
			return err
		}
	}
	return
}

func (rh *RelationHandler) Delete(j *job.Job) error {
	switch j.Operation {
	case job.OpThingCommandDelete:
		return rh.deleteRelation(j.Key, common.CmdRaw, func(thingId string) error {
			return rh.jm.DeleteThingCommand(thingId, time.Now().Add(common.TimeThreshold*time.Second).UTC())
		})
	case job.OpThingEventDelete:
		return rh.deleteRelation(j.Key, common.EventRaw, func(thingId string) error {
			return rh.jm.DeleteThingEvent(thingId, time.Now().Add(common.TimeThreshold*time.Second).UTC())
		})
	case job.OpThingActionDelete:
		return rh.deleteRelation(j.Key, common.ActionRaw, func(thingId string) error {
			return rh.jm.DeleteThingAction(thingId, time.Now().Add(common.TimeThreshold*time.Second).UTC())
		})
	}
	return nil
}

func (rh *RelationHandler) deleteRelation(thingId, database string, f func(thingId string) error) (err error) {
	if len(thingId) > 0 {
		td := data.NewTableDefinition("", nil, map[string]interface{}{data.Database: database})
		td.AddColumn(common.ThingId, data.ColumnTypePrimaryKey, data.ColumnDatatypeString)

		end := time.Now()
		ttlSeconds := rh.logStore.GetTTLSeconds(common.CmdRaw)
		start := end.Add(-time.Duration(ttlSeconds) * time.Second)
		qts := query.NewRange([]*data.TableDefinition{td}, start, end, nil, map[string]interface{}{common.ThingId: thingId}, 1, false, false, nil)

		tableData, err := rh.logStore.ListTables(qts)
		if err != nil {
			klog.V(2).InfoS("Failed to get tables", "database", database)
			return err
		}
		tables, _ := tableData.([]string)
		if len(tables) == 0 {
			return nil
		}

		var deleteTables [common.DeleteThreshold]string
		for i := 0; i < len(tables) && i < common.DeleteThreshold; i++ {
			deleteTables[i] = tables[i]
		}

		for _, t := range deleteTables {
			if len(t) != 0 {
				td.TableName(t)
				statement := insert.NewDelete(td, start, end, map[string]interface{}{common.ThingId: thingId})
				if _, err := rh.logStore.Delete(statement); err != nil {
					klog.V(2).InfoS("Failed to delete thing relation data", "thingId", thingId, "database", database, "err", err)
					return err
				}
			}
		}

		if len(tables) > common.DeleteThreshold {
			if err := f(thingId); err != nil {
				klog.V(2).InfoS("Failed to write delete thing relation data job", "thingId", thingId, "database", database, "err", err)
				return err
			}
		}

	}
	return nil
}
