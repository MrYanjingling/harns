package v1

const (
	CreateRecipientTemplate = `{
  "name": "{{.Name}}",
  "detail": [
    {
      "address": "maintainer@getech.cn",
      "type": "email"
    },
    {
      "address": "maintainer@getech.cn",
      "type": "weCom"
    },
    {
      "address": "maintainer@getech.cn",
      "type": "weChat"
    }
  ]
}`
	CreateMessageTemplate = `{
  "name": "{{.Name}}",
  "type": "text",
  "content": "{{.Content}}"
}`
	UpdateServerConfigTemplate = `{
  "email": {
    "smtp": {
      "hostname": "mail.example.com",
      "port": "587",
      "username": "update@example.com",
      "password": "xxxxxxxx"
    }
  },
  "weCom": {
    "corpId": "xxxxxxxx",
    "corpSecret": "xxxxxxxx",
    "agentId": "3"
  },
  "weChat": {
    "appId": "xxxxxx",
    "appSecret": "xxxxxx"
  }
}`
	CreateServerConfigTemplate = `{
  "email": {
    "smtp": {
      "hostname": "mail.example.com",
      "port": "587",
      "username": "user@example.com",
      "password": "xxxxxxxx"
    }
  },
  "weCom": {
    "corpId": "xxxxxxxx",
    "corpSecret": "xxxxxxxx",
    "agentId": "1"
  },
  "weChat": {
    "appId": "xxxxxx",
    "appSecret": "xxxxxx"
  }
}`
	CreateCommandTypeTemplate = `{
  "name": "{{.Name}}",
  "description": "设置温湿度",
  "options": [
    {
      "name": "{{.OptionName}}",
      "description": "摄氏度",
      "required": true,
      "filterable": true,
      "datatype": "double",
      "values": [
        "string"
      ],
      "default": "25",
      "min": 18,
      "max": 32
    }
  ]
}`
	CreateRuleTemplate = `{
  "name": "{{.Name}}",
  "description": "摄氏度转成华氏度",
  "thingId": "{{.ThingId}}",
  "realTime": true,
  "active": true,
  "evaluations": [
    {
      "expression": "(\u001d传感器属性集\u001f温度\u001d * 9) / 5 + 32"
    }
  ],
  "actions": {
    "virtualParameter": {
      "active": true,
      "thingId": "{{.ThingId}}",
      "propertySetName": "传感器属性集",
      "property": {
        "name": "温度",
        "datatype": "double",
        "unit": "度",
        "length": 0
      }
    },
    "event": {
      "active": true,
      "interval": "20s",
      "severity": 50,
      "description": "峨眉山会议室温度过高"
    }
  }
}`
	CreateThingTemplate = `
{
	"name": "{{.Name}}",
	"description": "this thing is for test",
	"typeId": "{{.ThingTypeId}}",
	"characteristics": []
}`
	CreateThingTypeTemplate = `
{
  "characteristics": [],
  "instantiable": true,
  "name": "{{.ThingTypeName}}",
  "parentTypeId": "{{.ParentThingTypeId}}",
  "propertysets": [
    {
	  "name": "{{.PsName}}",
      "properties": [
        {
          "datatype": "double",
          "accessMode": "rw",
          "name": "温度",
          "unit": "度"
        }
      ]
    }
  ]
}`
)
