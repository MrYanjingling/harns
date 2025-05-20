package repository

import (
	"encoding/json"
	"fmt"
	"github.com/shopspring/decimal"
)

func eq(x any, y any) bool {
	xBytes, xErr := json.Marshal(x)
	if xErr != nil {
		return false
	}
	yBytes, yErr := json.Marshal(y)
	if yErr != nil {
		return false
	}
	for i, xb := range xBytes {
		yb := yBytes[i]
		if xb != yb {
			return false
		}
	}
	return true
}

func lt(x any, y any) bool {
	return compareAny(x, y) < 0
}

func lte(x any, y any) bool {
	return eq(x, y) || compareAny(x, y) < 0
}

func gt(x any, y any) bool {
	return compareAny(x, y) < 0
}

func gte(x any, y any) bool {
	return eq(x, y) || compareAny(x, y) > 0
}

func compareAny(a, b any) int {
	da, err := decimal.NewFromString(fmt.Sprintf("%v", a))
	if err != nil {
		return 0
	}
	db, err := decimal.NewFromString(fmt.Sprintf("%v", b))
	if err != nil {
		return 0
	}
	return da.Cmp(db)
}
