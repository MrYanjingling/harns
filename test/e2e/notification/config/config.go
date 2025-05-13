package config

import (
	"encoding/json"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"lightiot/pkg/apis"
	"lightiot/pkg/client"
	"lightiot/pkg/notification/runtime"
	v12 "lightiot/pkg/notification/v1"
	"lightiot/test/e2e/common"
	v1 "lightiot/test/e2e/v1"
	"lightiot/test/util/httpclient"
	"net/http"
)

var (
	notificationClient httpclient.TestClient
)

var _ = Describe("Config", Ordered, func() {

	BeforeAll(func() {
		notificationClient = common.CreateNotificationClient()
	})

	Context("Server config crud", func() {
		var c runtime.Config
		When("input right parameters", func() {
			It("should get server config", func() { getServerConfig(&c) })

			// todo should after create
			// It("should update server config", func() {
			//	updateServerConfig(&c)
			//	getUpdateServerConfig()
			// })
		})
	})
})

func getServerConfig(c *runtime.Config) {
	res, err := notificationClient.Get("/serverConfigurations", http.Header{}, nil)
	defer client.Drain(res)
	Expect(err).To(BeNil())
	Expect(res).ToNot(BeNil())
	Expect(res.StatusCode).Should(Equal(http.StatusOK))

	err = json.NewDecoder(res.Body).Decode(&c)
	if err != nil {
		Fail("can't decode response body:" + err.Error())
	}
	Expect(err).To(BeNil())
	Expect(c.Email.Smtp.Hostname).Should(Equal("mail.example.com"))
	Expect(c.Email.Smtp.Port).Should(Equal("587"))
	Expect(c.Email.Smtp.Username).Should(Equal("update@example.com"))
	Expect(c.Email.Smtp.Password).Should(Equal(v12.Secret("<secret>")))
	Expect(c.WeCom.CorpId).Should(Equal("xxxxxxxx"))
	Expect(c.WeCom.CorpSecret).Should(Equal(v12.Secret("<secret>")))
	Expect(c.WeCom.AgentId).Should(Equal("3"))
	Expect(c.WeChat.AppSecret).Should(Equal(v12.Secret("<secret>")))
	Expect(c.WeChat.AppId).Should(Equal("xxxxxx"))
}

func updateServerConfig(c *runtime.Config) {
	header := http.Header{}
	header.Add(apis.IfMatch, c.Version)
	res, err := notificationClient.Put("/serverConfigurations", header, v1.UpdateServerConfigTemplate)
	defer client.Drain(res)
	Expect(err).To(BeNil())
	Expect(res).ToNot(BeNil())
	Expect(res.StatusCode).Should(Equal(http.StatusOK))

	err = json.NewDecoder(res.Body).Decode(&c)
	if err != nil {
		Fail("can't decode response body:" + err.Error())
	}
	Expect(err).To(BeNil())
	Expect(c.Email.Smtp.Username).Should(Equal("update@example.com"))
	Expect(c.WeCom.AgentId).Should(Equal("3"))
}

func getUpdateServerConfig() {
	var c runtime.Config
	res, err := notificationClient.Get("/serverConfigurations", http.Header{}, nil)
	defer client.Drain(res)
	Expect(err).To(BeNil())
	Expect(res).ToNot(BeNil())
	Expect(res.StatusCode).Should(Equal(http.StatusOK))

	err = json.NewDecoder(res.Body).Decode(&c)
	if err != nil {
		Fail("can't decode response body:" + err.Error())
	}
	Expect(err).To(BeNil())
	Expect(c.Email.Smtp.Hostname).Should(Equal("mail.example.com"))
	Expect(c.Email.Smtp.Port).Should(Equal("587"))
	Expect(c.Email.Smtp.Username).Should(Equal("update@example.com"))
	Expect(c.Email.Smtp.Password).Should(Equal(v12.Secret("<secret>")))
	Expect(c.WeCom.CorpId).Should(Equal("xxxxxxxx"))
	Expect(c.WeCom.CorpSecret).Should(Equal(v12.Secret("<secret>")))
	Expect(c.WeCom.AgentId).Should(Equal("3"))
	Expect(c.WeChat.AppSecret).Should(Equal(v12.Secret("<secret>")))
	Expect(c.WeChat.AppId).Should(Equal("xxxxxx"))
}
