package runtime

import (
	"fmt"
	"k8s.io/klog/v2"
	event "lightiot/pkg/event/runtime"
	eventv1 "lightiot/pkg/event/v1"
	notification "lightiot/pkg/notification/v1"
	"lightiot/pkg/rule/computation"
	"time"
)

func (ba *BaseAction) IsActive() bool {
	return ba != nil && ba.Active
}

func (ba *BaseAction) GetInterval() time.Duration {
	if ba.Interval == nil {
		return 0
	}
	return ba.Interval.Duration
}

func (ba *BaseAction) SetLastSendAt(ts time.Time) {
	ba.lastSendAt = ts
}

func (ba *BaseAction) GetLastSendAt() time.Time {
	return ba.lastSendAt
}

func (vp *VirtualParameter) Send(_ notification.ChannelType, _, ruleName string, ts time.Time, actualValue float64, actuator computation.Actuator) error {
	klog.V(5).InfoS("Triggered virtual parameter", "rule", ruleName)
	return actuator.SaveOrUpdateVirtualParameter(vp.ThingId, vp.PropertySetName, &vp.Property, ts, actualValue)
}

func (e *Event) Send(_ notification.ChannelType, thingId, ruleName string, ts time.Time, _ float64, actuator computation.Actuator) error {
	se := &eventv1.StandardEvent{
		BaseEvent: eventv1.BaseEvent{
			Time:    ts.UTC().Format(time.RFC3339Nano),
			ThingId: thingId,
		},
		Severity:     e.Severity,
		Description:  e.Description,
		Source:       fmt.Sprintf("Rule/%s", ruleName),
		Acknowledged: false,
	}
	klog.V(5).InfoS("Triggered event", "rule", ruleName)
	return actuator.SendEvent(se)
}

func (wh *Webhook) Send(ct notification.ChannelType, thingId, ruleName string, ts time.Time, actualValue float64, actuator computation.Actuator) error {
	// TODO
	klog.V(5).InfoS("Triggered webhook", "rule", ruleName)
	return nil
}

func (n *Notify) Send(ct notification.ChannelType, thingId, ruleName string, ts time.Time, actualValue float64, actuator computation.Actuator) error {
	severity, _ := event.StandardEventSeverity[n.Severity]
	p := map[string]interface{}{
		"content":         n.Description,
		"thingId":         thingId,
		"evaluationValue": actualValue,
		"fireAt":          ts.UTC().Format(time.RFC3339Nano),
		"severity":        n.Severity,
		"ruleName":        ruleName,
	}

	msg := &notification.Message{
		Channel: ct,
		Subject: fmt.Sprintf("[%s] %s", severity, ruleName),
		Payload: p,
		To:      n.Addresses,
	}
	klog.V(5).InfoS("Triggered channel", "rule", ruleName, "channelType", notification.ChannelTypeToString[ct])
	return actuator.SendNotification(msg)
}
