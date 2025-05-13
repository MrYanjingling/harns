package command

import (
	"encoding/json"
	"fmt"
	"k8s.io/klog/v2"
	"lightiot/pkg/apis/response"
	"lightiot/pkg/client"
	"lightiot/pkg/control/runtime"
	"lightiot/pkg/control/v1"
	model "lightiot/pkg/model/runtime"
	"lightiot/pkg/util/randutil"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// command

func (m *Manager) deliverCommand(thingId, commandTypeId string, obj map[string]interface{}) *response.MultiError {
	thing, _ := m.modelMgr.GetThingById(thingId)
	if thing == nil {
		return response.NewMultiError(response.ErrThingNotFound(thingId))
	}
	thingType, _ := m.modelMgr.GetThingTypeById(thing.Type.ID, false)
	if thingType == nil {
		return response.NewMultiError(response.ErrThingTypeNotFound(thing.Type.ID))
	}
	commandType, _ := m.getCommandTypeById(thingType.ID, commandTypeId, true)
	if commandType == nil {
		return response.NewMultiError(response.ErrCommandTypeNotFound(commandTypeId))
	}

	errs := &response.MultiError{}
	var invalidOptions []string
	for k := range obj {
		if _, ok := commandType.OptionByName[k]; !ok {
			invalidOptions = append(invalidOptions, k)
		}
	}
	if len(invalidOptions) != 0 {
		errs.Add(response.ErrCommandOptionNotFound(invalidOptions))
	}

	invalidOptions = nil
	requiredOptions := map[string]*runtime.Option{}
	for _, o := range commandType.Options {
		if o.Default == nil {
			if v, ok := obj[o.Name]; ok && v == nil {
				invalidOptions = append(invalidOptions, o.Name)
			}
		}
		if o.Required {
			requiredOptions[o.Name] = o
		}
	}
	if len(invalidOptions) != 0 {
		errs.Add(response.ErrCommandOptionValueMissed(invalidOptions))
	}
	if errs.Len() != 0 {
		return errs
	}

	legalOptions := map[string]interface{}{}
	for k, v := range obj {
		o, _ := commandType.OptionByName[k]
		if v == nil {
			legalOptions[k] = v
			continue
		}
		switch o.Datatype {
		case v1.DatatypeString:
			legalOptions[k] = fmt.Sprintf("%v", v)
		case v1.DatatypeInt:
			switch v.(type) {
			case float64:
				i := int64(v.(float64))
				if i >= o.GetMinInt() && i <= o.GetMaxInt() {
					legalOptions[k] = i
				} else {
					errs.Add(response.ErrIntegerInvalid(k, fmt.Sprintf("Valid range [%v, %v]", o.GetMinInt(), o.GetMaxInt())))
				}
			default:
				errs.Add(response.ErrIntegerInvalid(k))
			}
		case v1.DatatypeDouble:
			switch v.(type) {
			case float64:
				f := v.(float64)
				if f >= o.GetMinFloat() && f <= o.GetMaxFloat() {
					legalOptions[k] = f
				} else {
					errs.Add(response.ErrDoubleInvalid(k, fmt.Sprintf("Valid range [%v, %v]", o.GetMinFloat(), o.GetMaxFloat())))
				}
			default:
				errs.Add(response.ErrDoubleInvalid(k))
			}
		case v1.DatatypeBool:
			switch v.(type) {
			case bool:
				legalOptions[k] = v
			case string:
				b, err := strconv.ParseBool(v.(string))
				if err == nil {
					legalOptions[k] = b
				} else {
					errs.Add(response.ErrBooleanInvalid(k))
				}
			default:
				errs.Add(response.ErrBooleanInvalid(k))
			}
		case v1.DatatypeTimestamp:
			switch v.(type) {
			case float64:
				legalOptions[k] = int64(v.(float64))
			case string:
				t, err := strconv.ParseInt(v.(string), 10, 64)
				if err == nil {
					legalOptions[k] = t
				} else {
					errs.Add(response.ErrTimestampInvalid(k))
				}
			default:
				errs.Add(response.ErrTimestampInvalid(k))
			}
		case v1.DatatypeEnum:
			if e, ok := v.(string); ok {
				hit := false
				for _, item := range o.Values {
					if item == e {
						hit = true
					}
				}
				if hit {
					legalOptions[k] = e
				} else {
					errs.Add(response.ErrEnumInvalid(k, fmt.Sprintf("Valid values [%s]", strings.Join(o.Values, "|"))))
				}
			} else {
				errs.Add(response.ErrEnumInvalid(k))
			}
		default:
			// should never be here
			klog.V(1).InfoS("Unknown datatype", "datatype", o.Datatype)
		}
	}

	if errs.Len() != 0 {
		return errs
	}

	for k, v := range requiredOptions {
		if _, ok := legalOptions[k]; !ok {
			legalOptions[k] = v.Default
		}
	}

	cmd := &runtime.Command{
		Seq:     randutil.Int63n(),
		ThingId: thingId,
		TypeId:  commandTypeId,
		Time:    time.Now(),
		State:   runtime.StateDelivering,
		Options: legalOptions,
	}

	defer m.logStore.CreateCommand(cmd, commandType)

	m.dispatchCmd(cmd)

	return nil
}

func (m *Manager) dispatchCmd(cmd *runtime.Command) {
	header := http.Header{"Content-Type": []string{"application/json"}}
	resp, err := m.brokerClient.Post("/api/data/v1/exchange", header, map[string]interface{}{"downlink": true}, cmd)
	if err != nil {
		klog.V(1).InfoS("Failed to access iot data broker", "err", err)
		cmd.State = runtime.StateFail
		return
	}
	defer client.Drain(resp)
	if resp.StatusCode == http.StatusOK {
		var ad model.AckData
		if err = json.NewDecoder(resp.Body).Decode(&ad); err != nil {
			klog.V(3).InfoS("Failed parse response", "err", err)
			cmd.State = runtime.StateFail
		} else {
			cmd.Code = ad.Code
			cmd.Message = ad.Message
			if cmd.Code == 0 {
				cmd.State = runtime.StateSuccess
			} else {
				cmd.State = runtime.StateFail
			}
		}
	} else {
		cmd.State = runtime.StateFail
		klog.V(2).InfoS("Received HTTP response", "code", resp.StatusCode)
		var ret response.MultiError
		if err = json.NewDecoder(resp.Body).Decode(&ret); err != nil {
			klog.V(3).InfoS("Failed parse response", "err", err)
		} else {
			s := ret.Error()
			cmd.Message = &s
			klog.V(3).InfoS("Received HTTP response error", "err", s)
		}
	}
	klog.V(3).InfoS("Delivered command", "seq", cmd.Seq, "thing", cmd.ThingId, "result", cmd.State)
}
