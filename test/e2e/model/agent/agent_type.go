package agent

import (
	"encoding/json"
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
	"lightiot/pkg/apis"
	"lightiot/pkg/model/runtime"
	v1 "lightiot/pkg/model/v1"
	"lightiot/test/e2e/common"
	"lightiot/test/util/httpclient"
	"net/http"
)

const createAgentParam = `
{
  "name": "getech_agent",
  "description": "getech_agent_template",
  "dataSources": [
    {
      "name": "DHT11_temperature_sensor",
      "description": "DHT11_temperature_sensor",
      "dataPoints": [
        {
          "id": "analog001",
          "name": "temperature",
          "description": "read temperature",
          "datatype": "double",
          "unit": "centigrade",
          "accessMode": "rw",
          "customData": {
            "host": "10.125.10.101"
          }
        }
      ],
      "customData": {
        "host": "10.125.10.101"
      }
    }
  ]
}
`

var _ = Describe("AgentType", Label("AgentType"), Ordered, func() {
	var modelClient httpclient.TestClient

	BeforeAll(func() {
		modelClient = common.CreateModelClient()
	})

	When("well conditions", func() {
		var agentType runtime.AgentType

		It("should create an agentType", func() {
			res, err := modelClient.Post("/agenttypes", http.Header{}, make(map[string]interface{}, 0), createAgentParam)
			Expect(err).To(BeNil())
			Expect(res).ToNot(BeNil())
			Expect(res.StatusCode).Should(Equal(http.StatusCreated))
			err = json.NewDecoder(res.Body).Decode(&agentType)
			if err != nil {
				Fail("can't decode response body:" + err.Error())
			}
			Expect(err).To(BeNil())
			datasourceNameIdentifier := func(ele interface{}) string {
				source := ele.(*runtime.DataSource)
				return source.Name
			}
			dataPointNameIdentifier := func(ele interface{}) string {
				source := ele.(*runtime.DataPoint)
				return source.Name
			}
			Expect(&agentType).Should(gstruct.PointTo(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
				"Name":        Equal("getech_agent"),
				"Description": gstruct.PointTo(Equal("getech_agent_template")),
				"DataSources": gstruct.MatchAllElements(datasourceNameIdentifier, gstruct.Elements{
					"DHT11_temperature_sensor": gstruct.PointTo(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
						"Description": gstruct.PointTo(Equal("DHT11_temperature_sensor")),
						"DataPoints": gstruct.MatchAllElements(dataPointNameIdentifier, gstruct.Elements{
							"temperature": gstruct.PointTo(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
								"Id":          Equal("analog001"),
								"Name":        Equal("temperature"),
								"Description": gstruct.PointTo(Equal("read temperature")),
								"DataType":    Equal(v1.DataTypeDouble),
								"Unit":        Equal("centigrade"),
								"AccessMode":  Equal(v1.AccessMode(v1.AccessModeReadWrite)),
							})),
						}),
					})),
				}),
			})))
		})

		It("should list agentTypes", func() {
			res, err := modelClient.Get("/agenttypes", nil, nil)
			Expect(err).To(BeNil())
			Expect(res.StatusCode).Should(Equal(http.StatusOK))
			body := runtime.ResponseModel{}
			err = json.NewDecoder(res.Body).Decode(&body)
			Expect(err).To(BeNil())
			Expect(body.AgentTypes).NotTo(BeNil())
			b, ok := body.AgentTypes.([]interface{})
			Expect(ok).To(BeTrue())
			Expect(len(b)).Should(BeNumerically(">=", 1))
		})

		It("should get an agentType by id", func() {
			res, err := modelClient.Get(fmt.Sprintf("/agenttypes/%s", "main.getech_agent"), nil, nil)
			Expect(err).To(BeNil())
			Expect(res.StatusCode).Should(Equal(http.StatusOK))
			at := runtime.AgentType{}
			err = json.NewDecoder(res.Body).Decode(&at)
			Expect(err).To(BeNil())
			Expect(at.Name).Should(Equal("getech_agent"))
			Expect(len(at.DataSources)).Should(BeNumerically(">=", 1))
			Expect(at.DataSources[0].Name).Should(Equal("DHT11_temperature_sensor"))
			Expect(len(at.DataSources[0].DataPoints)).Should(BeNumerically(">=", 1))
			Expect(at.DataSources[0].DataPoints[0].Name).Should(Equal("temperature"))
		})

		It("shouldn't create an agentType of same name", func() {
			res, err := modelClient.Post("/agenttypes", http.Header{}, make(map[string]interface{}, 0), createAgentParam)
			Expect(err).To(BeNil())
			Expect(res).ToNot(BeNil())
			Expect(res.StatusCode).Should(Equal(http.StatusBadRequest))
		})

		It("should delete an agentType", func() {
			header := http.Header{}
			header.Add(apis.IfMatch, agentType.Version)
			res, err := modelClient.Delete(fmt.Sprintf("/agenttypes/%s", "main.getech_agent"), header, nil)
			Expect(err).To(BeNil())
			Expect(res.StatusCode).Should(Equal(http.StatusOK))
		})

		When("an agentType deleted", func() {
			It("shouldn't get an agentType By id", func() {
				res, err := modelClient.Get(fmt.Sprintf("/agenttypes/%s", "main.getech_agent"), nil, nil)
				Expect(err).To(BeNil())
				Expect(res.StatusCode).Should(Equal(http.StatusNotFound))
			})
		})
	})
})
