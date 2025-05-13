package exchange

import (
	"lightiot/pkg/model/runtime"
	"lightiot/pkg/util/randutil"
	"path"
	"testing"
)

func benchmarkTTL(f func(map[runtime.PropertySetInfo]bool, map[string]bool, string, string), b *testing.B) {
	testStructs, testStrings := dataProvider()
	thingId, psName := randutil.StringN(32), randutil.StringN(16)
	for n := 0; n < b.N; n++ {
		f(testStructs, testStrings, thingId, psName)
	}
}

func BenchmarkStringKey(b *testing.B) {
	benchmarkTTL(func(testStructs map[runtime.PropertySetInfo]bool, testStrings map[string]bool, thingId, psName string) {
		if _, ok := testStrings[path.Join(thingId, psName)]; ok {
		}
	}, b)
}

func BenchmarkStructKey(b *testing.B) {
	benchmarkTTL(func(testStructs map[runtime.PropertySetInfo]bool, testStrings map[string]bool, thingId, psName string) {
		if _, ok := testStructs[runtime.PropertySetInfo{
			ThingId:         thingId,
			PropertySetName: psName,
		}]; ok {
		}
	}, b)
}

func dataProvider() (map[runtime.PropertySetInfo]bool, map[string]bool) {
	testStructs := make(map[runtime.PropertySetInfo]bool)
	testStrings := make(map[string]bool)

	for i := 0; i < 1000; i++ {
		psi := runtime.PropertySetInfo{
			ThingId:         randutil.StringN(32),
			PropertySetName: randutil.StringN(16),
		}

		testStructs[psi] = true
		testStrings[path.Join(psi.ThingId, psi.PropertySetName)] = true
	}

	return testStructs, testStrings
}
