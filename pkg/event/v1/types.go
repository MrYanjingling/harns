package v1

type EventType struct {
	Id       string   `json:"id" binding:"max=80"`
	Name     string   `json:"name" binding:"required,max=64"`
	ParentId *string  `json:"parentId" binding:"omitempty,max=80"`
	TTL      *int     `json:"ttl" binding:"min=0"`
	Fields   []*Field `json:"fields" binding:"required,dive"`
}

type Field struct {
	Name       string      `json:"name" binding:"required"`
	Filterable bool        `json:"filterable"`
	Required   bool        `json:"required"`
	Updatable  bool        `json:"updatable"`
	DataType   Datatype    `json:"datatype" binding:"gte=0"`
	Values     interface{} `json:"values"`
}

type Datatype byte

const (
	DatatypeString Datatype = iota
	DatatypeInt
	DatatypeDouble
	DatatypeBool
	DatatypeLink
	DatatypeTimestamp
	DatatypeUuid
	DatatypeEnum
	DatatypeMap
)

var DatatypeToString = map[Datatype]string{
	DatatypeString:    "string",
	DatatypeInt:       "int",
	DatatypeDouble:    "double",
	DatatypeBool:      "bool",
	DatatypeLink:      "link",
	DatatypeTimestamp: "timestamp",
	DatatypeUuid:      "uuid",
	DatatypeEnum:      "enum",
	DatatypeMap:       "map",
}

var DatatypeFromString = map[string]Datatype{
	"string":    DatatypeString,
	"int":       DatatypeInt,
	"double":    DatatypeDouble,
	"bool":      DatatypeBool,
	"link":      DatatypeLink,
	"timestamp": DatatypeTimestamp,
	"uuid":      DatatypeUuid,
	"enum":      DatatypeEnum,
	"map":       DatatypeMap,
}

func (dt Datatype) String() string {
	return DatatypeToString[dt]
}

type BaseEvent struct {
	Id            string                 `json:"id" validate:"omitempty,len=32"`
	TypeId        string                 `json:"typeId" validate:"max=80"`
	CorrelationId string                 `json:"correlationId" validate:"omitempty,len=32"`
	Time          string                 `json:"_time" validate:"required"`
	ThingId       string                 `json:"thingId" validate:"required,len=32"`
	Fields        map[string]interface{} `json:",inline"`
}

type StandardEvent struct {
	BaseEvent    `json:",inline"`
	Severity     int    `json:"severity" validate:"omitempty,min=1,max=99"`
	Description  string `json:"description" validate:"max=255"`
	Code         string `json:"code" validate:"max=16"`
	Source       string `json:"source" validate:"max=255"`
	Acknowledged bool   `json:"acknowledged"`
}
