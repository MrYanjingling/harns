package rule

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
)

func TestRegExpression(t *testing.T) {
	var rgx = regexp.MustCompile(`\x1D(.*?)\x1D`)
	s := "((\x1D温(-湿度\x1Ftmp\x1D + \x1D温湿#度\x1Fhum\x1D) * \x1D温湿#度\x1F高度\x1D) > 10"
	i := 0
	reps := rgx.ReplaceAllStringFunc(s, func(s string) string{
		variable := s[1:len(s)-1]

		res := strings.Split(variable, "\x1F")
		t.Log(res)

		rep := fmt.Sprintf("_%d", i)
		i++
		return rep
	})

	t.Log(reps)
}
