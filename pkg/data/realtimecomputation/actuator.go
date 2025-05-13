package realtimecomputation

import (
	"context"
	"errors"
	"k8s.io/klog/v2"
	"lightiot/pkg/client"
	"lightiot/pkg/data/storage"
	event "lightiot/pkg/event/v1"
	model "lightiot/pkg/model/runtime"
	notification "lightiot/pkg/notification/v1"
	"lightiot/pkg/promql"
	"lightiot/pkg/promql/labels"
	"lightiot/pkg/rule/rule"
	"lightiot/pkg/rule/runtime"
	"net/http"
	"strings"
	"time"
)

type Actuator struct {
	engine    *promql.Engine
	ruleMgr   *Manager
	store     *storage.Store
	eventMgr  client.Client
	notifyMgr client.Client
	stopCh    <-chan struct{}
}

func NewActuator(stopCh <-chan struct{}, engine *promql.Engine, ruleMgr *Manager, store *storage.Store, eventMgr, notifyMgr client.Client) *Actuator {
	engine = promql.NewEngine(promql.EngineOpts{
		MaxSamples:    100,
		Timeout:       5 * time.Minute, // change it to hour for debug
		LookbackDelta: 5 * time.Second, // all data in one payload have to be in 5 seconds
	})

	return &Actuator{
		engine:    engine,
		ruleMgr:   ruleMgr,
		store:     store,
		eventMgr:  eventMgr,
		notifyMgr: notifyMgr,
		stopCh:    stopCh,
	}
}

func (a *Actuator) Evaluate(key string, rawData storage.RawData) {
	if v, ok := a.ruleMgr.Rules.Load(key); ok {
		exprs := v.([]*runtime.Expression)

		for _, expr := range exprs {
			if expr.EvaluationTimestamp.Add(expr.MinInterval).After(time.Now()) {
				klog.V(5).InfoS("Ignored expression evaluate because it's not time yet", "ruleId", expr.RuleId)
				continue
			}

			go func(expr *runtime.Expression) {
				qry, err := a.engine.NewInstantQuery2(&stream{rawData, expr.Properties}, expr.Expr, getLatestTimeFromRawData(rawData))

				expr.EvaluationTimestamp = time.Now()

				if err == promql.ErrValidationAtModifierDisabled {
					err = errors.New("@ modifier is disabled, use --enable-feature=promql-at-modifier to enable it")
				} else if err == promql.ErrValidationNegativeOffsetDisabled {
					err = errors.New("negative offset is disabled, use --enable-feature=promql-negative-offset to enable it")
				}
				if err != nil {
					klog.V(3).InfoS("Failed to initialise expression", "ruleId", expr.RuleId, "err", err)
					return
				}

				res := qry.Exec(context.Background())
				if res.Err != nil {
					klog.V(3).InfoS("Failed to evaluate expression", "ruleId", expr.RuleId, "err", res.Err)
				}

				var result promql.Vector
				switch v := res.Value.(type) {
				case promql.Vector:
					result = v
				case promql.Scalar:
					result = promql.Vector{promql.Sample{
						Point:  promql.Point(v),
						Metric: labels.Labels{},
					}}
				default:
					klog.V(3).InfoS("Invalid rule result (vector or scalar)", "ruleId", expr.RuleId, "value", res.Err)
				}

				for _, smpl := range result {
					for k, v := range expr.Actions.Actions {
						// https://code.ketianyun.io/platform/lightiot/-/blob/v1.6.0/pkg/data/realtimecomputation/stream.go#L48
						ts := time.Unix(0, smpl.T*int64(time.Millisecond))
						if v.IsActive() && v.GetLastSendAt().Add(v.GetInterval()).Before(time.Now()) {
							ct := rule.GetChannelTypeByActionType(k)
							_ = v.Send(ct, strings.Split(key, "/")[0], expr.RuleName, ts, smpl.V, a)
						}
					}
				}
				qry.Close()
			}(expr)
		}
	} else {
		return
	}
}

func (a *Actuator) SaveOrUpdateVirtualParameter(thingId, psName string, property *model.Property, ts time.Time, v float64) error {
	return a.store.SaveOrUpdateVirtualParameter(thingId, psName, property, ts, v)
}

func (a *Actuator) SendEvent(e *event.StandardEvent) error {
	resp, err := a.eventMgr.Post("/api/event/v1/events", http.Header{"Content-Type": []string{"application/json"}}, nil, e)
	if err != nil {
		klog.V(1).InfoS("Failed to access iot event manager", "err", err)
		return err
	}
	defer client.Drain(resp)
	return nil
}

func (a *Actuator) SendNotification(msg *notification.Message) error {
	resp, err := a.notifyMgr.Post("/api/notification/v1/messages", http.Header{"Content-Type": []string{"application/json"}}, nil, msg)
	if err != nil {
		klog.V(1).InfoS("Failed to access iot notification manager", "err", err)
		return err
	}
	defer client.Drain(resp)
	return nil
}

func getLatestTimeFromRawData(rawData storage.RawData) time.Time {
	max := time.Time{}
	for k := range rawData {
		if k.After(max) {
			max = k
		}
	}
	return max
}
