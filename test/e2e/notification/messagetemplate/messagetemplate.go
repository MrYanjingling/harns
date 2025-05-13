package messagetemplate

import (
	"bytes"
	"encoding/json"
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"lightiot/pkg/apis"
	"lightiot/pkg/client"
	"lightiot/pkg/notification/runtime"
	"lightiot/test/e2e/common"
	v1 "lightiot/test/e2e/v1"
	"lightiot/test/util/httpclient"
	"net/http"
	"text/template"
)

var (
	messageTemplateName    = "e2e-温度超过 25 摄氏度消息模板-1"
	messageTemplateContent = `尊敬的{{name}}， 您好。 \\n\\t你的{{meetingRoom}}温度超过{{expectedValue}}，已到{{actualValue}}。\\n\\t详情请见：<a href={{actualUrl}}>更多信息</a>`
)

var (
	notificationClient httpclient.TestClient
)

var _ = Describe("MessageTemplate", Ordered, func() {
	BeforeAll(func() {
		notificationClient = common.CreateNotificationClient()
	})
	Context("MessageTemplate crud", func() {
		var mt runtime.MessageTemplate
		When("input right parameters", func() {
			It("should create messageTemplate", func() { createMessageTemplate(&mt) })

			It("should list messageTemplates", func() { listMessageTemplate() })

			It("should get messageTemplate by id", func() { getMessageTemplateById(&mt) })

			It("should delete messageTemplate", func() {
				deleteMessageTemplate(&mt)
				canNotGetMessageTemplate(mt.ID)
			})
		})
	})
})

func createMessageTemplate(mt *runtime.MessageTemplate) {
	messageTemplate, _ := template.New("messageTemplate").Parse(v1.CreateMessageTemplate)
	b := new(bytes.Buffer)
	_ = messageTemplate.Execute(b, map[string]string{"Name": messageTemplateName, "Content": messageTemplateContent})
	res, err := notificationClient.Post("/messageTemplates", http.Header{}, nil, b.String())
	defer client.Drain(res)
	Expect(err).To(BeNil())
	Expect(res).ToNot(BeNil())
	Expect(res.StatusCode).Should(Equal(http.StatusCreated))

	err = json.NewDecoder(res.Body).Decode(mt)
	if err != nil {
		Fail("can't decode response body:" + err.Error())
	}
	Expect(err).To(BeNil())
	Expect(mt.Name).Should(Equal(messageTemplateName))
}

func listMessageTemplate() {
	res, err := notificationClient.Get("/messageTemplates", nil, nil)
	defer client.Drain(res)

	Expect(err).To(BeNil())
	Expect(res.StatusCode).Should(Equal(http.StatusOK))
	body := map[string][]interface{}{
		"messageTemplates": make([]interface{}, 0),
	}
	err = json.NewDecoder(res.Body).Decode(&body)
	Expect(err).To(BeNil())
	Expect(body["messageTemplates"]).NotTo(BeNil())
	b, ok := body["messageTemplates"]
	Expect(ok).To(BeTrue())
	Expect(len(b)).Should(BeNumerically(">=", 1))
}

func getMessageTemplateById(mt *runtime.MessageTemplate) {
	res, err := notificationClient.Get(fmt.Sprintf("/messageTemplates/%s", mt.ID), nil, nil)
	defer client.Drain(res)

	Expect(err).To(BeNil())
	Expect(res).ToNot(BeNil())
	Expect(res.StatusCode).Should(Equal(http.StatusOK))
	rmt := runtime.MessageTemplate{}
	err = json.NewDecoder(res.Body).Decode(&rmt)
	Expect(err).To(BeNil())
	Expect(rmt.Name).Should(Equal(mt.Name))
}

func deleteMessageTemplate(mt *runtime.MessageTemplate) {
	header := http.Header{}
	header.Add(apis.IfMatch, mt.Version)
	res, err := notificationClient.Delete(fmt.Sprintf("/messageTemplates/%s", mt.ID), header, nil)
	defer client.Drain(res)
	Expect(err).To(BeNil())
	Expect(res.StatusCode).Should(Equal(http.StatusOK))
}

func canNotGetMessageTemplate(id string) {
	res, err := notificationClient.Get(fmt.Sprintf("/messageTemplates/%s", id), nil, nil)
	defer client.Drain(res)
	Expect(err).To(BeNil())
	Expect(res.StatusCode).Should(Equal(http.StatusNoContent))
}
