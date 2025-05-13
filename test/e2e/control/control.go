package control

import (
	"bytes"
	"encoding/json"
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
	"lightiot/pkg/apis"
	"lightiot/pkg/client"
	"lightiot/pkg/control/runtime"
	model "lightiot/pkg/model/runtime"
	"lightiot/test/e2e/common"
	v1 "lightiot/test/e2e/v1"
	"lightiot/test/util/httpclient"
	"net/http"
	"net/url"
	"text/template"
)

const (
	ThingTypeID  = "thingTypeID"
	CommandTypes = "commandTypes"
)

var (
	controlName       = "e2e-设置温湿度"
	thingTypeName     = "e2e-test-control"
	psName            = "传感器属性集"
	parentThingTypeID = "main.BaseAgent"
	thingType         *model.ThingType
	controlClient     httpclient.TestClient
)

var _ = Describe("Control", Ordered, func() {
	BeforeAll(func() {
		var err error
		controlClient = common.CreateControlClient()
		data := map[string]string{"ThingTypeName": thingTypeName, "PsName": psName, "ParentThingTypeID": parentThingTypeID}
		thingType, err = common.CreateThingType(data)
		Expect(err).To(BeNil())
		Expect(thingType).ToNot(BeNil())
	})
	Context("Post command", func() {})

	Context("CommandType crud", func() {
		var ct runtime.CommandType
		Context("input right parameters", func() {
			It("should create a commandType", func() { createCommandType(&ct) })

			It("should list commandType", func() { listCommandTypes() })

			It("should get commandType by ID", func() { getCommandTypeByID(&ct) })

			It("should delete commandType", func() {
				deleteCommandType(&ct)
				canNotGetCommandTypeByID(ct.ID)
			})
		})
	})
})

func createCommandType(ct *runtime.CommandType) {
	commandTypeTempl, _ := template.New("control").Parse(v1.CreateCommandTypeTemplate)
	b := new(bytes.Buffer)
	_ = commandTypeTempl.Execute(b, map[string]string{"Name": controlName, "OptionName": controlName})
	query := map[string]interface{}{ThingTypeID: thingType.ID}
	res, err := controlClient.Post("/commandTypes", http.Header{}, query, b.String())
	defer client.Drain(res)
	Expect(err).To(BeNil())
	Expect(res).ToNot(BeNil())
	Expect(res.StatusCode).Should(Equal(http.StatusCreated))

	err = json.NewDecoder(res.Body).Decode(ct)
	if err != nil {
		Fail("can't decode response body:" + err.Error())
	}
	Expect(err).To(BeNil())
	Expect(ct.Name).Should(Equal(controlName))
	Expect(len(ct.Options)).Should(BeNumerically("==", 1))
	Expect(ct.Options[0].Name).Should(Equal(controlName))
	Expect(ct.Options[0].Required).Should(Equal(true))
	Expect(ct.Options[0].Filterable).Should(Equal(true))
	Expect(ct.Options[0].Default).Should(Equal("25"))
	Expect(ct.Options[0].Min).Should(gstruct.PointTo(Equal(float64(18))))
	Expect(ct.Options[0].Max).Should(gstruct.PointTo(Equal(float64(32))))
}

func listCommandTypes() {
	query := url.Values{
		ThingTypeID: []string{thingType.ID},
	}
	res, err := controlClient.Get("/commandTypes", nil, &query)
	defer client.Drain(res)
	Expect(err).To(BeNil())
	Expect(res.StatusCode).Should(Equal(http.StatusOK))

	body := map[string][]interface{}{
		CommandTypes: make([]interface{}, 0),
	}
	err = json.NewDecoder(res.Body).Decode(&body)
	Expect(err).To(BeNil())
	Expect(body[CommandTypes]).NotTo(BeNil())
	b, ok := body[CommandTypes]
	Expect(ok).To(BeTrue())
	Expect(len(b)).Should(BeNumerically(">=", 1))
}
func getCommandTypeByID(ct *runtime.CommandType) {
	query := url.Values{
		ThingTypeID: []string{thingType.ID},
	}
	res, err := controlClient.Get(fmt.Sprintf("/commandTypes/%s", ct.ID), nil, &query)
	defer client.Drain(res)
	Expect(err).To(BeNil())
	Expect(res).ToNot(BeNil())
	Expect(res.StatusCode).Should(Equal(http.StatusOK))

	c := runtime.CommandType{}
	err = json.NewDecoder(res.Body).Decode(&c)
	Expect(err).To(BeNil())
	Expect(c.ID).Should(Equal(ct.ID))
	Expect(c.Name).Should(Equal(ct.Name))
	Expect(c.ThingTypeId).Should(Equal(thingType.ID))
}

func deleteCommandType(ct *runtime.CommandType) {
	query := &url.Values{
		ThingTypeID: []string{thingType.ID},
	}
	header := http.Header{}
	header.Add(apis.IfMatch, ct.Version)
	path := fmt.Sprintf("/commandTypes/%s", ct.ID)
	res, err := controlClient.Delete(path, header, query)
	defer client.Drain(res)
	Expect(err).To(BeNil())
	Expect(res.StatusCode).Should(Equal(http.StatusOK))
}

func canNotGetCommandTypeByID(ID string) {
	query := url.Values{
		ThingTypeID: []string{thingType.ID},
	}
	res, err := controlClient.Get(fmt.Sprintf("/commandTypes/%s", ID), nil, &query)
	defer client.Drain(res)
	Expect(err).To(BeNil())
	Expect(res.StatusCode).Should(Equal(http.StatusNotFound))
}
