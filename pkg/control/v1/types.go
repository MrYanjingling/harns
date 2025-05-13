package v1

type Datatype byte

const (
	DatatypeString Datatype = iota
	DatatypeInt
	DatatypeDouble
	DatatypeBool
	DatatypeTimestamp
	DatatypeEnum
)

var DatatypeToString = map[Datatype]string{
	DatatypeString:    "string",
	DatatypeInt:       "int",
	DatatypeDouble:    "double",
	DatatypeBool:      "bool",
	DatatypeTimestamp: "timestamp",
	DatatypeEnum:      "enum",
}

var DatatypeFromString = map[string]Datatype{
	"string":    DatatypeString,
	"int":       DatatypeInt,
	"double":    DatatypeDouble,
	"bool":      DatatypeBool,
	"timestamp": DatatypeTimestamp,
	"enum":      DatatypeEnum,
}

func (dt Datatype) String() string {
	return DatatypeToString[dt]
}

type Option struct {
	Name        string      `json:"name" binding:"required,min=1,max=64,excludesall=\u002F\u005C"`
	Description *string     `json:"description" binding:"omitempty,max=255"`
	Required    bool        `json:"required"`
	Filterable  bool        `json:"filterable"`
	Datatype    Datatype    `json:"datatype"`
	Values      []string    `json:"values,omitempty"`
	Default     interface{} `json:"default,omitempty"`
	Min         *float64    `json:"min,omitempty"`
	Max         *float64    `json:"max,omitempty"`
}

type CommandType struct {
	Name        string    `json:"name" binding:"required,min=1,max=64,excludesall=\u002E\u002F\u003F\u005C"`
	Description *string   `json:"description" binding:"omitempty,max=255"`
	Ack         bool      `json:"ack"`
	Options     []*Option `json:"options,omitempty" binding:"dive"`
}
