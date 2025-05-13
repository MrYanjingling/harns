package validation

import (
	"fmt"
	htmlTmpl "html/template"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"lightiot/pkg/generic"
	"lightiot/pkg/notification/runtime"
	v1 "lightiot/pkg/notification/v1"
	"lightiot/pkg/util/validation"
	"net/url"
	"regexp"
	"strings"
	textTmpl "text/template"
)

const (
	_illegalNameChars = "\u002F\u005C" // /\
)

func ValidateServerConfig(c *v1.Config) field.ErrorList {
	var allErrs field.ErrorList
	allErrs = append(allErrs, validateEmailServerConfig(c.Email, field.NewPath("email"))...)
	allErrs = append(allErrs, validateWeComConfig(c.WeCom, field.NewPath("weCom"))...)
	allErrs = append(allErrs, validateWeChatConfig(c.WeChat, field.NewPath("weChat"))...)
	return allErrs
}

func validateEmailServerConfig(e *v1.EmailConfig, fldPath *field.Path) (errs field.ErrorList) {
	if e == nil {
		return
	}
	smtp := fldPath.Child("smtp")
	if len(e.Smtp.Hostname) == 0 {
		errs = append(errs, field.Required(smtp.Child("hostname"), ""))
	} else if !validation.IsHostnameRFC952(e.Smtp.Hostname) {
		errs = append(errs, field.Invalid(smtp.Child("hostname"), e.Smtp.Hostname, "hostname must follow RFC952"))
	}
	if len(e.Smtp.Port) == 0 {
		errs = append(errs, field.Required(smtp.Child("port"), ""))
	} else if !validation.IsNumber(e.Smtp.Port) {
		errs = append(errs, field.Invalid(smtp.Child("port"), e.Smtp.Port, "port must be a number"))
	}
	if len(e.Smtp.Username) == 0 {
		errs = append(errs, field.Required(smtp.Child("username"), ""))
	} else if !validation.IsEmail(e.Smtp.Username) {
		errs = append(errs, field.Invalid(smtp.Child("username"), e.Smtp.Username, "username must be an email"))
	}
	if len(e.Smtp.Password) == 0 {
		errs = append(errs, field.Required(smtp.Child("password"), ""))
	}
	return
}

func validateWeComConfig(com *v1.WeComConfig, fldPath *field.Path) (errs field.ErrorList) {
	if com == nil {
		return
	}
	if len(com.CorpId) == 0 {
		errs = append(errs, field.Required(fldPath.Child("corpId"), ""))
	}
	if len(com.CorpSecret) == 0 {
		errs = append(errs, field.Required(fldPath.Child("corpSecret"), ""))
	}
	if len(com.AgentId) == 0 {
		errs = append(errs, field.Required(fldPath.Child("agentId"), ""))
	}
	return
}

func validateWeChatConfig(chat *v1.WeChatConfig, fldPath *field.Path) (errs field.ErrorList) {
	if chat == nil {
		return
	}
	if len(chat.AppId) == 0 {
		errs = append(errs, field.Required(fldPath.Child("appId"), ""))
	}
	if len(chat.AppSecret) == 0 {
		errs = append(errs, field.Required(fldPath.Child("appSecret"), ""))
	}
	return
}

func isValidName(name string) bool {
	return validation.ExcludesAll(name, _illegalNameChars)
}

func validateName(name string) error {
	if !isValidName(name) {
		return fmt.Errorf("contains illegal chars [%s]", _illegalNameChars)
	}
	if !validation.IsStringLengthInRange(name, 1, 64) {
		return fmt.Errorf("must have at most %d bytes", 64)
	}
	return nil
}

func ValidateRecipient(r *v1.Recipient) field.ErrorList {
	var allErrs field.ErrorList
	allErrs = append(allErrs, generic.ValidateName(r.Name, validateName, field.NewPath("name"))...)
	detailPath := field.NewPath("detail")
	if len(r.Detail) == 0 {
		allErrs = append(allErrs, field.Required(detailPath, ""))
	} else {
		for i, d := range r.Detail {
			allErrs = append(allErrs, generic.ValidateName(d.Address, validateName, detailPath.Index(i).Child("address"))...)
			if d.Type < 0 {
				allErrs = append(allErrs, field.Required(detailPath.Index(i).Child("type"), ""))
			}
			if err := recipientValidators[d.Type](d.Address, detailPath.Index(i).Child("address")); err != nil {
				allErrs = append(allErrs, err)
			}
		}
	}
	return allErrs
}

var recipientValidators = map[v1.ChannelType]func(string, *field.Path) *field.Error{
	v1.ChannelTypeEmail: func(address string, fldPath *field.Path) *field.Error {
		if !validation.IsEmail(address) {
			return field.Invalid(fldPath, address, "must be an email")
		}
		return nil
	},
	v1.ChannelTypeWebhook: func(address string, fldPath *field.Path) *field.Error {
		if _, err := url.Parse(address); err != nil {
			return field.Invalid(fldPath, address, "must be an URL")
		}
		return nil
	},
	v1.ChannelTypeWeCom: func(address string, fldPath *field.Path) *field.Error {
		return nil
	},
	v1.ChannelTypeWeChat: func(address string, fldPath *field.Path) *field.Error {
		return nil
	},
}

func ValidateMessageTmpl(mt *v1.MessageTmpl) field.ErrorList {
	var allErrs field.ErrorList
	allErrs = append(allErrs, generic.ValidateName(mt.Name, validateName, field.NewPath("name"))...)
	if mt.Type == 0 {
		allErrs = append(allErrs, field.Required(field.NewPath("type"), ""))
	}
	if len(mt.Content) == 0 {
		allErrs = append(allErrs, field.Required(field.NewPath("content"), ""))
	} else {
		allErrs = append(allErrs, messageTemplateValidators[mt.Type](mt.Content, field.NewPath("content"))...)
	}

	return allErrs
}

func validatePlaceholder(ph string, fldPath *field.Path) (string, *field.Error) {
	ph = strings.TrimSpace(ph)
	if strings.HasPrefix(ph, ".") {
		return "", field.Invalid(fldPath, ph, "placeholder must not start with .")
	}
	r := []rune(ph)
	if len(r) > 64 {
		return "", field.TooLong(fldPath, ph, 64)
	}
	return ph, nil
}

var messageTemplateValidators = map[v1.ContentType]func(string, *field.Path) (errs field.ErrorList){
	v1.ContentTypeText: func(content string, fldPath *field.Path) (errs field.ErrorList) {
		rgx := regexp.MustCompile(runtime.PlaceholderRegExp)
		reps := rgx.ReplaceAllStringFunc(content, func(s string) string {
			ph := s[2 : len(s)-2]
			ph, err := validatePlaceholder(ph, fldPath)
			if err != nil {
				errs = append(errs, err)
				return ""
			}
			return ""
		})
		_, err := textTmpl.New("").Parse(reps)
		if err != nil {
			errs = append(errs, field.Invalid(fldPath, content, "invalid"))
		}
		return errs
	},
	v1.ContentTypeHtml: func(content string, fldPath *field.Path) (errs field.ErrorList) {
		rgx := regexp.MustCompile(runtime.PlaceholderRegExp)
		reps := rgx.ReplaceAllStringFunc(content, func(s string) string {
			ph := s[2 : len(s)-2]
			ph, err := validatePlaceholder(ph, fldPath)
			if err != nil {
				errs = append(errs, err)
				return ""
			}
			return ""
		})
		_, err := htmlTmpl.New("").Parse(reps)
		if err != nil {
			errs = append(errs, field.Invalid(fldPath, content, "invalid"))
		}
		return errs
	},
	v1.ContentTypeMarkdown: func(address string, fldPath *field.Path) (errs field.ErrorList) {
		return nil
	},
}

func ValidateTemplate(t *v1.Template) field.ErrorList {
	var allErrs field.ErrorList
	allErrs = append(allErrs, generic.ValidateName(t.Name, validateName, field.NewPath("name"))...)
	if len(t.Subject) == 0 {
		allErrs = append(allErrs, field.Required(field.NewPath("subject"), ""))
	} else if len(t.Subject) > 77 {
		allErrs = append(allErrs, field.TooLong(field.NewPath("subject"), t.Subject, 77))
	}

	allErrs = append(allErrs, generic.ValidateName(t.From, validateName, field.NewPath("from"))...)

	if len(t.Recipients) == 0 {
		allErrs = append(allErrs, field.Required(field.NewPath("recipients"), ""))
	}

	if len(t.MessageTemplates) == 0 {
		allErrs = append(allErrs, field.Required(field.NewPath("messageTemplates"), ""))
	}
	return allErrs
}

func ValidateMessage(msg *v1.Message) field.ErrorList {
	var allErrs field.ErrorList
	if msg.Channel == 0 {
		allErrs = append(allErrs, field.Required(field.NewPath("channel"), ""))
	}
	if len(msg.Subject) > 77 {
		allErrs = append(allErrs, field.TooLong(field.NewPath("subject"), msg.Subject, 77))
	}
	if len(msg.From) > 64 {
		allErrs = append(allErrs, field.TooLong(field.NewPath("from"), msg.From, 64))
	}
	if len(msg.Payload) == 0 {
		allErrs = append(allErrs, field.Required(field.NewPath("payload"), ""))
	}
	if len(msg.To) == 0 {
		if len(msg.TemplateId) == 0 {
			allErrs = append(allErrs, field.Required(nil, "either to or templateId is required"))
		}
	}
	return allErrs
}
