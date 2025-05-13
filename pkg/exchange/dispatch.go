package exchange

import (
	"bytes"
	"encoding/json"
	"k8s.io/klog/v2"
	"lightiot/pkg/apis/response"
	"lightiot/pkg/model/runtime"
	v1 "lightiot/pkg/model/v1"
	"math"
	"path"
	"strconv"
	"time"
)

const (
	placeholder = "command"
	timeoutCode = math.MinInt32
	timeoutMsg  = "time out"
)

type dispatcher interface {
	downRoute(agentId string, m *Manager, et runtime.EventType)
	downMapping(dpm *runtime.DataPointMapping, m *Manager, et runtime.EventType)
}

type nopDispatch struct{}

func (nd *nopDispatch) downRoute(_ string, _ *Manager, _ runtime.EventType)                      {}
func (nd *nopDispatch) downMapping(_ *runtime.DataPointMapping, _ *Manager, _ runtime.EventType) {}

type dispatch struct{}

func (d *dispatch) downRoute(agentId string, m *Manager, et runtime.EventType) {
	switch et {
	case runtime.Create:
		if _, ok := m.downMappings[agentId]; !ok {
			m.downMappings[agentId] = map[string]*runtime.DataPointInfo{placeholder: nil}
		} else {
			klog.V(3).InfoS("Agent already exists", "agent", agentId)
		}
	case runtime.Remove:
		// Please make sure the all data point mappings have been deleted before
		delete(m.downMappings, agentId)
	default:
		klog.V(1).InfoS("Unsupported operation for agent", "operation", et)
	}
}

func (d *dispatch) downMapping(dpm *runtime.DataPointMapping, m *Manager, et runtime.EventType) {
	switch et {
	case runtime.Create:
		if dpm.PropertyAccessMode == v1.AccessModeReadWrite {
			if _, ok := m.downMappings[dpm.ThingId]; !ok {
				m.downMappings[dpm.ThingId] = make(map[string]*runtime.DataPointInfo)
			}
			m.downMappings[dpm.ThingId][path.Join(dpm.PropertySetName, dpm.PropertyName)] = &runtime.DataPointInfo{
				AgentId:      dpm.AgentId,
				DataSourceId: dpm.DataSourceId,
				DataPointId:  dpm.DataPointId,
			}
		}
	case runtime.Remove:
		if dpis, ok := m.downMappings[dpm.ThingId]; ok {
			delete(dpis, path.Join(dpm.PropertySetName, dpm.PropertyName))
			if len(dpis) == 0 {
				delete(m.downMappings, dpm.ThingId)
			}
		}
	default:
		klog.V(1).InfoS("Unsupported operation for dataPointMapping", "operation", et)
	}
}

func (m *Manager) dispatch(obj *v1.ControlData) (*runtime.AckData, error) {
	if len(obj.TypeId) != 0 {
		return m.exchangeCmd(obj)
	} else {
		return m.exchangeAction(obj)
	}
}

func (m *Manager) exchangeCmd(obj *v1.ControlData) (*runtime.AckData, error) {
	if dpis, ok := m.downMappings[obj.ThingId]; ok {
		if _, ok = dpis[placeholder]; ok {
			cmdData := runtime.CommandData{
				Seq:       strconv.FormatInt(obj.Seq, 10),
				Timestamp: obj.Time.UnixMilli(),
				Type:      obj.TypeId,
				Options:   obj.Options,
			}
			agentTopic := path.Join(runtime.TopicCtrlPrefix, obj.ThingId)
			var bb bytes.Buffer
			_ = json.NewEncoder(&bb).Encode(cmdData)
			token := m.mqttClient.Publish(agentTopic, 2, false, bb)
			if token.Wait() && token.Error() != nil {
				klog.V(1).InfoS("Failed to publish MQTT", "topic", agentTopic, "err", token.Error())
				return nil, response.ErrDeliverFailed
			} else {
				return m.waitCtrlResult(cmdData.Seq, obj.Ack, obj.Timeout), nil
			}
		}
	}
	return nil, response.ErrDataPointMappingNotFound
}

func (m *Manager) exchangeAction(obj *v1.ControlData) (*runtime.AckData, error) {
	if dpis, ok := m.downMappings[obj.ThingId]; ok {
		actionData := runtime.ActionData{
			Seq:       strconv.FormatInt(obj.Seq, 16),
			Timestamp: obj.Time.UnixMilli(),
		}
		valueByTopic := make(map[string][]v1.DataPointValue)
		for k, v := range obj.Options {
			if dpi, ok := dpis[path.Join(obj.PsName, k)]; ok {
				agentTopic := path.Join(runtime.TopicCtrlPrefix, dpi.AgentId, dpi.DataSourceId)
				var values []v1.DataPointValue
				if values, ok = valueByTopic[agentTopic]; !ok {
					valueByTopic[agentTopic] = make([]v1.DataPointValue, 0)
				}
				values = append(values, v1.DataPointValue{
					DataPointId: dpi.DataPointId,
					Value:       v,
				})
				valueByTopic[agentTopic] = values
			}
		}
		for k, v := range valueByTopic {
			actionData.Values = v
			var bb bytes.Buffer
			_ = json.NewEncoder(&bb).Encode(actionData)
			token := m.mqttClient.Publish(k, 2, false, bb)
			if token.Wait() && token.Error() != nil {
				klog.V(1).InfoS("Failed to publish MQTT", "topic", k, "err", token.Error())
				return nil, response.ErrDeliverFailed
			} else {
				return m.waitCtrlResult(actionData.Seq, obj.Ack, obj.Timeout), nil
			}
		}
	}
	return nil, response.ErrDataPointMappingNotFound
}

func (m *Manager) waitCtrlResult(seq string, ack bool, timeout time.Duration) *runtime.AckData {
	if !ack {
		// si, _ := strconv.ParseInt(seq, 10, 64)
		return &runtime.AckData{
			Seq:  seq,
			Code: 0,
		}
	}
	m.waitChannels[seq] = make(chan *v1.AckData)
	if timeout > m.ackTimeout {
		timeout = m.ackTimeout
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	defer delete(m.waitChannels, seq)
	for {
		select {
		case ad := <-m.waitChannels[seq]:
			// si, _ := strconv.ParseInt(ad.Seq, 10, 64)
			return &runtime.AckData{
				Seq:     seq,
				Code:    ad.Code,
				Message: ad.Message,
			}
		case <-timer.C:
			klog.V(3).InfoS("Downlink execution timeout", "seq", seq)
			s := timeoutMsg
			// si, _ := strconv.ParseInt(seq, 10, 64)
			return &runtime.AckData{
				Seq:     seq,
				Code:    timeoutCode,
				Message: &s,
			}
		case <-m.stopCh:
			klog.V(2).InfoS("Stopped wait ack due to process exited")
		}
	}
}

func (m *Manager) exchangeAck(ackData []v1.AckData) {
	for _, ad := range ackData {
		if _, ok := m.waitChannels[ad.Seq]; ok {
			m.waitChannels[ad.Seq] <- &ad
		}
	}
}
