package agent

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
	return "agents", resources.Resource{
		Table: "agent",
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
            "pattern": "^\\d{4}-\\d{2}-\\d{2} \\d{2}:\\d{2}:\\d{2}\\.\\d{3}$"
        },
        "updatedTime": {
            "type": "string",
            "pattern": "^\\d{4}-\\d{2}-\\d{2} \\d{2}:\\d{2}:\\d{2}\\.\\d{3}$"
        },
        "description": {
            "type": "string"
        },
        "agentType": {
            "type": "string",
            "enum": ["Modbus", "MQTT", "modbusTCP", "opcUa"]
        },
        "communication": {
            "type": "object",
            "properties": {
                "type": {
                    "type": "string",
                    "enum": ["modbusTcp"]
                },
                "collectorCycle": {
                    "type": "integer"
                },
                "slave": {
                    "type": "integer"
                },
                "memoryLayout": {
                    "type": "string",
                    "enum": ["ABCD"]
                }
            },
            "required": ["type", "collectorCycle", "slave", "memoryLayout"]
        },
        "address": {
            "type": "object",
            "properties": {
                "location": {
                    "type": "string"
                },
                "port": {
                    "type": "string"
                },
                "username": {
                    "type": "string"
                },
                "password": {
                    "type": "string"
                }
            },
            "required": ["location", "port", "username", "password"]
        },
        "datasources": {
            "type": "array",
            "items": {
                "type": "object",
                "properties": {
                    "source": {
                        "type": "object",
                        "properties": {
                            "dataType": {
                                "type": "string",
                                "enum": ["float64"]
                            },
                            "address": {
                                "type": "integer"
                            },
                            "bit": {
                                "type": "integer"
                            },
                            "functionCode": {
                                "type": "integer"
                            },
                            "rate": {
                                "type": "number"
                            },
                            "offset": {
                                "type": "number"
                            },
                            "amount": {
                                "type": "integer"
                            },
                            "defaultValue": {
                                "type": "number"
                            },
                            "max": {
                                "type": "number"
                            },
                            "min": {
                                "type": "number"
                            },
                            "value": {
                                "type": "number"
                            },
                            "accessMode": {
                                "type": "string",
                                "enum": ["rw"]
                            }
                        },
                        "required": [
                            "dataType", "address", "bit", "functionCode", 
                            "rate", "offset", "amount", "defaultValue", 
                            "max", "min", "value", "accessMode"
                        ]
                    },
                    "target": {
                        "type": "object",
                        "properties": {
                            "thingId": {
                                "type": "string"
                            },
                            "thingName": {
                                "type": "string"
                            },
                            "thingTypeName": {
                                "type": "string"
                            },
                            "thingTypeId": {
                                "type": "string"
                            },
                            "propertySetName": {
                                "type": "string"
                            },
                            "property": {
                                "type": "string"
                            }
                        },
                        "required": [
                            "thingId", "thingName", "thingTypeName", 
                            "thingTypeId", "propertySetName", "property"
                        ]
                    }
                },
                "required": ["source", "target"]
            }
        }
    },
    "required": [
        "name", "tenant", "id", "version", "createdBy", 
        "updatedBy", "createdTime", "updatedTime", "description", 
        "agentType", "communication", "address", "datasources"
    ]
}
`
