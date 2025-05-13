package template

import (
	"bytes"
	"encoding/json"
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"lightiot/pkg/apis"
	"lightiot/pkg/client"
	"lightiot/pkg/notification/runtime"
	p "lightiot/pkg/notification/v1"
	"lightiot/test/e2e/common"
	v1 "lightiot/test/e2e/v1"
	"lightiot/test/util/httpclient"
	"net/http"
	"net/url"
	"strconv"
	"text/template"
)

var (
	mtContent          = `尊敬的{{name}}， 您好。 \\n\\t你的{{meetingRoom}}温度超过{{expectedValue}}，已到{{actualValue}}。\\n\\t详情请见：<a href={{actualUrl}}>更多信息</a>`
	notificationClient httpclient.TestClient
)

var _ = Describe("Template", Ordered, func() {
	var t runtime.Template
	var r runtime.Recipient
	var mt runtime.MessageTemplate
	var recipientName = "e2e-二仙桥会议室服务部"
	var messageTemplateName = "e2e-温度超过 30 摄氏度消息模板"
	var templateName = "e2e-温度超过 30 摄氏度消息模板"
	BeforeAll(func() {
		notificationClient = common.CreateNotificationClient()
		createRecipient(&r, recipientName)
		createMessageTemplate(&mt, messageTemplateName, mtContent)
	})

	AfterAll(func() {
		deleteRecipient(&r, true, http.StatusOK)
		deleteMessageTemplate(&mt, true, http.StatusOK)
	})

	Context("Template crud", func() {
		When("input right parameters", func() {
			It("should create template", func() { createTemplate(&t, templateName, []string{r.ID}, []string{mt.ID}) })

			It("should list template", func() { listTemplates() })

			It("should get template by ID", func() { getTemplateByID(t.ID, &t) })

			It("should delete template", func() {
				deleteTemplate(&t)
				canNotGetTemplate(t.ID)
			})
		})
	})
})

var _ = Describe("Template orphanRemoval", Ordered, func() {
	var readyRecipient runtime.Recipient
	var readyMessageTemplate runtime.MessageTemplate
	var recipientName = "e2e-兴隆湖会议室服务部"
	var messageTemplateName = "e2e-温度超过 35 摄氏度消息模板"
	BeforeAll(func() {
		notificationClient = common.CreateNotificationClient()
	})
	BeforeEach(func() {
		createRecipient(&readyRecipient, recipientName+strconv.Itoa(GinkgoParallelProcess()))
		createMessageTemplate(&readyMessageTemplate, messageTemplateName+strconv.Itoa(GinkgoParallelProcess()), mtContent)
	})
	Context("when delete messageTemplate", func() {

		It("should delete messageTemplate and update the template which remove the delete messageTemplate", func() {
			var t runtime.Template
			var retainMessageTemplate runtime.MessageTemplate
			var deletedMessageTemplate = "e2e-温度超过 70 摄氏度消息模板"
			var deletedTemplateName = "e2e-温度超过 70 摄氏度消息模板"
			createMessageTemplate(&retainMessageTemplate, deletedMessageTemplate+strconv.Itoa(GinkgoParallelProcess()), mtContent)
			createTemplate(&t, deletedTemplateName+strconv.Itoa(GinkgoParallelProcess()), []string{readyRecipient.ID}, []string{readyMessageTemplate.ID, retainMessageTemplate.ID})
			deleteMessageTemplate(&readyMessageTemplate, true, http.StatusOK)
			updateTemplate := getUpdateTemplate(t.ID, readyRecipient.ID, retainMessageTemplate.ID)
			deleteTemplate(updateTemplate)
			deleteRecipient(&readyRecipient, true, http.StatusOK)
			deleteMessageTemplate(&retainMessageTemplate, true, http.StatusOK)
		})
		It("should delete messageTemplate and delete the template which has the only messageTemplate", func() {
			var t runtime.Template
			var deletedTemplateName = "e2e-温度超过 40 摄氏度消息模板"
			createTemplate(&t, deletedTemplateName+strconv.Itoa(GinkgoParallelProcess()), []string{readyRecipient.ID}, []string{readyMessageTemplate.ID})
			deleteMessageTemplate(&readyMessageTemplate, true, http.StatusOK)
			canNotGetTemplate(t.ID)
			deleteRecipient(&readyRecipient, true, http.StatusOK)
		})
	})

	Context("when delete recipient", func() {
		It("should delete recipient and update the template which remove the delete recipient", func() {
			var t runtime.Template
			var retainRecipient runtime.Recipient
			var deletedRecipientName = "e2e-锦城湖会议室服务部"
			var deletedTemplateName = "e2e-温度超过 40 摄氏度消息模板"
			createRecipient(&retainRecipient, deletedRecipientName+strconv.Itoa(GinkgoParallelProcess()))
			createTemplate(&t, deletedTemplateName+strconv.Itoa(GinkgoParallelProcess()), []string{readyRecipient.ID, retainRecipient.ID}, []string{readyMessageTemplate.ID})
			deleteRecipient(&readyRecipient, true, http.StatusOK)
			updateTemplate := getUpdateTemplate(t.ID, retainRecipient.ID, readyMessageTemplate.ID)
			deleteTemplate(updateTemplate)
			deleteRecipient(&retainRecipient, true, http.StatusOK)
			deleteMessageTemplate(&readyMessageTemplate, true, http.StatusOK)
		})
		It("should delete recipient and delete the template which has the only recipientID", func() {
			var t runtime.Template
			var deletedTemplateName = "e2e-温度超过 40 摄氏度消息模板"
			createTemplate(&t, deletedTemplateName+strconv.Itoa(GinkgoParallelProcess()), []string{readyRecipient.ID}, []string{readyMessageTemplate.ID})
			deleteRecipient(&readyRecipient, true, http.StatusOK)
			canNotGetTemplate(t.ID)
			deleteMessageTemplate(&readyMessageTemplate, true, http.StatusOK)
		})
	})

	Context("when delete recipient which associate template", func() {
		It("should return 400", func() {
			var t runtime.Template
			var deletedTemplateName = "e2e-温度超过 50 摄氏度消息模板"
			createTemplate(&t, deletedTemplateName+strconv.Itoa(GinkgoParallelProcess()), []string{readyRecipient.ID}, []string{readyMessageTemplate.ID})
			deleteRecipient(&readyRecipient, false, http.StatusBadRequest)
			deleteTemplate(&t)
			deleteRecipient(&readyRecipient, true, http.StatusOK)
			deleteMessageTemplate(&readyMessageTemplate, true, http.StatusOK)
		})
	})

	Context("when delete messageTemplate which associate template", func() {
		It("should return 400", func() {
			var t runtime.Template
			var deletedTemplateName = "e2e-温度超过 55 摄氏度消息模板"
			createTemplate(&t, deletedTemplateName+strconv.Itoa(GinkgoParallelProcess()), []string{readyRecipient.ID}, []string{readyMessageTemplate.ID})
			deleteMessageTemplate(&readyMessageTemplate, false, http.StatusBadRequest)
			deleteTemplate(&t)
			deleteRecipient(&readyRecipient, true, http.StatusOK)
			deleteMessageTemplate(&readyMessageTemplate, true, http.StatusOK)
		})
	})
})

func createRecipient(r *runtime.Recipient, name string) {
	recipientTempl, _ := template.New("recipient").Parse(v1.CreateRecipientTemplate)
	b := new(bytes.Buffer)
	_ = recipientTempl.Execute(b, map[string]string{"Name": name})
	res, err := notificationClient.Post("/recipients", http.Header{}, make(map[string]interface{}, 0), b.String())
	defer client.Drain(res)

	Expect(err).To(BeNil())
	Expect(res).ToNot(BeNil())
	Expect(res.StatusCode).Should(Equal(http.StatusCreated))
	err = json.NewDecoder(res.Body).Decode(r)
	Expect(err).To(BeNil())
}

func createMessageTemplate(mt *runtime.MessageTemplate, mtName, mtContent string) {
	messageTemplate, _ := template.New("messageTemplate").Parse(v1.CreateMessageTemplate)
	b := new(bytes.Buffer)
	_ = messageTemplate.Execute(b, map[string]string{"Name": mtName, "Content": mtContent})
	res, err := notificationClient.Post("/messageTemplates", http.Header{}, nil, b.String())
	defer client.Drain(res)
	Expect(err).To(BeNil())
	Expect(res).ToNot(BeNil())
	Expect(res.StatusCode).Should(Equal(http.StatusCreated))

	err = json.NewDecoder(res.Body).Decode(mt)
	Expect(err).To(BeNil())
}

func createTemplate(t *runtime.Template, name string, recipientID []string, messageID []string) {
	tp := p.Template{
		Name:             name,
		Subject:          "【警告】室温过高",
		From:             "Afflatus",
		MessageTemplates: messageID,
	}
	tps := make([]p.TmplRecipient, 0, len(recipientID))
	for _, ID := range recipientID {
		tps = append(tps, p.TmplRecipient{Id: ID})
	}
	tp.Recipients = tps
	tpValue, _ := json.Marshal(tp)
	res, err := notificationClient.Post("/templates", http.Header{}, make(map[string]interface{}, 0), string(tpValue))
	defer client.Drain(res)

	Expect(err).To(BeNil())
	Expect(res).ToNot(BeNil())
	Expect(res.StatusCode).Should(Equal(http.StatusCreated))
	err = json.NewDecoder(res.Body).Decode(t)
	Expect(err).To(BeNil())
	Expect(t.Name).Should(Equal(name))
	Expect(len(t.Recipients)).Should(BeNumerically("==", len(recipientID)))
	Expect(t.Recipients[0].Id).Should(Equal(recipientID[0]))
	Expect(len(t.MessageTemplates)).Should(BeNumerically("==", len(messageID)))
	Expect(t.MessageTemplates[0]).Should(Equal(messageID[0]))
}

func listTemplates() {
	res, err := notificationClient.Get("/templates", nil, nil)
	defer client.Drain(res)

	Expect(err).To(BeNil())
	Expect(res.StatusCode).Should(Equal(http.StatusOK))
	body := map[string][]interface{}{
		"templates": make([]interface{}, 0),
	}
	err = json.NewDecoder(res.Body).Decode(&body)
	Expect(err).To(BeNil())
	Expect(body["templates"]).NotTo(BeNil())
	b, ok := body["templates"]
	Expect(ok).To(BeTrue())
	Expect(len(b)).Should(BeNumerically(">=", 1))
}

func getTemplateByID(tID string, targetTemplate *runtime.Template) {
	res, err := notificationClient.Get(fmt.Sprintf("/templates/%s", tID), nil, nil)
	defer client.Drain(res)

	Expect(err).To(BeNil())
	Expect(res).ToNot(BeNil())
	Expect(res.StatusCode).Should(Equal(http.StatusOK))
	rt := runtime.Template{}
	err = json.NewDecoder(res.Body).Decode(&rt)
	Expect(err).To(BeNil())
	Expect(rt.ID).Should(Equal(targetTemplate.ID))
	Expect(len(rt.Recipients)).Should(BeNumerically("==", 1))
	Expect(rt.Recipients[0].Id).Should(Equal(targetTemplate.Recipients[0].Id))
	Expect(len(rt.MessageTemplates)).Should(BeNumerically("==", 1))
	Expect(rt.MessageTemplates[0]).Should(Equal(targetTemplate.MessageTemplates[0]))
}

func getUpdateTemplate(ID string, retainRecipientID, retainMessageTemplateID string) (dr *runtime.Template) {
	res, err := notificationClient.Get(fmt.Sprintf("/templates/%s", ID), nil, nil)
	defer client.Drain(res)

	Expect(err).To(BeNil())
	Expect(res).ToNot(BeNil())
	Expect(res.StatusCode).Should(Equal(http.StatusOK))
	err = json.NewDecoder(res.Body).Decode(&dr)
	Expect(err).To(BeNil())
	Expect(dr.ID).Should(Equal(ID))
	Expect(len(dr.Recipients)).Should(BeNumerically("==", 1))
	Expect(dr.Recipients[0].Id).Should(Equal(retainRecipientID))
	Expect(len(dr.MessageTemplates)).Should(BeNumerically("==", 1))
	Expect(dr.MessageTemplates[0]).Should(Equal(retainMessageTemplateID))
	return
}

func deleteTemplate(t *runtime.Template) {
	header := http.Header{}
	header.Add(apis.IfMatch, t.Version)
	res, err := notificationClient.Delete(fmt.Sprintf("/templates/%s", t.ID), header, nil)
	defer client.Drain(res)
	Expect(err).To(BeNil())
	Expect(res.StatusCode).Should(Equal(http.StatusOK))
}

func canNotGetTemplate(ID string) {
	res, err := notificationClient.Get(fmt.Sprintf("/templates/%s", ID), nil, nil)
	defer client.Drain(res)
	Expect(err).To(BeNil())
	Expect(res.StatusCode).Should(Equal(http.StatusNoContent))
}

func deleteRecipient(r *runtime.Recipient, OrphanRemoval bool, code int) {
	var query *url.Values
	if OrphanRemoval {
		query = &url.Values{
			apis.OrphanRemoval: []string{strconv.FormatBool(OrphanRemoval)},
		}
	}
	header := http.Header{}
	header.Add(apis.IfMatch, r.Version)
	res, err := notificationClient.Delete(fmt.Sprintf("/recipients/%s", r.ID), header, query)
	defer client.Drain(res)
	Expect(err).To(BeNil())
	Expect(res.StatusCode).Should(Equal(code))
}

func deleteMessageTemplate(mt *runtime.MessageTemplate, OrphanRemoval bool, code int) {
	var query *url.Values
	if OrphanRemoval {
		query = &url.Values{
			apis.OrphanRemoval: []string{strconv.FormatBool(OrphanRemoval)},
		}
	}
	header := http.Header{}
	header.Add(apis.IfMatch, mt.Version)
	res, err := notificationClient.Delete(fmt.Sprintf("/messageTemplates/%s", mt.ID), header, query)
	defer client.Drain(res)
	Expect(err).To(BeNil())
	Expect(res.StatusCode).Should(Equal(code))
}
