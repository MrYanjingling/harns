package wechat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"k8s.io/klog/v2"
	"lightiot/pkg/client"
	"lightiot/pkg/notification/config"
	"lightiot/pkg/notification/notify"
	"lightiot/pkg/notification/runtime"
	"lightiot/pkg/util/aesutil"
	"lightiot/pkg/util/security"
	"net/http"
	"net/url"
	"time"
)

type WeChat struct {
	// TODO it violates principle of least privilege
	configMgr   *config.Manager
	client      client.Client
	accessToken map[string]token
}

// token is the AccessToken with corpid and corpsecret.
type token struct {
	AccessToken string `json:"access_token"`
	at          time.Time
}

type weChatMessage struct {
	Text   weChatMessageContent `yaml:"text,omitempty" json:"text,omitempty"`
	ToUser string               `yaml:"touser,omitempty" json:"touser,omitempty"`
	Type   string               `yaml:"msgtype,omitempty" json:"msgtype,omitempty"`
}

type weChatMessageContent struct {
	Content string `json:"content"`
}

type weChatResponse struct {
	Code  int    `json:"errcode"`
	Error string `json:"errmsg"`
}

var _ notify.Notifier = new(WeChat)

func New(configMgr *config.Manager, weChatClient client.Client) notify.Notifier {
	return &WeChat{
		configMgr:   configMgr,
		client:      weChatClient,
		accessToken: map[string]token{},
	}
}

func (wc *WeChat) Notify(stopCh <-chan struct{}, msg *runtime.Message) (bool, error) {
	server, err := wc.configMgr.GetConfig()
	if err != nil {
		return false, err
	}
	conf := server.WeChat

	t, ok := wc.accessToken[security.GetTenant()]
	if !ok {
		t = token{
			AccessToken: "",
			at:          time.Time{},
		}
	}

	// Refresh AccessToken over 2 hours
	if t.AccessToken == "" || time.Since(t.at) > 2*time.Hour {
		parameters := url.Values{}
		parameters.Add("grant_type", "client_credential")
		parameters.Add("appid", conf.AppId)
		parameters.Add("secret", string(aesutil.DecryptCBC([]byte(conf.AppSecret), config.GetAesKey(security.GetTenant()))))

		resp, err := wc.client.Get("/cgi-bin/token", http.Header{"Content-Type": []string{"application/json"}}, &parameters)
		if err != nil {
			klog.V(1).InfoS("Failed to access wechat", "err", err)
			return true, err
		}
		defer resp.Body.Close()
		var wechatToken token
		if err := json.NewDecoder(resp.Body).Decode(&wechatToken); err != nil {
			return false, err
		}

		if wechatToken.AccessToken == "" {
			return false, fmt.Errorf("invalid APPSecret for APPId: %s", conf.AppId)
		}

		// Cache accessToken
		t.AccessToken = wechatToken.AccessToken
		t.at = time.Now()
	}

	weChatMsg := &weChatMessage{}

	for _, msgTmpl := range msg.MsgTmpls {
		if msgTmpl.ParsedTmpl.TextTmpl != nil {
			weChatMsg.Type = msgTmpl.Type.String()
			var buf bytes.Buffer
			err = msgTmpl.ParsedTmpl.TextTmpl.Execute(&buf, msg)
			if err != nil {
				return false, errors.Wrap(err, "execute text template")
			}
			if weChatMsg.Type == "text" {
				weChatMsg.Text = weChatMessageContent{
					Content: buf.String(),
				}
			}

			for _, user := range msg.To {
				weChatMsg.ToUser = user
				resp, err := wc.client.Post("/cgi-bin/message/custom/send", http.Header{"Content-Type": []string{"application/json"}}, map[string]interface{}{"access_token": t.AccessToken}, weChatMsg)
				if err != nil {
					klog.V(1).InfoS("Failed to access wechat", "err", err)
					return true, err
				}
				if resp.StatusCode != 200 {
					klog.V(2).InfoS("Received HTTP response", "code", resp.StatusCode)
					return true, fmt.Errorf("unexpected status code %v", resp.StatusCode)
				}

				body, err := io.ReadAll(resp.Body)
				// TODO defer
				resp.Body.Close()
				if err != nil {
					return true, err
				}
				klog.V(5).InfoS("Received wechat response", "response", string(body))

				var weResp weChatResponse
				if err := json.Unmarshal(body, &weResp); err != nil {
					return true, err
				}

				// https://developers.weixin.qq.com/doc/offiaccount/Getting_Started/Global_Return_Code.html
				if weResp.Code == 0 {
					continue
				}

				// AccessToken is expired
				if weResp.Code == 42001 {
					t.AccessToken = ""
					return true, errors.New(weResp.Error)
				}
			}
		}
	}

	return false, nil
}
