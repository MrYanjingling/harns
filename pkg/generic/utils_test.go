package generic

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"reflect"
	"sort"
	"testing"
)

func TestRemoveString(t *testing.T) {
	var tests = []struct {
		in     []string
		i      int
		expect []string
	}{
		{[]string{"apple", "boy", "cat", "dog"}, 0, []string{"boy", "cat", "dog"}},
		{[]string{"apple", "boy", "cat", "dog"}, 1, []string{"apple", "cat", "dog"}},
		{[]string{"apple", "boy", "cat", "dog"}, 2, []string{"apple", "boy", "dog"}},
		{[]string{"apple", "boy", "cat", "dog"}, 3, []string{"apple", "boy", "cat"}},
	}

	for _, tt := range tests {
		testName := fmt.Sprintf("Remove [%d]: %s", tt.i, tt.in[tt.i])
		t.Run(testName, func(t *testing.T) {
			actual := RemoveString(tt.in, tt.i)
			sort.Strings(actual)
			if !reflect.DeepEqual(actual, tt.expect) {
				t.Errorf("actual %v, expect %v", actual, tt.expect)
			}
		})
	}
}

func TestRemoveStringWithOrder(t *testing.T) {
	var tests = []struct {
		in     []string
		i      int
		expect []string
	}{
		{[]string{"apple", "boy", "cat", "dog"}, 0, []string{"boy", "cat", "dog"}},
		{[]string{"apple", "boy", "cat", "dog"}, 1, []string{"apple", "cat", "dog"}},
		{[]string{"apple", "boy", "cat", "dog"}, 2, []string{"apple", "boy", "dog"}},
		{[]string{"apple", "boy", "cat", "dog"}, 3, []string{"apple", "boy", "cat"}},
	}

	for _, tt := range tests {
		testName := fmt.Sprintf("Remove [%d]: %s", tt.i, tt.in[tt.i])
		t.Run(testName, func(t *testing.T) {
			actual := RemoveStringWithOrder(tt.in, tt.i)
			if !reflect.DeepEqual(actual, tt.expect) {
				t.Errorf("actual %v, expect %v", actual, tt.expect)
			}
		})
	}
}

func TestDifferenceAndIntersection(t *testing.T) {
	type In struct {
		src []string
		des []string
	}

	type Expect struct {
		deleted  []string
		updated  []string
		inserted []string
	}

	var tests = []struct {
		in     In
		expect Expect
	}{
		{in: In{[]string{"1", "2"}, []string{}}, expect: Expect{[]string{"1", "2"}, []string{}, []string{}}},
		{in: In{[]string{}, []string{"3", "4", "5"}}, expect: Expect{[]string{}, []string{}, []string{"3", "4", "5"}}},
		{in: In{[]string{"1", "2"}, []string{"3", "4", "5"}}, expect: Expect{[]string{"1", "2"}, []string{}, []string{"3", "4", "5"}}},
		{in: In{[]string{"1", "2", "3"}, []string{"3", "4", "5"}}, expect: Expect{[]string{"1", "2"}, []string{"3"}, []string{"4", "5"}}},
		{in: In{[]string{"1", "2", "3", "4", "6"}, []string{"3", "4", "5", "7"}}, expect: Expect{[]string{"1", "2", "6"}, []string{"3", "4"}, []string{"5", "7"}}},
	}

	for _, tt := range tests {
		testName := fmt.Sprintf("src %v: des %v", tt.in.src, tt.in.des)
		t.Run(testName, func(t *testing.T) {
			del, up, in := DifferenceAndIntersectionStrings(tt.in.src, tt.in.des)
			assert.ElementsMatch(t, tt.expect.deleted, del)
			assert.ElementsMatch(t, tt.expect.updated, up)
			assert.ElementsMatch(t, tt.expect.inserted, in)
		})
	}
}

func TestDifferenceAndIntersectionObjects(t *testing.T) {
	type Object struct {
		Name string
	}

	type In struct {
		src []Object
		des []Object
	}

	type Expect struct {
		deleted  []string
		updated  []string
		inserted []string
	}

	var tests = []struct {
		in     In
		expect Expect
	}{
		{in: In{[]Object{{"1"}, {"2"}}, []Object{}}, expect: Expect{[]string{"1", "2"}, []string{}, []string{}}},
		{in: In{[]Object{}, []Object{{"3"}, {"4"}, {"5"}}}, expect: Expect{[]string{}, []string{}, []string{"3", "4", "5"}}},
		{in: In{[]Object{{"1"}, {"2"}}, []Object{{"3"}, {"4"}, {"5"}}}, expect: Expect{[]string{"1", "2"}, []string{}, []string{"3", "4", "5"}}},
		{in: In{[]Object{{"1"}, {"2"}, {"3"}}, []Object{{"3"}, {"4"}, {"5"}}}, expect: Expect{[]string{"1", "2"}, []string{"3"}, []string{"4", "5"}}},
		{in: In{[]Object{{"1"}, {"2"}, {"3"}, {"4"}, {"6"}}, []Object{{"3"}, {"4"}, {"5"}, {"7"}}}, expect: Expect{[]string{"1", "2", "6"}, []string{"3", "4"}, []string{"5", "7"}}},
	}

	for _, tt := range tests {
		testName := fmt.Sprintf("src %v: des %v", tt.in.src, tt.in.des)
		t.Run(testName, func(t *testing.T) {
			del, up, in := DifferenceAndIntersectionSameTypeObjects(tt.in.src, tt.in.des, func(obj interface{}) string {
				v, _ := obj.(Object)
				return v.Name
			})
			assert.ElementsMatch(t, tt.expect.deleted, del)
			assert.ElementsMatch(t, tt.expect.updated, up)
			assert.ElementsMatch(t, tt.expect.inserted, in)
		})
	}

}
