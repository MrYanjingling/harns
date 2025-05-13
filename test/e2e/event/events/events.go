package events

import (
	"encoding/json"
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"lightiot/pkg/apis"
	mr "lightiot/pkg/model/runtime"
	"lightiot/test/e2e/common"
	"lightiot/test/util/httpclient"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const createEventTypeParam = `{
    "name": "eventtype1",
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
	now         = time.Now().UTC().Truncate(time.Millisecond)
	thing       *mr.Thing
	eventValues = map[string]interface{}{
		"typeId": "main.eventtype1",
		"_time":  now,
		"f1":     0,
		"f2":     1,
	}
	eventId string
)

var _ = Describe("Event", Ordered, func() {
	BeforeAll(func() {
		eventClient = common.CreateEventClient()
		_, err := eventClient.Post("/eventtypes", http.Header{}, nil, createEventTypeParam)

		thingTypeData := map[string]string{"ThingTypeName": "e2e-test-event", "PsName": "event", "ParentThingTypeId": "main.BaseAgent"}
		thingType, err := common.CreateThingType(thingTypeData)
		Expect(err).To(BeNil())
		Expect(thingType).NotTo(BeNil())

		thingData := map[string]string{"Name": "e2e-test-event", "ThingTypeId": thingType.ID}
		thing, err = common.CreateThing(thingData)
		Expect(err).To(BeNil())
		Expect(thing).NotTo(BeNil())
		eventValues["thingId"] = thing.ID
	})

	Context("Event create list and read", func() {
		When("input right parameters", func() {
			It("should create an Event", func() { createEvent() })

			It("should wait 5s for influxdb to flush", func() { time.Sleep(5 * time.Second) })

			It("should list Events", func() { listEvents() })

			It("should get an Event by id", func() { getEvent() })

			It("should delete an Event error and http response code 428", func() { deleteEventError428() })

			It("should delete an Event error and http response code 404", func() { deleteEventError404() })

			It("should delete an Event error and http response code 412", func() { deleteEventError412() })

			It("should delete an Event success", func() { deleteEvent() })
		})
	})

	AfterAll(func() {
		// TODO delete
	})
})

func createEvent() {
	b, _ := json.Marshal(eventValues)
	res, err := eventClient.Post("/events", http.Header{}, nil, string(b))
	Expect(err).To(BeNil())
	defer httpclient.Drain(res)
	Expect(res).NotTo(BeNil())
	Expect(res.StatusCode).To(Equal(http.StatusAccepted))
}

func matchEvents(es []map[string]interface{}) {
	Expect(len(es)).To(Equal(1))

	e := es[0]

	typeId, ok := e["typeId"]
	Expect(ok).To(BeTrue())
	Expect(typeId.(string)).To(Equal(eventValues["typeId"]))

	ts, ok := e["_time"]
	Expect(ok).To(BeTrue())
	t, err := time.Parse(time.RFC3339Nano, ts.(string))
	Expect(err).To(BeNil())
	Expect(t).To(Equal(eventValues["_time"]))

	thingId, ok := e["thingId"]
	Expect(ok).To(BeTrue())
	Expect(thingId.(string)).To(Equal(eventValues["thingId"]))

	f1, ok := e["f1"]
	Expect(ok).To(BeTrue())
	Expect(f1.(string)).To(Equal("v1"))

	f2f, ok := e["f2"]
	Expect(ok).To(BeTrue())
	Expect(int(f2f.(float64))).To(Equal(eventValues["f2"]))

	id, ok := e["id"]
	Expect(ok).To(BeTrue())
	eventId, ok = id.(string)
	Expect(ok).To(BeTrue())
}

func listEvents() {
	query := &url.Values{
		"start":  []string{now.Format(time.RFC3339Nano)},
		"end":    []string{time.Now().UTC().Format(time.RFC3339Nano)},
		"filter": []string{`{"typeId":"main.eventtype1"}`},
	}
	res, err := eventClient.Get("/events", http.Header{}, query)
	Expect(err).To(BeNil())
	defer httpclient.Drain(res)
	Expect(res).NotTo(BeNil())
	Expect(res.StatusCode).To(Equal(http.StatusOK))

	ret := struct {
		Events []map[string]interface{} `json:"events"`
	}{}

	err = json.NewDecoder(res.Body).Decode(&ret)
	if err != nil {
		Fail("could not decode response body: " + err.Error())
	}

	matchEvents(ret.Events)
}

func getEvent() {
	query := &url.Values{
		"typeId": []string{"main.eventtype1"},
	}
	res, err := eventClient.Get(fmt.Sprintf("/events/%s", eventId), http.Header{}, query)
	Expect(err).To(BeNil())
	defer httpclient.Drain(res)
	Expect(res).NotTo(BeNil())
	Expect(res.StatusCode).To(Equal(http.StatusOK))

	var ret map[string]interface{}

	err = json.NewDecoder(res.Body).Decode(&ret)
	if err != nil {
		Fail("could not decode response body: " + err.Error())
	}

	matchEvents([]map[string]interface{}{ret})
}

func deleteEventError428() {
	query := &url.Values{
		"typeId": []string{"main.eventtype1"},
	}
	res, err := eventClient.Delete(fmt.Sprintf("/events/%s", eventId), http.Header{}, query)
	Expect(err).To(BeNil())
	defer httpclient.Drain(res)
	Expect(res).NotTo(BeNil())
	Expect(res.StatusCode).To(Equal(http.StatusPreconditionRequired))
}

func deleteEventError404() {
	query := &url.Values{
		"typeId": []string{"main.eventtype1"},
	}
	header := http.Header{}
	header.Add(apis.IfMatch, strconv.FormatInt(math.MaxInt, 10))
	notFoundEvent := eventId + "s"
	res, err := eventClient.Delete(fmt.Sprintf("/events/%s", notFoundEvent), header, query)
	Expect(err).To(BeNil())
	defer httpclient.Drain(res)
	Expect(res).NotTo(BeNil())
	Expect(res.StatusCode).To(Equal(http.StatusNotFound))
}

func deleteEventError412() {
	query := &url.Values{
		"typeId": []string{"main.eventtype1"},
	}
	header := http.Header{}
	header.Add(apis.IfMatch, strconv.FormatInt(math.MaxInt, 10))
	res, err := eventClient.Delete(fmt.Sprintf("/events/%s", eventId), header, query)
	Expect(err).To(BeNil())
	defer httpclient.Drain(res)
	Expect(res).NotTo(BeNil())
	Expect(res.StatusCode).To(Equal(http.StatusPreconditionFailed))
}

func deleteEvent() {
	query := &url.Values{
		"typeId": []string{"main.eventtype1"},
	}
	header := http.Header{}
	header.Add(apis.IfMatch, strconv.FormatInt(0, 10))
	res, err := eventClient.Delete(fmt.Sprintf("/events/%s", eventId), header, query)
	Expect(err).To(BeNil())
	defer httpclient.Drain(res)
	Expect(res).NotTo(BeNil())
	Expect(res.StatusCode).To(Equal(http.StatusOK))
}
