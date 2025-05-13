package runtime

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"lightiot/pkg/generic"
	model "lightiot/pkg/model/runtime"
	v1 "lightiot/pkg/model/v1"
	"testing"
	"time"
)

func TestCopyRule(t *testing.T) {
	desc := "Just for test"
	var tests = []struct {
		in Rule
	}{
		{
			Rule{
				ThingId:     "123456",
				Description: &desc,
				Evaluations: []Evaluation{{
					Template:   "",
					Expression: "(\\u001d传感器属性集\\u001f温度\\u001d * 9) / 5 + 32",
				}},
				Actions: Actions{map[ActionType]Actioner{
					ActionTypeVirtualParameter: &VirtualParameter{
						ThingId:         "123456",
						PropertySetName: "传感器属性集",
						Property: model.Property{
							Name:     "温度",
							DataType: v1.DataTypeDouble,
							Unit:     "摄氏度",
							Length:   3,
						},
						BaseAction: &BaseAction{
							Active: true,
						},
					},
					ActionTypeEvent: &Event{
						BaseAction: &BaseAction{
							Active:   true,
							Interval: &generic.Duration{time.Duration(20)},
						},
						Severity:    50,
						Description: "峨眉山会议室温度过高",
					},
				}},
			},
		},
	}

	for _, tt := range tests {
		testName := fmt.Sprintf("Copy rule")
		t.Run(testName, func(t *testing.T) {
			actual := *tt.in.DeepCopyObject().(*Rule)

			t.Logf("%p, %p", &tt.in, &actual)
			assert.Equal(t, tt.in, actual)
			assert.NotSame(t, &tt.in, &actual)

			t.Logf("%p, %p", tt.in.Evaluations, actual.Evaluations)
			assert.Equal(t, tt.in.Evaluations, actual.Evaluations)
			assert.NotSame(t, tt.in.Evaluations, actual.Evaluations)

			t.Logf("%p, %p", &tt.in.Evaluations[0], &actual.Evaluations[0])
			assert.Equal(t, tt.in.Evaluations[0], actual.Evaluations[0])
			assert.NotSame(t, &tt.in.Evaluations[0], &actual.Evaluations[0])

			t.Logf("%p, %p", &tt.in.Actions, &actual.Actions)
			assert.Equal(t, tt.in.Actions, actual.Actions)
			assert.NotSame(t, &tt.in.Actions, &actual.Actions)

			t.Logf("%p, %p", tt.in.Actions.Actions[ActionTypeVirtualParameter], actual.Actions.Actions[ActionTypeVirtualParameter])
			assert.Equal(t, tt.in.Actions.Actions[ActionTypeVirtualParameter], actual.Actions.Actions[ActionTypeVirtualParameter])
			assert.NotSame(t, tt.in.Actions.Actions[ActionTypeVirtualParameter], actual.Actions.Actions[ActionTypeVirtualParameter])

			t.Logf("%p, %p", tt.in.Actions.Actions[ActionTypeVirtualParameter].(*VirtualParameter).BaseAction, actual.Actions.Actions[ActionTypeVirtualParameter].(*VirtualParameter).BaseAction)
			assert.Equal(t, tt.in.Actions.Actions[ActionTypeVirtualParameter].(*VirtualParameter).BaseAction, actual.Actions.Actions[ActionTypeVirtualParameter].(*VirtualParameter).BaseAction)
			assert.NotSame(t, tt.in.Actions.Actions[ActionTypeVirtualParameter].(*VirtualParameter).BaseAction, actual.Actions.Actions[ActionTypeVirtualParameter].(*VirtualParameter).BaseAction)

			t.Logf("%p, %p", tt.in.Actions.Actions[ActionTypeEvent].(*Event).BaseAction, actual.Actions.Actions[ActionTypeEvent].(*Event).BaseAction)
			assert.Equal(t, tt.in.Actions.Actions[ActionTypeEvent].(*Event).BaseAction, actual.Actions.Actions[ActionTypeEvent].(*Event).BaseAction)
			assert.NotSame(t, tt.in.Actions.Actions[ActionTypeEvent].(*Event).BaseAction, actual.Actions.Actions[ActionTypeEvent].(*Event).BaseAction)
		})
	}
}
