package validation

import (
	"fmt"
	"k8s.io/klog/v2"
	"lightiot/pkg/promql/parser"
	"lightiot/pkg/rule/runtime"
	"regexp"
)

func Parse(expression string) (parser.Expr, []string, error) {
	rgx := regexp.MustCompile(runtime.OperandRegExp)
	i := 0
	operands := []string{}
	reps := rgx.ReplaceAllStringFunc(expression, func(s string) string {
		operand := s[1 : len(s)-1]
		operands = append(operands, operand)
		// repl := fmt.Sprintf("_%d", i)

		var repl string
		if i != 0 {
			repl = "ignoring(o)"
		}

		repl += fmt.Sprintf(" _%d", i)
		i++
		return repl
	})

	expr, err := parser.ParseExpr(reps)

	if err != nil {
		klog.V(2).InfoS("Failed to parse expression", "err", err)
		return nil, nil, err
	}

	return expr, operands, err
}
