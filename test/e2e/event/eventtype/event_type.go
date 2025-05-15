package eventtype

import (
	"encoding/json"
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
	"lightiot/pkg/apis"
	"lightiot/pkg/event/runtime"
	"lightiot/pkg/event/v1"
	"lightiot/test/e2e/common"
	"lightiot/test/util/httpclient"
	"math"
	"net/http"
	"strconv"
)

const createEventTypeParam = `{
    "name": "eventtype2",
    "parentId": "main.BaseEvent",
    "ttl": 10,
    "fields": [
        {
            "name": "f1",
            "filterable": true,
            "required": false,
            "updatable": true,
            "datatype": "enum",
            "values": [
                "v1"
            ]
        },
        {
            "name": "f2",
            "filterable": false,
            "required": true,
            "updatable": false,
            "datatype": "int"
        }
    ]
}
`

var (
	eventClient httpclient.TestClient
)

var _ = Describe("EventType", Ordered, func() {
	BeforeAll(func() {
		eventClient = common.CreateEventClient()
	})

	Context("EventType create list and read", func() {
		var et v1.EventType
		When("input right parameters", func() {
			It("should create an EventType", func() { createEventType(&et) })

			It("should list EventTypes", func() { listEventTypes() })

			It("should get an EventType by id", func() { getEventTypeById(&et) })

			It("should delete an EventType error and http response code 428", func() { deleteEventTypeError428(&et) })

			It("should delete an EventType error and http response code 404", func() { deleteEventTypeError404(&et) })

			It("should delete an EventType error and http response code 412", func() { deleteEventTypeError412(&et) })

			It("should delete an EventType error and http response code 400", func() { deleteEventTypeError400() })

			It("should delete an EventType success", func() { deleteEventType(&et) })
		})
	})

	AfterAll(func() {

	})

})

func matchEventType1(et *v1.EventType) {
	fieldNameIdentifier := func(ele interface{}) string {
		source := ele.(*v1.Field)
		return source.Name
	}

	Expect(et).To(gstruct.PointTo(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
		"Id":       Equal("main.eventtype2"),
		"Name":     Equal("eventtype2"),
		"ParentId": gstruct.PointTo(Equal("main.BaseEvent")),
		"TTL":      gstruct.PointTo(Equal(10)),
		"Properties": gstruct.MatchAllElements(fieldNameIdentifier, gstruct.Elements{
			"f1": gstruct.PointTo(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
				"Name":       Equal("f1"),
				"Filterable": Equal(true),
				"Required":   Equal(false),
				"Updatable":  Equal(true),
				"DataType":   Equal(v1.DatatypeEnum),
				"Values":     Equal([]interface{}{"v1"}),
			})),
			"f2": gstruct.PointTo(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
				"Name":       Equal("f2"),
				"Filterable": Equal(false),
				"Required":   Equal(true),
				"Updatable":  Equal(false),
				"DataType":   Equal(v1.DatatypeInt),
			})),
		}),
	})))
}

func createEventType(et *v1.EventType) {
	res, err := eventClient.Post("/eventtypes", http.Header{}, nil, createEventTypeParam)
	Expect(err).To(BeNil())
	defer httpclient.Drain(res)
	Expect(res).NotTo(BeNil())
	Expect(res.StatusCode).To(Equal(http.StatusCreated))

	err = json.NewDecoder(res.Body).Decode(et)
	if err != nil {
		Fail("could not decode response body: " + err.Error())
	}

	matchEventType1(et)
}

func listEventTypes() {
	res, err := eventClient.Get("/eventtypes", nil, nil)
	Expect(err).To(BeNil())
	defer httpclient.Drain(res)
	Expect(res.StatusCode).Should(Equal(http.StatusOK))

	ret := map[string][]interface{}{
		"eventTypes": make([]interface{}, 0),
	}
	err = json.NewDecoder(res.Body).Decode(&ret)
	Expect(err).To(BeNil())
	Expect(ret["eventTypes"]).NotTo(BeNil())

	ets, ok := ret["eventTypes"]
	Expect(ok).To(BeTrue())

	Expect(len(ets)).To(BeNumerically(">=", 3))
}

func getEventTypeById(et *v1.EventType) {
	res, err := eventClient.Get("/eventtypes/main.eventtype2", nil, nil)
	Expect(err).To(BeNil())
	defer httpclient.Drain(res)
	Expect(res.StatusCode).Should(Equal(http.StatusOK))

	err = json.NewDecoder(res.Body).Decode(et)
	if err != nil {
		Fail("could not decode response body: " + err.Error())
	}

	matchEventType1(et)
}

func deleteEventTypeError428(et *v1.EventType) {
	res, err := eventClient.Delete(fmt.Sprintf("/eventtypes/%s", et.Id), http.Header{}, nil)
	Expect(err).To(BeNil())
	defer httpclient.Drain(res)
	Expect(res).NotTo(BeNil())
	Expect(res.StatusCode).To(Equal(http.StatusPreconditionRequired))
}

func deleteEventTypeError404(et *v1.EventType) {
	header := http.Header{}
	header.Add(apis.IfMatch, strconv.FormatInt(math.MaxInt, 10))
	res, err := eventClient.Delete(fmt.Sprintf("/eventtypes/%s", et.Id), header, nil)
	Expect(err).To(BeNil())
	defer httpclient.Drain(res)
	Expect(res).NotTo(BeNil())
	Expect(res.StatusCode).To(Equal(http.StatusNotFound))
}

func deleteEventTypeError412(et *v1.EventType) {
	header := http.Header{}
	header.Add(apis.IfMatch, strconv.FormatInt(math.MaxInt, 10))
	res, err := eventClient.Delete(fmt.Sprintf("/eventtypes/%s", et.Id), header, nil)
	Expect(err).To(BeNil())
	defer httpclient.Drain(res)
	Expect(res).NotTo(BeNil())
	Expect(res.StatusCode).To(Equal(http.StatusPreconditionFailed))
}

func deleteEventTypeError400() {
	header := http.Header{}
	header.Add(apis.IfMatch, strconv.FormatInt(math.MaxInt, 10))
	res, err := eventClient.Delete(fmt.Sprintf("/eventtypes/%s", runtime.StandardEventTypeId), header, nil)
	Expect(err).To(BeNil())
	defer httpclient.Drain(res)
	Expect(res).NotTo(BeNil())
	Expect(res.StatusCode).To(Equal(http.StatusBadRequest))
	//todo check code
}

func deleteEventType(et *v1.EventType) {
	header := http.Header{}
	header.Add(apis.IfMatch, strconv.FormatInt(0, 10))
	res, err := eventClient.Delete(fmt.Sprintf("/eventtypes/%s", et.Id), header, nil)
	Expect(err).To(BeNil())
	defer httpclient.Drain(res)
	Expect(res).NotTo(BeNil())
	Expect(res.StatusCode).To(Equal(http.StatusOK))
}
