package action

import (
	"encoding/json"
	"fmt"
	"k8s.io/klog/v2"
	"lightiot/pkg/apis/response"
	"lightiot/pkg/client"
	"lightiot/pkg/control/runtime"
	"lightiot/pkg/control/storage"
	"lightiot/pkg/model/thing"
	"lightiot/pkg/model/v1"
	"lightiot/pkg/util/randutil"
	"net/http"
	"strconv"
	"time"
)

type Manager struct {
	stopCh       <-chan struct{}
	brokerClient client.Client
	modelMgr     *thing.Manager
	logStore     *storage.Store
	config       *runtime.Config
}

func NewManager(stopCh <-chan struct{},
	store *storage.Store,
	brokerClient client.Client,
	modelMgr *thing.Manager,
	config *runtime.Config) *Manager {
	return &Manager{
		stopCh:       stopCh,
		brokerClient: brokerClient,
		modelMgr:     modelMgr,
		logStore:     store,
		config:       config,
	}
}

func (m *Manager) deliverAction(thingId, psName string, obj []map[string]interface{}) *response.MultiError {
	thing, _ := m.modelMgr.GetThingById(thingId)
	if thing == nil {
		return response.NewMultiError(response.ErrThingNotFound(thingId))
	}
	thingType, _ := m.modelMgr.GetThingTypeById(thing.Type.ID, false)
	if thingType == nil {
		return response.NewMultiError(response.ErrThingTypeNotFound(thing.Type.ID))
	}
	ps, ok := thingType.PropertySetByName[psName]
	if !ok {
		return response.NewMultiError(response.ErrPropertySetNotFound(psName))
	}

	errs := &response.MultiError{}
	propertyByName := ps.PropertySetType.PropertyByName
	legalActions := make(map[string]interface{}, 0)
	for _, item := range obj {
		for k, v := range item {
			if _, exist := legalActions[k]; exist {
				errs.Add(response.ErrResourceExists(k))
				continue
			}
			p, ok := propertyByName[k]
			if !ok {
				errs.Add(response.ErrPropertyNotFound(k))
				continue
			}
			if p.AccessMode != v1.AccessModeReadWrite {
				errs.Add(response.ErrPropertyNotWritable(k))
				continue
			}

			switch p.DataType {
			case v1.DataTypeString:
				s := fmt.Sprintf("%v", v)
				rs := []rune(s)
				if len(rs) > p.Length {
					errs.Add(response.ErrStringTooLong(k, fmt.Sprintf("Max length %d", p.Length)))
				} else {
					legalActions[k] = s
				}
			case v1.DataTypeInt:
				switch v.(type) {
				case float64:
					i := int32(v.(float64))
					if i >= p.GetMinInt32() && i <= p.GetMaxInt32() {
						legalActions[k] = i
					} else {
						errs.Add(response.ErrIntegerInvalid(k, fmt.Sprintf("Valid range [%v, %v]", p.GetMinInt32(), p.GetMaxInt32())))
					}
				default:
					errs.Add(response.ErrIntegerInvalid(k))
				}
			case v1.DataTypeLong:
				switch v.(type) {
				case float64:
					i := int64(v.(float64))
					if i >= p.GetMinLong() && i <= p.GetMaxLong() {
						legalActions[k] = i
					} else {
						errs.Add(response.ErrLongInvalid(k, fmt.Sprintf("Valid range [%v, %v]", p.GetMinLong(), p.GetMaxLong())))
					}
				default:
					errs.Add(response.ErrLongInvalid(k))
				}
			case v1.DataTypeDouble:
				switch v.(type) {
				case float64:
					f := v.(float64)
					if f >= p.GetMinFloat() && f <= p.GetMaxFloat() {
						legalActions[k] = f
					} else {
						errs.Add(response.ErrDoubleInvalid(k, fmt.Sprintf("Valid range [%v, %v]", p.GetMinFloat(), p.GetMaxFloat())))
					}
				default:
					errs.Add(response.ErrDoubleInvalid(k))
				}
			case v1.DataTypeBoolean:
				switch v.(type) {
				case bool:
					legalActions[k] = v
				case string:
					b, err := strconv.ParseBool(v.(string))
					if err == nil {
						legalActions[k] = b
					} else {
						errs.Add(response.ErrBooleanInvalid(k))
					}
				default:
					errs.Add(response.ErrBooleanInvalid(k))
				}
			default:
				klog.V(3).InfoS("Unsupported dataType", "dataType", p.DataType)
			}
		}
	}

	if errs.Len() != 0 {
		return errs
	}

	if len(legalActions) == 0 {
		return response.NewMultiError(response.ErrLegalActionNotFound)
	}

	action := &runtime.Action{
		Seq:     randutil.Int63n(),
		ThingId: thingId,
		PsName:  psName,
		Time:    time.Now(),
		State:   runtime.StateDelivering,
		Actions: legalActions,
	}

	defer m.logStore.CreateAction(action, ps)

	m.dispatchAction(action)

	return nil
}

func (m *Manager) dispatchAction(action *runtime.Action) {
	header := http.Header{"Content-Type": []string{"application/json"}}
	resp, err := m.brokerClient.Post("/api/data/v1/exchange", header, map[string]interface{}{"downlink": true}, action)
	if err != nil {
		klog.V(1).InfoS("Failed to access iot data broker", "err", err)
		action.State = runtime.StateFail
		return
	}
	defer client.Drain(resp)
	if resp.StatusCode == http.StatusOK {
		var ad v1.AckData
		if err = json.NewDecoder(resp.Body).Decode(&ad); err != nil {
			klog.V(3).InfoS("Failed to parse ackData", "err", err)
			action.State = runtime.StateFail
		} else {
			action.Code = ad.Code
			action.Message = ad.Message
			if action.Code == 0 {
				action.State = runtime.StateSuccess
			} else {
				action.State = runtime.StateFail
			}
		}
	} else {
		action.State = runtime.StateFail
		var ret response.MultiError
		if err = json.NewDecoder(resp.Body).Decode(&ret); err != nil {
			klog.V(3).InfoS("Failed to parse ackData", "err", err)
		} else {
			s := ret.Error()
			action.Message = &s
		}
	}
}

func (m *Manager) ListActionHistory(thingId, psName string, start, end time.Time, desc, latest bool, limit int) (interface{}, error) {
	thing, _ := m.modelMgr.GetThingById(thingId)
	if thing == nil {
		return nil, nil
	}
	thingType, _ := m.modelMgr.GetThingTypeById(thing.Type.ID, false)
	if thingType == nil {
		return nil, nil
	}
	ps, ok := thingType.PropertySetByName[psName]
	if !ok {
		return nil, nil
	}
	return m.logStore.ListActions(thingId, ps, start, end, desc, latest, limit)
}
