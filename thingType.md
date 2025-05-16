# ThingType
```json
{
    "name": "sensor",
    "tenant": "main",
    "id": "fastUuid",
    "parentTypeId": "parentFastUuid",
    "version": "1289999",
    "createdBy": "anymouse",
    "updatedBy": "anymouse",
    "createdTime": "2025-05-15 12:00:00.000",
    "updatedTime": "2025-05-15 12:00:00.000",
    "description": "this is s senssor thing type",
    "characteristics": {
        "length": {
            "name": "长",
            "unit": "m",
            "length": "10",
            "dataType": "float64",
            "defaultValue": "3.33"
        },
        "width": {
            "name": "宽",
            "unit": "m",
            "length": 10,
            "dataType": "float64",
            "defaultValue": "3.33"
        }
    },
    "propertySets": {
        "default": {
            "temperature": {
                "name": "温度",
                "unit": "C",
                "length": 10,
                "dataType": "float64",
                "accessMode": "rw",
                "min": 35,
                "max": 22
            }
        },
        "location": {
            "x": {
                "name": "X",
                "unit": "",
                "length": 10,
                "dataType": "float64",
                "accessMode": "r",
                "min": 255,
                "max": -255
            },
            "y": {
                "name": "Y",
                "unit": "",
                "length": 10,
                "dataType": "float64",
                "accessMode": "r",
                "min": 255,
                "max": -255
            }
        }
    }
}
```