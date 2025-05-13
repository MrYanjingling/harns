package rule

import (
	"bytes"
	"encoding/json"
	"fmt"
	"lightiot/pkg/apis"
	"lightiot/pkg/client"
	"lightiot/pkg/generic"
	model "lightiot/pkg/model/runtime"
	"lightiot/pkg/rule/runtime"
	test "lightiot/test/e2e/common"
	v1 "lightiot/test/e2e/v1"
	"lightiot/test/util/httpclient"
	"net/http"
	"text/template"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
)

var (
	ruleName   = "摄氏度转成华氏度"
	ruleClient httpclient.TestClient
	thingType  *model.ThingType
	thing      *model.Thing
)

var _ = Describe("Rule", Ordered, func() {

	BeforeAll(func() {
		var err error
		ruleClient = test.CreateRuleClient()
		thingTypeData := map[string]string{"ThingTypeName": "e2e-test-rule", "PsName": "传感器属性集", "ParentThingTypeId": "main.BaseAgent"}
		thingType, err = test.CreateThingType(thingTypeData)
		Expect(err).To(BeNil())
		Expect(thingType).ToNot(BeNil())

		thingData := map[string]string{"Name": "e2e-test-rule", "ThingTypeId": thingType.ID}
		thing, err = test.CreateThing(thingData)
		Expect(err).To(BeNil())
		Expect(thing).ToNot(BeNil())
	})

	Describe("Rule crud", func() {
		var rule runtime.Rule

		It("should create rule", func() { createRule(&rule) })

		It("should list rules", func() { listRules() })

		It("should get a rule by id", func() { getRuleById(&rule) })

		It("should delete a rule by id", func() {
			deleteRule(&rule)
			canNotGetRuleById(rule.ID)
		})
	})
})

func createRule(rule *runtime.Rule) {
	ruleTempl, _ := template.New("rule").Parse(v1.CreateRuleTemplate)
	b := new(bytes.Buffer)
	_ = ruleTempl.Execute(b, map[string]string{"Name": ruleName, "ThingId": thing.ID})
	res, err := ruleClient.Post("/rules", http.Header{}, make(map[string]interface{}, 0), b.String())
	defer client.Drain(res)

	Expect(err).To(BeNil())
	Expect(res).ToNot(BeNil())
	Expect(res.StatusCode).Should(Equal(http.StatusCreated))
	err = json.NewDecoder(res.Body).Decode(rule)
	Expect(err).To(BeNil())

	Expect(rule.Name).Should(Equal(ruleName))
	Expect(len(rule.Evaluations)).Should(BeNumerically("==", 1))
	Expect(rule.Evaluations[0].Expression).Should(Equal("(\u001d传感器属性集\u001f温度\u001d * 9) / 5 + 32"))
	Expect(len(rule.Actions.Actions)).Should(BeNumerically("==", 2))

	actions := rule.Actions.Actions
	Expect(actions[runtime.ActionTypeVirtualParameter].(*runtime.VirtualParameter).PropertySetName).Should(Equal("传感器属性集"))
	Expect(actions[runtime.ActionTypeVirtualParameter].(*runtime.VirtualParameter).Property.Name).Should(Equal("温度"))
	Expect(actions[runtime.ActionTypeEvent].(*runtime.Event).Severity).Should(Equal(50))
	Expect(actions[runtime.ActionTypeEvent].(*runtime.Event).BaseAction.Interval).Should(gstruct.PointTo(Equal(generic.Duration{20 * time.Second})))
}

func listRules() {
	res, err := ruleClient.Get("/rules", nil, nil)
	defer client.Drain(res)

	Expect(err).To(BeNil())
	Expect(res.StatusCode).Should(Equal(http.StatusOK))
	body := map[string][]interface{}{
		"rules": make([]interface{}, 0),
	}
	err = json.NewDecoder(res.Body).Decode(&body)
	Expect(err).To(BeNil())
	Expect(body["rules"]).NotTo(BeNil())
	b, ok := body["rules"]
	Expect(ok).To(BeTrue())
	Expect(len(b)).Should(BeNumerically(">=", 1))
}

func getRuleById(rule *runtime.Rule) {
	res, err := ruleClient.Get(fmt.Sprintf("/rules/%s", rule.ID), nil, nil)
	defer client.Drain(res)

	Expect(err).To(BeNil())
	Expect(res).ToNot(BeNil())
	Expect(res.StatusCode).Should(Equal(http.StatusOK))
	r := runtime.Rule{}
	err = json.NewDecoder(res.Body).Decode(&r)
	Expect(err).To(BeNil())
	Expect(r.ID).Should(Equal(rule.ID))
	Expect(r.ThingId).Should(Equal(rule.ThingId))
	Expect(len(r.Evaluations)).Should(BeNumerically(">=", 1))
	Expect(r.Evaluations[0].Expression).Should(Equal(rule.Evaluations[0].Expression))
}

func deleteRule(rule *runtime.Rule) {
	header := http.Header{}
	header.Add(apis.IfMatch, rule.Version)
	res, err := ruleClient.Delete(fmt.Sprintf("/rules/%s", rule.ID), header, nil)
	defer client.Drain(res)
	Expect(err).To(BeNil())
	Expect(res.StatusCode).Should(Equal(http.StatusOK))
}

func canNotGetRuleById(id string) {
	res, err := ruleClient.Get(fmt.Sprintf("/rules/%s", id), nil, nil)
	defer client.Drain(res)
	Expect(err).To(BeNil())
	Expect(res.StatusCode).Should(Equal(http.StatusNoContent))
}
