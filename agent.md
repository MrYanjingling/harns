# Agent
```json5
{
    "name": "网关1",
    "tenant": "main",
    "id": "fastUuid",
    "version": "1289999",
    "createdBy": "anymouse",
    "updatedBy": "anymouse",
    "createdTime": "2025-05-15 12:00:00.000",
    "updatedTime": "2025-05-15 12:00:00.000",
    "description": "this is s senssor thing type",
    "agentType": "Modbus",
    //MQTT modbusTCP opcUa etc
    "communication": {
        "type": "modbusTcp",
        "collectorCycle": 1000,
        "slave": 1,
        "memoryLayout": "ABCD"
    },
    "address": {
        "location": "",
        "port": "",
        "username": "",
        "password": ""
    }
    "datasources": [
        {
            "source": {
                "dataType": "float64",
                "address": 1,
                "bit": 1,
                "functionCode": 3,
                "rate": 1.1,
                "offset": -1.1,
                "amount": 2,
                "defaultValue": 1.23,
                "max": 1.43,
                "min": 0.99,
                "value": 1.33,
                "accessMode": "rw"
            },
            "target": {
                "thingId": "",
                "thingName": "",
                "thingTypeName": "",
                "thingTypeId": "",
                "propertySetName": "",
                "property": ""
            }
        }
    ]
}
```