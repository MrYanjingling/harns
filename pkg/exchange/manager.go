package exchange

import (
	"encoding/json"
	"github.com/eclipse/paho.mqtt.golang"
	"github.com/mitchellh/mapstructure"
	"k8s.io/klog/v2"
	"lightiot/pkg/apis/response"
	"lightiot/pkg/client"
	"lightiot/pkg/data/storage"
	event "lightiot/pkg/event/runtime"
	"lightiot/pkg/generic"
	"lightiot/pkg/model/runtime"
	v1 "lightiot/pkg/model/v1"
	"net/http"
	"path"
	"reflect"
	"strings"
	"time"
)

type Manager struct {
	mqttClient      mqtt.Client
	msgQueues       map[string]chan mqtt.Message
	queueLength     int
	collectorClient client.Client
	eventMgrClient  client.Client
	stopCh          <-chan struct{}
	upMappings      map[string]map[string][]*runtime.PropertyInfo // (agentId/dataSourceId) -> datapointId -> []*runtime.PropertyInfo

	// downlink related
	downMappings map[string]map[string]*runtime.DataPointInfo // thingId -> (propertySetName/propertyName) -> *runtime.DataPointInfo
	dispatcher   dispatcher
	waitChannels map[string]chan *v1.AckData
	ackTimeout   time.Duration
}

func NewManager(stopCh <-chan struct{},
	mqttClient mqtt.Client,
	queueLength int,
	collectorClient, eventMgrClient client.Client,
	biDirection bool,
	ackTimeout time.Duration) *Manager {
	em := &Manager{
		mqttClient:      mqttClient,
		msgQueues:       make(map[string]chan mqtt.Message),
		queueLength:     queueLength,
		collectorClient: collectorClient,
		eventMgrClient:  eventMgrClient,
		stopCh:          stopCh,
		upMappings:      make(map[string]map[string][]*runtime.PropertyInfo),
		downMappings:    make(map[string]map[string]*runtime.DataPointInfo),
		waitChannels:    make(map[string]chan *v1.AckData),
		ackTimeout:      ackTimeout,
	}

	if biDirection {
		em.dispatcher = new(dispatch)
	} else {
		em.dispatcher = new(nopDispatch)
	}

	return em
}

func OnAgentReceived(new, old *runtime.Agent, m *Manager, et runtime.EventType) {
	agentTopic := path.Join(runtime.TopicDataPrefix, new.ID)
	switch et {
	case runtime.Create:
		subscribeMqttTopic(agentTopic, m)
		for _, ds := range new.DataSources {
			datasourceTopic := path.Join(agentTopic, *ds.Id)
			subscribeMqttTopic(datasourceTopic, m)
		}
	case runtime.Update:
		delDSs, _, inDSs := generic.DifferenceAndIntersectionSameTypeObjects(old.DataSources, new.DataSources,
			func(value interface{}) string { return *value.(*runtime.DataSource).Id })
		for _, ds := range delDSs {
			datasourceTopic := path.Join(agentTopic, ds)
			unSubscribeMqttTopic(datasourceTopic, m)
		}
		for _, ds := range inDSs {
			datasourceTopic := path.Join(agentTopic, ds)
			subscribeMqttTopic(datasourceTopic, m)
		}
	case runtime.Remove:
		unSubscribeMqttTopic(agentTopic, m)
		for _, ds := range new.DataSources {
			datasourceTopic := path.Join(agentTopic, *ds.Id)
			unSubscribeMqttTopic(datasourceTopic, m)
		}
	default:
		klog.V(1).InfoS("Unsupported operation for agent", "operation", et)
	}
	m.dispatcher.downRoute(new.ID, m, et)
}

func OnDataPointMappingReceived(dpm *runtime.DataPointMapping, m *Manager, et runtime.EventType) {
	ad := path.Join(dpm.AgentId, dpm.DataSourceId)
	switch et {
	case runtime.Create:
		if _, ok := m.upMappings[ad]; ok {
			pi := &runtime.PropertyInfo{
				PropertySetInfo: runtime.PropertySetInfo{
					ThingId:         dpm.ThingId,
					PropertySetName: dpm.PropertySetName,
				},
				PropertyName: dpm.PropertyName,
			}
			if _, ok := m.upMappings[ad][dpm.DataPointId]; ok {
				m.upMappings[ad][dpm.DataPointId] = append(m.upMappings[ad][dpm.DataPointId], pi)
			} else {
				pis := make([]*runtime.PropertyInfo, 0)
				pis = append(pis, pi)
				m.upMappings[ad][dpm.DataPointId] = pis
			}
		} else {
			m.upMappings[ad] = make(map[string][]*runtime.PropertyInfo)
			pis := make([]*runtime.PropertyInfo, 0)
			pis = append(pis, &runtime.PropertyInfo{
				PropertySetInfo: runtime.PropertySetInfo{
					ThingId:         dpm.ThingId,
					PropertySetName: dpm.PropertySetName,
				},
				PropertyName: dpm.PropertyName,
			})
			m.upMappings[ad][dpm.DataPointId] = pis
		}
	case runtime.Remove:
		if _, ok := m.upMappings[ad]; ok {
			if pis, ok := m.upMappings[ad][dpm.DataPointId]; ok {
				var index int
				exist := false
				for i, pi := range pis {
					if pi.ThingId == dpm.ThingId &&
						pi.PropertySetName == dpm.PropertySetName &&
						pi.PropertyName == dpm.PropertyName {
						exist = true
						index = i
						break
					}
				}

				if exist {
					// order doesn't matter
					pis[len(pis)-1], pis[index] = pis[index], pis[len(pis)-1]
					pis = pis[:len(pis)-1]
				}

				if len(pis) == 0 {
					delete(m.upMappings[ad], dpm.DataPointId)
					if len(m.upMappings[ad]) == 0 {
						delete(m.upMappings, ad)
					}
				} else {
					m.upMappings[ad][dpm.DataPointId] = pis
				}
			}
		}
	default:
		klog.V(1).InfoS("Unsupported operation for dataPointMapping", "operation", et)
	}

	m.dispatcher.downMapping(dpm, m, et)
}

func (m *Manager) Close() {
	// TODO need unsubscribe topics?
	m.mqttClient.Disconnect(uint((60 * time.Second).Milliseconds()))
}

func (m *Manager) processMqttMessage(stopCh <-chan struct{}, queue chan mqtt.Message) {
	for {
		select {
		case msg, ok := <-queue:
			if !ok {
				klog.V(3).InfoS("MQTT message channel closed")
				return
			}
			var agentId, dataSourceId string
			dir, file := path.Split(strings.TrimPrefix(msg.Topic(), runtime.TopicDataPrefix))
			if len(dir) > 0 {
				dataSourceId = file
				agentId = path.Dir(dir)
			} else {
				agentId = file
			}
			data := m.convertMqttMessage(msg, dataSourceId)
			if data != nil {
				_ = m.exchange(agentId, data)
			}
		case <-stopCh:
			klog.V(2).InfoS("Stopped MQTT message process")
			return
		}
	}
}

func (m *Manager) convertMqttMessage(msg mqtt.Message, dataSourceId string) *v1.AgentData {
	var obj v1.AgentData
	if err := json.Unmarshal(msg.Payload(), &obj); err != nil {
		klog.V(3).InfoS("Failed to parse MQTT message", "err", err)
		return nil
	}
	obj.Payload.DataSourceId = &dataSourceId
	return &obj
}

func (m *Manager) exchange(agentId string, in *v1.AgentData) error {
	switch in.Payload.Type {
	case v1.AgentDataTypeTimeSeries:
		ds, ok := m.upMappings[path.Join(agentId, *in.Payload.DataSourceId)]
		if !ok {
			klog.V(3).InfoS("DataPointMapping not found", "agent", agentId, "dataSource", *in.Payload.DataSourceId)
			return response.ErrDataPointMappingNotFound
		}
		var tsData []v1.TimeSeriesData
		if err := decode(in.Payload.Data, &tsData); err != nil {
			klog.V(3).InfoS("Failed to parse time series data", "err", err)
			return response.ErrMalformedJSON
		}
		m.exchangeTimeSeries(agentId, *in.Payload.DataSourceId, ds, tsData)
	case v1.AgentDataTypeEvent:
		var eventData []v1.EventData
		if err := decode(in.Payload.Data, &eventData); err != nil {
			klog.V(3).InfoS("Failed to parse event data", "err", err)
			return response.ErrMalformedJSON
		}
		m.exchangeEvent(agentId, eventData)
	case v1.AgentDataTypeCtrlAck:
		var ackData []v1.AckData
		if err := decode(in.Payload.Data, &ackData); err != nil {
			klog.V(3).InfoS("Failed to parse ack data", "err", err)
			return response.ErrMalformedJSON
		}
		m.exchangeAck(ackData)
	default:
		klog.V(1).InfoS("Unsupported agent data type from south end", "datatype", in.Payload.Type)
	}
	return nil
}

func (m *Manager) exchangeTimeSeries(agentId, dsId string, ds map[string][]*runtime.PropertyInfo, tsData []v1.TimeSeriesData) {
	out := make(map[runtime.PropertySetInfo]storage.RawData)
	for _, inItem := range tsData {
		timestamp := inItem.Timestamp
		for _, v := range inItem.Values {
			if propertyInfos, ok := ds[v.DataPointId]; ok {
				for _, propertyInfo := range propertyInfos {
					if _, ok := out[propertyInfo.PropertySetInfo]; !ok {
						out[propertyInfo.PropertySetInfo] = make(storage.RawData)
					}
					rawData := out[propertyInfo.PropertySetInfo]
					if _, ok := rawData[timestamp]; !ok {
						rawData[timestamp] = map[string]interface{}{
							storage.TimeKey: timestamp,
						}
					}
					outItem := rawData[timestamp]
					outItem[propertyInfo.PropertyName] = v.Value
				}
			} else {
				klog.V(3).InfoS("Unmapped data point", "dataPoint", v.DataPointId, "dataSource", dsId, "agent", agentId)
			}
		}
	}

	for propertySetInfo, item := range out {
		p := path.Join("/api/data/v1/timeseries/", propertySetInfo.ThingId, propertySetInfo.PropertySetName)

		data := make([]map[string]interface{}, 0)
		for _, v := range item {
			data = append(data, v)
		}
		func() {
			resp, err := m.collectorClient.Put(p, http.Header{"Content-Type": []string{"application/json"}}, data)
			if err != nil {
				klog.V(1).InfoS("Failed to access iot collector", "err", err)
				return
			}
			defer client.Drain(resp)
		}()
	}
}

func (m *Manager) exchangeEvent(agentId string, eventData []v1.EventData) {
	var out []map[string]interface{}
	for _, inItem := range eventData {
		e := map[string]interface{}{
			event.BaseEventFieldId:            inItem.Id,
			event.BaseEventFieldTypeId:        inItem.Type,
			event.BaseEventFieldCorrelationId: inItem.CorrelationId,
			event.BaseEventFieldTime:          inItem.Timestamp,
			event.BaseEventFieldThingId:       agentId,
		}
		for k, v := range inItem.Fields {
			e[k] = v
		}
		out = append(out, e)
	}

	p := "/api/event/v1/events"
	for _, item := range out {
		func() {
			resp, err := m.eventMgrClient.Post(p, http.Header{"Content-Type": []string{"application/json"}}, nil, item)
			if err != nil {
				klog.V(1).InfoS("Failed to access iot event manager", "err", err)
				return
			}
			defer client.Drain(resp)
		}()
	}
}

func subscribeMqttTopic(topic string, m *Manager) {
	// https://stackoverflow.com/questions/33480730/understanding-mqtt-subscriber-qos
	if token := m.mqttClient.Subscribe(topic, 1, func(c mqtt.Client, msg mqtt.Message) {
		if queue, ok := m.msgQueues[msg.Topic()]; ok {
			queue <- msg
		} else {
			klog.InfoS("Topic not found", "topic", msg.Topic())
		}
	}); token.Wait() && token.Error() != nil {
		klog.V(1).InfoS("Failed to subscribe MQTT", "topic", topic, "err", token.Error())
	}
	// fixed race condition
	q := make(chan mqtt.Message, m.queueLength)
	m.msgQueues[topic] = q
	go m.processMqttMessage(m.stopCh, q)
}

func unSubscribeMqttTopic(topic string, m *Manager) {
	if token := m.mqttClient.Unsubscribe(topic); token.Wait() && token.Error() != nil {
		klog.V(1).InfoS("Failed to unsubscribe MQTT", "topic", topic, "err", token.Error())
	}
	if mCh, ok := m.msgQueues[topic]; ok {
		close(mCh)
		delete(m.msgQueues, topic)
	}
}

// https://github.com/mitchellh/mapstructure/issues/159

func toTimeHookFunc() mapstructure.DecodeHookFunc {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{}) (interface{}, error) {
		if t != reflect.TypeOf(time.Time{}) {
			return data, nil
		}

		switch f.Kind() {
		case reflect.String:
			return time.Parse(time.RFC3339Nano, data.(string))
		case reflect.Float64:
			return time.Unix(0, int64(data.(float64))*int64(time.Millisecond)), nil
		case reflect.Int64:
			return time.Unix(0, data.(int64)*int64(time.Millisecond)), nil
		default:
			return data, nil
		}
		// Convert it by parsing
	}
}

func decode(input interface{}, result interface{}) error {
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Metadata: nil,
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			toTimeHookFunc()),
		Result: result,
	})
	if err != nil {
		return err
	}

	if err := decoder.Decode(input); err != nil {
		return err
	}
	return err
}
