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
    "characteristics": [
        {
            "name": "长",
            "unit": "m",
            "length": "10",
            "dataType": "float64",
            "defaultValue": "3.33"
        },
        {
            "name": "宽",
            "unit": "m",
            "length": 10,
            "dataType": "float64",
            "defaultValue": "3.33"
        }
    ],
    "propertySets": [
        {
            "name": "default",
            "property": [
                {
                    "name": "温度",
                    "unit": "C",
                    "length": 10,
                    "dataType": "float64",
                    "accessMode": "rw",
                    "min": 35,
                    "max": 22
                }
            ]
        },
        {
            "name": "location",
            "property": [
                {
                    "name": "X",
                    "unit": "",
                    "length": 10,
                    "dataType": "float64",
                    "accessMode": "r",
                    "min": 255,
                    "max": -255
                }
            ]
        }
    ]
}