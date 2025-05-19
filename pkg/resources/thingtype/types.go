package thingtype

type ThingType struct {
	Name            string                         `json:"name"`
	Tenant          string                         `json:"tenant"`
	ID              string                         `json:"id"`
	ParentTypeID    string                         `json:"parentTypeId"`
	Version         string                         `json:"version"`
	CreatedBy       string                         `json:"createdBy"`
	UpdatedBy       string                         `json:"updatedBy"`
	CreatedTime     string                         `json:"createdTime"`
	UpdatedTime     string                         `json:"updatedTime"`
	Description     string                         `json:"description"`
	Characteristics map[string]Characteristic      `json:"characteristics"`
	PropertySets    map[string]map[string]Property `json:"propertySets"`
}

type Characteristic struct {
	Name         string `json:"name"`
	Unit         string `json:"unit"`
	Length       int    `json:"length"`
	DataType     string `json:"dataType"`
	DefaultValue string `json:"defaultValue"`
}

type Property struct {
	Name       string `json:"name"`
	Unit       string `json:"unit"`
	Length     int    `json:"length"`
	DataType   string `json:"dataType"`
	AccessMode string `json:"accessMode"`
	Min        int    `json:"min"`
	Max        int    `json:"max"`
}
