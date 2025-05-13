package data

import (
	"lightiot/pkg/apis/response"
	model "lightiot/pkg/model/runtime"
	v1 "lightiot/pkg/model/v1"
	"regexp"
	"strconv"
	"strings"
	"time"

	"k8s.io/klog/v2"
)

type RequestValidator struct {
	request   *Request
	startDate time.Time
	endDate   time.Time
	location  *time.Location
	interval  Interval
	limit     int
	weekStart time.Weekday
}

var intervalRE = regexp.MustCompile(`^(?P<amount>\d{1,2})(?P<unit>[s|m|h|D|W|M])$`)

func (validator *RequestValidator) parseDates() error {
	start, err := time.Parse(time.RFC3339Nano, validator.request.start)
	if err != nil {
		return response.ErrRollupStartInvalid(validator.request.start)
	}
	end, err := time.Parse(time.RFC3339Nano, validator.request.end)
	if err != nil {
		return response.ErrRollupEndInvalid(validator.request.end)
	}

	interval := ToInterval(validator.request.interval)
	if interval == nil {
		return response.ErrRollupIntervalInvalid(validator.request.interval)
	}

	validator.interval = *interval
	validator.startDate = start
	validator.endDate = end
	return nil
}

func (validator *RequestValidator) alignStartDate() error {
	year, month, day := validator.startDate.Date()
	hour, minute, second := validator.startDate.Clock()
	switch validator.interval.unit {
	case QueryIntervalUnitMonth:
		day = 1
		hour = 0
		minute = 0
		second = 0
	case QueryIntervalUnitWeek:
		yearOfWeek, _ := validator.startDate.ISOWeek()
		if yearOfWeek == year {
			day -= int(validator.startDate.Weekday() - validator.weekStart)
		} else {
			day += (DaysOneWeek - int(validator.startDate.Weekday()-validator.weekStart)) % DaysOneWeek
		}
		fallthrough
	case QueryIntervalUnitDay:
		hour = 0
		fallthrough
	case QueryIntervalUnitHour:
		minute = 0
		fallthrough
	case QueryIntervalUnitMinute:
		second = 0
	case QueryIntervalUnitSecond:
	default:
		return response.ErrRollupIntervalInvalid(validator.request.interval)
	}
	validator.startDate = time.Date(year, month, day, hour, minute, second, 0, validator.location)
	return nil
}

func (validator *RequestValidator) alignEndDate() error {
	year, month, day := validator.endDate.Date()
	hour, minute, second := validator.endDate.Clock()
	var padding time.Duration
	switch validator.interval.unit {
	case QueryIntervalUnitMonth:
		if day > 1 || second > 0 || minute > 0 || hour > 0 {
			day = 1
			year, month, _ = validator.endDate.AddDate(0, validator.interval.amount, 0).Date()
			hour = 0
			minute = 0
			second = 0
		}
	case QueryIntervalUnitWeek:
		day += (DaysOneWeek - int(validator.endDate.Weekday()-validator.weekStart)) % DaysOneWeek
		if second > 0 || minute > 0 || hour > 0 {
			hour = 0
			minute = 0
			second = 0
		}
	case QueryIntervalUnitDay:
		if second > 0 || minute > 0 || hour > 0 {
			padding += QueryIntervalUnitToDuration[QueryIntervalUnitDay]
			hour = 0
			minute = 0
			second = 0
		}
	case QueryIntervalUnitHour:
		if second > 0 || minute > 0 {
			padding += QueryIntervalUnitToDuration[QueryIntervalUnitHour]
			minute = 0
			second = 0
		}
	case QueryIntervalUnitMinute:
		if second > 0 {
			padding += QueryIntervalUnitToDuration[QueryIntervalUnitMinute]
			second = 0
		}
	case QueryIntervalUnitSecond:
	default:
		return response.ErrRollupIntervalInvalid(validator.request.interval)
	}
	validator.endDate = time.Date(year, month, day, hour, minute, second, 0, validator.location)
	validator.endDate = validator.endDate.Add(padding)

	return nil
}

func (validator *RequestValidator) alignDate(manager *Manager) error {
	thing, err := manager.GetThingById(validator.request.thingId)
	if err != nil {
		return response.ErrThingNotFound(validator.request.thingId)
	}
	location := (*time.Location)(thing.TimeZone)
	// if location == nil {
	// 	return response.ErrTimeZoneInvalid(thing.TimeZone.String())
	// }
	validator.startDate = validator.startDate.UTC().In(location)
	validator.endDate = validator.endDate.UTC().In(location)
	validator.location = location
	if validator.request.weekStartFromSunday {
		validator.weekStart = time.Sunday
	} else {
		validator.weekStart = time.Monday
	}

	if err := validator.alignStartDate(); err != nil {
		return err
	}
	if err := validator.alignEndDate(); err != nil {
		return err
	}
	return nil
}

func (validator *RequestValidator) GetPSTAndSelects(manager *Manager) (*model.PropertySetType, map[string][]byte, error) {
	thing, err := manager.GetThingById(validator.request.thingId)
	if err != nil {
		klog.V(3).InfoS("Thing not found", "id", validator.request.thingId)
		return nil, nil, response.ErrThingNotFound(validator.request.thingId)
	}

	ps, ok := thing.PropertySetByName[validator.request.propertySetName]
	if !ok {
		klog.V(3).InfoS("PropertySet not found", "propertySet", validator.request.propertySetName, "thingId", validator.request.thingId)
		return nil, nil, response.ErrPropertySetTypeNotFound(validator.request.propertySetName)
	}

	propertiesByName := ps.PropertySetType.PropertyByName
	legalProperties := make(map[string][]byte)
	var selects []string
	if len(validator.request.filter) > 0 {
		selects = strings.Split(validator.request.filter, ",")
	}

	if len(selects) > 0 {
		for _, s := range selects {
			ss := strings.Split(s, ".")
			p := ss[0]
			if property, ok := propertiesByName[p]; ok {
				if _, exist := legalProperties[p]; !exist {
					legalProperties[p] = make([]byte, 0)
				}

				if property.DataType == v1.DataTypeInt || property.DataType == v1.DataTypeDouble || property.DataType == v1.DataTypeLong {
					if len(ss) == 1 {
						legalProperties[p] = allRollupNumericFields
						continue
					}

					field := strings.ToLower(ss[1])
					if f, ok := RollupNumericFieldFromString[field]; ok {
						legalProperties[p] = append(legalProperties[p], byte(f))
					} else {
						klog.V(4).InfoS("Invalid numeric rollup field", "name", property.Name, "field", ss[1])
					}
				} else if property.DataType == v1.DataTypeBoolean {
					if len(ss) == 1 {
						legalProperties[p] = allRollupBoolFields
						continue
					}

					field := strings.ToLower(ss[1])
					if f, ok := RollupBoolFieldFromString[field]; ok {
						legalProperties[p] = append(legalProperties[p], byte(f))
					} else {
						klog.V(4).InfoS("Invalid boolean rollup field", "name", property.Name, "field", ss[1])
					}
				} else {
					klog.V(3).InfoS("Invalid rollup property", "name", property.Name, "datatype", property.DataType)
				}
			} else {
				klog.V(3).InfoS("Property not found", "property", p, "propertySet", validator.request.propertySetName, "thingId", validator.request.thingId)
			}
		}
		if len(legalProperties) == 0 {
			return ps.PropertySetType, nil, response.ErrAllSelectInvalid
		}
	} else {
		for k, property := range propertiesByName {
			if property.DataType == v1.DataTypeInt || property.DataType == v1.DataTypeDouble || property.DataType == v1.DataTypeLong {
				legalProperties[k] = allRollupNumericFields
			} else if property.DataType == v1.DataTypeBoolean {
				legalProperties[k] = allRollupBoolFields
			}
		}
	}

	return ps.PropertySetType, legalProperties, nil
}

func (validator *RequestValidator) CheckQueries() error {
	if err := validator.parseDates(); err != nil {
		return err
	}
	if validator.startDate.After(validator.endDate) {
		klog.V(2).InfoS("End time was earlier than start time")
		return response.ErrTimeInvalid(validator.request.end)
	}

	if validator.interval.unit != QueryIntervalUnitMonth {
		if validator.startDate.Add(validator.interval.toDuration()).After(validator.endDate) {
			klog.V(2).InfoS("The duration between start and end was less than interval")
			return response.ErrRollupIntervalInvalid(validator.request.interval)
		}
		return nil
	}

	sy, sm, _ := validator.startDate.Date()
	ey, em, _ := validator.endDate.Date()

	if (ey-sy)*12+(int(em-sm)) < validator.interval.amount {
		klog.V(2).InfoS("The duration between start and end was less than interval")
		return response.ErrRollupIntervalInvalid(validator.request.interval)
	}
	return nil
}

func (validator *RequestValidator) PrepareRequest(manager *Manager) error {
	if err := validator.alignDate(manager); err != nil {
		return err
	}
	// date, _ := json.Marshal(validator.startDate)
	// validator.request.start = string(date)
	// date, _ = json.Marshal(validator.endDate)
	// validator.request.end = string(date)
	return nil
}

func ToInterval(in string) *Interval {
	match := intervalRE.FindStringSubmatch(in)
	if len(match) != 3 {
		return nil
	}
	a, _ := strconv.Atoi(match[1])
	u := []byte(match[2])[0]

	return &Interval{
		amount: a,
		unit:   QueryIntervalUnit(u),
	}
}

func (i *Interval) toDuration() time.Duration {
	return time.Duration(i.amount) * QueryIntervalUnitToDuration[i.unit]
}
