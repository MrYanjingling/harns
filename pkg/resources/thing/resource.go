package thing

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
	return "things", resources.Resource{
		Table: "thing",
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
        "parentId": {
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
                    "value": {
                        "type": "string"
                    }
                },
                "required": [
                    "name",
                    "value"
                ]
            }
        },
        "combination": {
            "type": "array",
            "items": {
                "type": "object",
                "properties": {
                    "thing": {
                        "type": "string"
                    },
                    "combination": {
                        "type": "array",
                        "items": {}
                    }
                },
                "required": [
                    "thing",
                    "combination"
                ]
            }
        }
    },
    "required": [
        "name",
        "tenant",
        "id",
        "parentId",
        "version",
        "createdBy",
        "updatedBy",
        "createdTime",
        "updatedTime",
        "characteristics",
        "combination"
    ]
}
    
`
