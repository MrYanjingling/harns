package wecom

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
	"strings"
	"time"
)

type WeCom struct {
	// TODO it violates principle of least privilege
	configMgr   *config.Manager
	client      client.Client
	accessToken map[string]token
}

//
// type accessToken struct {
//	token string
//	at    time.Time
// }

// token is the AccessToken with corpid and corpsecret.
type token struct {
	AccessToken string `json:"access_token"`
	at          time.Time
}

type weComMessage struct {
	Text     weComMessageContent `yaml:"text,omitempty" json:"text,omitempty"`
	ToUser   string              `yaml:"touser,omitempty" json:"touser,omitempty"`
	ToParty  string              `yaml:"toparty,omitempty" json:"toparty,omitempty"`
	Totag    string              `yaml:"totag,omitempty" json:"totag,omitempty"`
	AgentID  string              `yaml:"agentid,omitempty" json:"agentid,omitempty"`
	Safe     string              `yaml:"safe,omitempty" json:"safe,omitempty"`
	Type     string              `yaml:"msgtype,omitempty" json:"msgtype,omitempty"`
	Markdown weComMessageContent `yaml:"markdown,omitempty" json:"markdown,omitempty"`
}

type weComMessageContent struct {
	Content string `json:"content"`
}

type weComResponse struct {
	Code  int    `json:"code"`
	Error string `json:"error"`
}

var _ notify.Notifier = new(WeCom)

func New(configMgr *config.Manager, weComClient client.Client) notify.Notifier {
	return &WeCom{
		configMgr:   configMgr,
		client:      weComClient,
		accessToken: map[string]token{},
	}
}

func (wc *WeCom) Notify(stopCh <-chan struct{}, msg *runtime.Message) (bool, error) {
	server, err := wc.configMgr.GetConfig()
	if err != nil {
		return false, err
	}
	conf := server.WeCom

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
		parameters.Add("corpsecret", string(aesutil.DecryptCBC([]byte(conf.CorpSecret), config.GetAesKey(security.GetTenant()))))
		parameters.Add("corpid", conf.CorpId)

		resp, err := wc.client.Get("/cgi-bin/gettoken", http.Header{"Content-Type": []string{"application/json"}}, &parameters)
		if err != nil {
			klog.V(1).InfoS("Failed to access wecom", "err", err)
			return true, err
		}
		defer resp.Body.Close()
		var wechatToken token
		if err := json.NewDecoder(resp.Body).Decode(&wechatToken); err != nil {
			return false, err
		}

		if wechatToken.AccessToken == "" {
			return false, fmt.Errorf("invalid APISecret for CorpID: %s", conf.CorpId)
		}

		// Cache accessToken
		t.AccessToken = wechatToken.AccessToken
		t.at = time.Now()
	}

	weChatMsg := &weComMessage{
		ToUser:  strings.Join(msg.To, "|"),
		AgentID: conf.AgentId,
		Safe:    "0",
	}

	for _, msgTmpl := range msg.MsgTmpls {
		if msgTmpl.ParsedTmpl.TextTmpl != nil {
			weChatMsg.Type = msgTmpl.Type.String()
			var buf bytes.Buffer
			err = msgTmpl.ParsedTmpl.TextTmpl.Execute(&buf, msg)
			if err != nil {
				return false, errors.Wrap(err, "execute text template")
			}
			if weChatMsg.Type == "markdown" {
				weChatMsg.Markdown = weComMessageContent{
					Content: buf.String(),
				}
			} else {
				weChatMsg.Text = weComMessageContent{
					Content: buf.String(),
				}
			}

			resp, err := wc.client.Post("/cgi-bin/message/send", http.Header{"Content-Type": []string{"application/json"}}, map[string]interface{}{"access_token": t.AccessToken}, weChatMsg)
			if err != nil {
				klog.V(1).InfoS("Failed to access wecom", "err", err)
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
			klog.V(5).InfoS("Received wecom response", "response", string(body))

			var weResp weComResponse
			if err := json.Unmarshal(body, &weResp); err != nil {
				return true, err
			}

			// https://work.weixin.qq.com/api/doc#10649
			if weResp.Code == 0 {
				return false, nil
			}

			// AccessToken is expired
			if weResp.Code == 42001 {
				t.AccessToken = ""
				return true, errors.New(weResp.Error)
			}
		}
	}

	return false, nil
}
