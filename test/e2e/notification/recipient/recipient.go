package recipient

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
	recipientName      = "峨眉山会议室服务部"
	notificationClient httpclient.TestClient
)

var _ = Describe("Recipient", Ordered, func() {

	BeforeAll(func() {
		notificationClient = common.CreateNotificationClient()
	})

	Context("Recipient crud", func() {
		var r runtime.Recipient
		When("input right parameters", func() {
			It("should create recipient", func() { createRecipient(&r) })

			It("should list recipients", func() { listRecipients() })

			It("should get recipient by id", func() { getRecipientById(&r) })

			It("should delete recipient", func() {
				deleteRecipient(&r)
				canNotGetRecipient(r.ID)
			})
		})
	})
})

func createRecipient(r *runtime.Recipient) {
	recipientTempl, _ := template.New("recipient").Parse(v1.CreateRecipientTemplate)
	b := new(bytes.Buffer)
	_ = recipientTempl.Execute(b, map[string]string{"Name": recipientName})
	res, err := notificationClient.Post("/recipients", http.Header{}, make(map[string]interface{}, 0), b.String())
	defer client.Drain(res)

	Expect(err).To(BeNil())
	Expect(res).ToNot(BeNil())
	Expect(res.StatusCode).Should(Equal(http.StatusCreated))
	err = json.NewDecoder(res.Body).Decode(r)
	Expect(err).To(BeNil())

	Expect(r.Name).Should(Equal(recipientName))
	Expect(len(r.Detail)).Should(BeNumerically("==", 3))
}

func listRecipients() {
	res, err := notificationClient.Get("/recipients", nil, nil)
	defer client.Drain(res)

	Expect(err).To(BeNil())
	Expect(res.StatusCode).Should(Equal(http.StatusOK))
	body := map[string][]interface{}{
		"recipients": make([]interface{}, 0),
	}
	err = json.NewDecoder(res.Body).Decode(&body)
	Expect(err).To(BeNil())
	Expect(body["recipients"]).NotTo(BeNil())
	b, ok := body["recipients"]
	Expect(ok).To(BeTrue())
	Expect(len(b)).Should(BeNumerically(">=", 1))
}

func getRecipientById(r *runtime.Recipient) {
	res, err := notificationClient.Get(fmt.Sprintf("/recipients/%s", r.ID), nil, nil)
	defer client.Drain(res)

	Expect(err).To(BeNil())
	Expect(res).ToNot(BeNil())
	Expect(res.StatusCode).Should(Equal(http.StatusOK))
	rr := runtime.Recipient{}
	err = json.NewDecoder(res.Body).Decode(&rr)
	Expect(err).To(BeNil())
	Expect(rr.ID).Should(Equal(r.ID))
	Expect(len(rr.Detail)).Should(BeNumerically("==", 3))
}

func deleteRecipient(r *runtime.Recipient) {
	header := http.Header{}
	header.Add(apis.IfMatch, r.Version)
	res, err := notificationClient.Delete(fmt.Sprintf("/recipients/%s", r.ID), header, nil)
	defer client.Drain(res)
	Expect(err).To(BeNil())
	Expect(res.StatusCode).Should(Equal(http.StatusOK))
}

func canNotGetRecipient(id string) {
	res, err := notificationClient.Get(fmt.Sprintf("/recipients/%s", id), nil, nil)
	defer client.Drain(res)
	Expect(err).To(BeNil())
	Expect(res.StatusCode).Should(Equal(http.StatusNoContent))
}
