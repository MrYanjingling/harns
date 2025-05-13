package common

import (
	"bytes"
	"encoding/json"
	"errors"
	"lightiot/pkg/client"
	"lightiot/pkg/model/runtime"
	v1 "lightiot/test/e2e/v1"
	"net/http"
	"text/template"
)

func CreateThingType(data map[string]string) (*runtime.ThingType, error) {
	var tt runtime.ThingType
	thingTypeTempl, _ := template.New("thingType").Parse(v1.CreateThingTypeTemplate)
	b := new(bytes.Buffer)
	_ = thingTypeTempl.Execute(b, data)

	res, err := CreateModelClient().Post("/thingtypes", http.Header{}, make(map[string]interface{}, 0), b.String())
	defer client.Drain(res)
	if err != nil {
		return nil, err
	}
	if res == nil || res.StatusCode != http.StatusCreated {
		return nil, errors.New("failed to create thingType")
	}
	if err = json.NewDecoder(res.Body).Decode(&tt); err != nil {
		return nil, err
	}
	return &tt, err
}

func CreateThing(data map[string]string) (*runtime.Thing, error) {
	var t runtime.Thing
	thingTempl, _ := template.New("thing").Parse(v1.CreateThingTemplate)
	b := new(bytes.Buffer)
	_ = thingTempl.Execute(b, data)

	res, err := CreateModelClient().Post("/things", http.Header{}, make(map[string]interface{}, 0), b.String())
	defer client.Drain(res)
	if err != nil {
		return nil, err
	}
	if res == nil || res.StatusCode != http.StatusCreated {
		return nil, errors.New("failed to create thing")
	}
	if err = json.NewDecoder(res.Body).Decode(&t); err != nil {
		return nil, err
	}
	return &t, err
}
