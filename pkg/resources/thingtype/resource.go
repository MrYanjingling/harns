package thingtype

import (
	"github.com/bytedance/sonic"
	"lightiot/pkg/repository"
	"lightiot/pkg/resources"
)

func Resource() (resources.ResourceName, resources.Resource) {
	schema, err := sonic.Get([]byte(jsonschema))
	if err != nil {
		panic(err)
	}
	return "thingTypes", resources.Resource{
		Table: "thing_type",
		Schema: repository.JsonSchema{
			Node: schema,
		},
	}
}

const jsonschema = `
{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "type": "object",
    "properties": {
        "name": {
            "type": "string"
        },
        "tenant": {
            "type": "string"
        },
        "id": {
            "type": "string"
        },
        "parentTypeId": {
            "type": "string"
        },
        "version": {
            "type": "string"
        },
        "createdBy": {
            "type": "string"
        },
        "updatedBy": {
            "type": "string"
        },
        "createdTime": {
            "type": "string",
            "format": "date-time"
        },
        "updatedTime": {
            "type": "string",
            "format": "date-time"
        },
        "description": {
            "type": "string"
        },
        "characteristics": {
            "type": "object",
            "additionalProperties": {
                "type": "object",
                "properties": {
                    "name": {
                        "type": "string"
                    },
                    "unit": {
                        "type": "string"
                    },
                    "length": {
                        "anyOf": [
                            {
                                "type": "integer"
                            },
                            {
                                "type": "string"
                            }
                        ]
                    },
                    "dataType": {
                        "type": "string"
                    },
                    "defaultValue": {
                        "type": "string"
                    }
                },
                "required": [
                    "name",
                    "unit",
                    "length",
                    "dataType",
                    "defaultValue"
                ]
            }
        },
        "propertySets": {
            "type": "object",
            "additionalProperties": {
                "type": "object",
                "additionalProperties": {
                    "type": "object",
                    "properties": {
                        "name": {
                            "type": "string"
                        },
                        "unit": {
                            "type": "string"
                        },
                        "length": {
                            "type": "integer"
                        },
                        "dataType": {
                            "type": "string"
                        },
                        "accessMode": {
                            "type": "string"
                        },
                        "min": {
                            "type": "integer"
                        },
                        "max": {
                            "type": "integer"
                        }
                    },
                    "required": [
                        "name",
                        "unit",
                        "length",
                        "dataType",
                        "accessMode",
                        "min",
                        "max"
                    ]
                }
            }
        }
    },
    "required": [
        "name",
        "tenant",
        "id",
        "parentTypeId",
        "version",
        "createdBy",
        "updatedBy",
        "createdTime",
        "updatedTime",
        "description",
        "characteristics",
        "propertySets"
    ]
}
`
