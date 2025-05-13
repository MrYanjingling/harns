package v1

const (
	SecretPlaceholder = "<secret>"
)

type ChannelType byte

const (
	_ ChannelType = iota
	ChannelTypeEmail
	ChannelTypeWebhook
	ChannelTypeWeCom
	ChannelTypeWeChat
)

var ChannelTypeToString = map[ChannelType]string{
	ChannelTypeEmail:   "email",
	ChannelTypeWebhook: "webhook",
	ChannelTypeWeCom:   "weCom",
	ChannelTypeWeChat:  "weChat",
}

var ChannelTypeFromString = map[string]ChannelType{
	"email":   ChannelTypeEmail,
	"webhook": ChannelTypeWebhook,
	"weCom":   ChannelTypeWeCom,
	"weChat":  ChannelTypeWeChat,
}

type RecipientIem struct {
	Address string      `json:"address" binding:"required,min=1,max=64"`
	Type    ChannelType `json:"type" binding:"gte=0"`
}

type Recipient struct {
	Name   string         `json:"name" binding:"required,min=1,max=32"`
	Detail []RecipientIem `json:"detail" binding:"required,dive"`
}

type ContentType byte

const (
	_ ContentType = iota
	ContentTypeText
	ContentTypeHtml
	ContentTypeMarkdown
)

var ContentTypeToString = map[ContentType]string{
	ContentTypeText:     "text",
	ContentTypeHtml:     "html",
	ContentTypeMarkdown: "markdown",
}

var ContentTypeFromString = map[string]ContentType{
	"text": ContentTypeText,
	"html": ContentTypeHtml,
	// "markdown":   ContentTypeMarkdown,
}

func (ct ContentType) String() string {
	return ContentTypeToString[ct]
}

type MessageTmpl struct {
	Name    string      `json:"name" binding:"required,min=1,max=64"`
	Type    ContentType `json:"type" binding:"gte=0"`
	Content string      `json:"content" binding:"required"`
}

type TmplRecipient struct {
	Id string `json:"id" binding:"required,min=22,max=26"`
}

type Template struct {
	Name             string          `json:"name" binding:"required,min=1,max=64"`
	Subject          string          `json:"subject" binding:"required,min=1,max=77"`
	From             string          `json:"from" binding:"required,min=1,max=64"`
	Recipients       []TmplRecipient `json:"recipients" binding:"required,dive"`
	MessageTemplates []string        `json:"messageTemplates" binding:"required"`
}

type Message struct {
	Channel    ChannelType            `json:"channel" binding:"gte=0"`
	Subject    string                 `json:"subject" binding:"max=77"`
	From       string                 `json:"from" binding:"max=64"`
	Payload    map[string]interface{} `json:"payload" binding:"required,min=1"`
	To         []string               `json:"to" binding:"required_without=TemplateId,min=1"`
	TemplateId string                 `json:"templateId" binding:"required_without=To"`
}

type Secret string

type SMTPConfig struct {
	Hostname string `json:"hostname" binding:"required,hostname"`
	Port     string `json:"port" binding:"required,number"`
	Username string `json:"username" binding:"required,email"`
	Password Secret `json:"password" binding:"required"`
}

type WeComConfig struct {
	CorpId     string `json:"corpId" binding:"required,min=1"`
	CorpSecret Secret `json:"corpSecret" binding:"required,min=1"`
	AgentId    string `json:"agentId" binding:"required,min=1"`
}

type WeChatConfig struct {
	AppId     string `json:"appId" binding:"required,min=1"`
	AppSecret Secret `json:"appSecret" binding:"required,min=1"`
}

type EmailConfig struct {
	Smtp SMTPConfig `json:"smtp" binding:"required,dive"`
}

type Config struct {
	Email  *EmailConfig  `json:"email,omitempty"`
	WeCom  *WeComConfig  `json:"weCom,omitempty"`
	WeChat *WeChatConfig `json:"weChat,omitempty"`
}
