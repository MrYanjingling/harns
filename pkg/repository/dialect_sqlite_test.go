package repository

import (
	"testing"
)

func Test_parseJsonKey(t *testing.T) {
	d := dialectSqlite{}
	if d.parseJsonKey("labels.color") != "JSON_EXTRACT(labels,'$.color')" {
		t.Errorf("parse failed")
	}
}
