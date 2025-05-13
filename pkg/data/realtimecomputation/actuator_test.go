package realtimecomputation

// func TestEvaluate(t *testing.T) {
// 	rule := &runtime.Rule{
// 		ThingId:  "thingId",
// 		RealTime: true,
// 		Active:   true,
// 		Evaluations: []runtime.Evaluation{
// 			{
// 				// Expression: "\x1DpropertySetName\x1F温度\x1D",
// 				// Expression: "\x1DpropertySetName\x1F温度\x1D * 9 / 5 + 32",
// 				// Expression: "(\x1DpropertySetName\x1F温度\x1D * 9) / 5 + 32",
// 				// Expression: "\x1DpropertySetName\x1F温度\x1D > 30 or \u001DpropertySetName\u001F温度\u001D < 50",
// 				// Expression: "(\x1DpropertySetName\x1F温度\x1D + \x1DpropertySetName\x1F湿度\x1D) > 87 and \u001DpropertySetName\u001F温度\u001D < 30",
// 				Expression: "\x1DpropertySetName\x1Fonline\x1D",
// 			},
// 		},
// 		Actions: runtime.Actions{},
// 	}
//
// 	timestamp := time.Now()
// 	timestamp2 := timestamp.Add(-3 * time.Second)
// 	data := storage.RawData{
// 		timestamp: map[string]interface{}{
// 			"温度":     26.9,
// 			"湿度":     60,
// 			"online": true,
// 		},
// 		timestamp2: map[string]interface{}{
// 			"温度":     27,
// 			"湿度":     61,
// 			"online": false,
// 		},
// 	}
//
// 	ruleMgr := NewManager(nil)
// 	_ = ruleMgr.onRuleReceived(rule)
// 	a := NewActuator(nil, nil, ruleMgr, nil, nil, nil)
// 	a.Evaluate("thingId/propertySetName", data)
// }
