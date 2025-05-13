package email

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"github.com/pkg/errors"
	"k8s.io/klog/v2"
	"lightiot/pkg/apis/response"
	"lightiot/pkg/notification/config"
	"lightiot/pkg/notification/notify"
	"lightiot/pkg/notification/runtime"
	v1 "lightiot/pkg/notification/v1"
	"lightiot/pkg/util/aesutil"
	"lightiot/pkg/util/security"
	"math/rand"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"os"
	"strings"
	"time"
)

type Email struct {
	// TODO it violates principle of least privilege
	configMgr *config.Manager
	hostName  string
}

var _ notify.Notifier = new(Email)

func New(configMgr *config.Manager) notify.Notifier {
	h, err := os.Hostname()
	// If we can't get the hostname, we'll use localhost
	if err != nil {
		h = "localhost.localdomain"
	}
	return &Email{
		configMgr: configMgr,
		hostName:  h,
	}
}

func (e *Email) Notify(stopCh <-chan struct{}, msg *runtime.Message) (bool, error) {
	var (
		c       *smtp.Client
		conn    net.Conn
		err     error
		success = false
	)
	server, err := e.configMgr.GetConfig()
	if err != nil {
		return false, err
	}
	conf := server.Email.Smtp

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// TODO set Timeout
	d := net.Dialer{}
	conn, err = d.DialContext(ctx, "tcp", fmt.Sprintf("%s:%s", conf.Hostname, conf.Port))
	if err != nil {
		return true, errors.Wrap(err, "establish connection to server")
	}
	c, err = smtp.NewClient(conn, conf.Hostname)
	if err != nil {
		conn.Close()
		return true, errors.Wrap(err, "create SMTP client")
	}
	defer func() {
		// Try to clean up after ourselves but don't log anything if something has failed.
		if err := c.Quit(); success && err != nil {
			klog.V(2).InfoS("Failed to close SMTP connection", "err", err)
		}
	}()

	err = c.Hello(e.hostName)
	if err != nil {
		return true, errors.Wrap(err, "send EHLO command")
	}

	if ok, _ := c.Extension("STARTTLS"); ok {
		tlsConfig := &tls.Config{ServerName: conf.Hostname}
		// if testHookStartTLS != nil {
		//	testHookStartTLS(config)
		// }
		if err = c.StartTLS(tlsConfig); err != nil {
			return true, errors.Wrap(err, "send STARTTLS command")
		}
	}

	if ok, mech := c.Extension("AUTH"); ok {
		auth, err := e.auth(mech, &conf)
		if err != nil {
			return true, errors.Wrap(err, "find auth mechanism")
		}
		if auth != nil {
			if err := c.Auth(auth); err != nil {
				return true, errors.Wrapf(err, "%T auth", auth)
			}
		}
	}

	addrs, err := mail.ParseAddressList(msg.From)
	if err != nil {
		return false, errors.Wrap(err, "parse 'from' addresses")
	}
	if len(addrs) != 1 {
		return false, errors.Errorf("must be exactly one 'from' address (got: %d)", len(addrs))
	}
	if err = c.Mail(addrs[0].Address); err != nil {
		return true, errors.Wrap(err, "send MAIL command")
	}
	addrs, err = mail.ParseAddressList(strings.Join(msg.To, ", "))
	if err != nil {
		return false, errors.Wrapf(err, "parse 'to' addresses")
	}
	for _, addr := range addrs {
		if err = c.Rcpt(addr.Address); err != nil {
			return true, errors.Wrapf(err, "send RCPT command")
		}
	}

	// Send the email headers and body.
	message, err := c.Data()
	if err != nil {
		return true, errors.Wrapf(err, "send DATA command")
	}
	defer message.Close()

	buffer := &bytes.Buffer{}

	fmt.Fprintf(buffer, "%s: %s\r\n", "Subject", mime.QEncoding.Encode("utf-8", msg.Subject))
	fmt.Fprintf(buffer, "%s: %s\r\n", "To", mime.QEncoding.Encode("utf-8", strings.Join(msg.To, ", ")))
	fmt.Fprintf(buffer, "%s: %s\r\n", "From", mime.QEncoding.Encode("utf-8", msg.From))
	fmt.Fprintf(buffer, "Message-Id: %s\r\n", fmt.Sprintf("<%d.%d@%s>", time.Now().UnixNano(), rand.Uint64(), e.hostName))

	multipartBuffer := &bytes.Buffer{}
	multipartWriter := multipart.NewWriter(multipartBuffer)

	fmt.Fprintf(buffer, "Date: %s\r\n", time.Now().Format(time.RFC1123Z))
	fmt.Fprintf(buffer, "Content-Type: multipart/alternative;  boundary=%s\r\n", multipartWriter.Boundary())
	fmt.Fprintf(buffer, "MIME-Version: 1.0\r\n\r\n")

	// TODO: Add some useful headers here, such as URL of the alertmanager
	// and active/resolved.
	_, err = message.Write(buffer.Bytes())
	if err != nil {
		return false, errors.Wrap(err, "write headers")
	}

	for _, msgTmpl := range msg.MsgTmpls {
		if msgTmpl.ParsedTmpl.TextTmpl != nil {
			// Text template
			w, err := multipartWriter.CreatePart(textproto.MIMEHeader{
				"Content-Transfer-Encoding": {"quoted-printable"},
				"Content-Type":              {"text/plain; charset=UTF-8"},
			})
			if err != nil {
				return false, errors.Wrap(err, "create part for text template")
			}
			// body, err := n.tmpl.ExecuteTextString(n.conf.Text, data)
			var buf bytes.Buffer
			err = msgTmpl.ParsedTmpl.TextTmpl.Execute(&buf, msg)
			if err != nil {
				return false, errors.Wrap(err, "execute text template")
			}
			qw := quotedprintable.NewWriter(w)
			_, err = qw.Write(buf.Bytes())
			if err != nil {
				return true, errors.Wrap(err, "write text part")
			}
			err = qw.Close()
			if err != nil {
				return true, errors.Wrap(err, "close text part")
			}
		}

		if msgTmpl.ParsedTmpl.HtmlTmpl != nil {
			// Html template
			// Preferred alternative placed last per section 5.1.4 of RFC 2046
			// https://www.ietf.org/rfc/rfc2046.txt
			w, err := multipartWriter.CreatePart(textproto.MIMEHeader{
				"Content-Transfer-Encoding": {"quoted-printable"},
				"Content-Type":              {"text/html; charset=UTF-8"},
			})
			if err != nil {
				return false, errors.Wrap(err, "create part for html template")
			}
			// body, err := n.tmpl.ExecuteHTMLString(n.conf.HTML, data)
			var buf bytes.Buffer
			err = msgTmpl.ParsedTmpl.HtmlTmpl.Execute(&buf, msg)
			if err != nil {
				return false, errors.Wrap(err, "execute html template")
			}
			qw := quotedprintable.NewWriter(w)
			_, err = qw.Write(buf.Bytes())
			if err != nil {
				return true, errors.Wrap(err, "write HTML part")
			}
			err = qw.Close()
			if err != nil {
				return true, errors.Wrap(err, "close HTML part")
			}
		}
	}

	err = multipartWriter.Close()
	if err != nil {
		return false, errors.Wrap(err, "close multipartWriter")
	}

	_, err = message.Write(multipartBuffer.Bytes())
	if err != nil {
		return false, errors.Wrap(err, "write body buffer")
	}

	success = true
	return false, nil
}

// auth resolves a string of authentication mechanisms.
func (e *Email) auth(mechs string, conf *v1.SMTPConfig) (smtp.Auth, error) {
	username := conf.Username
	passwd := aesutil.DecryptCBC([]byte(conf.Password), config.GetAesKey(security.GetTenant()))

	err := &response.MultiError{}
	for _, mech := range strings.Split(mechs, " ") {
		switch mech {
		case "CRAM-MD5":
			secret := string(passwd)
			if secret == "" {
				err.Add(errors.New("missing secret for CRAM-MD5 auth mechanism"))
				continue
			}
			return smtp.CRAMMD5Auth(username, secret), nil

		case "PLAIN":
			password := string(passwd)
			if password == "" {
				err.Add(errors.New("missing password for PLAIN auth mechanism"))
				continue
			}
			identity := conf.Username

			return smtp.PlainAuth(identity, username, password, conf.Hostname), nil
		case "LOGIN":
			password := string(passwd)
			if password == "" {
				err.Add(errors.New("missing password for LOGIN auth mechanism"))
				continue
			}
			return LoginAuth(username, password), nil
		}
	}
	if err.Len() == 0 {
		err.Add(errors.New("unknown auth mechanism: " + mechs))
	}
	return nil, err
}

type loginAuth struct {
	username, password string
}

func LoginAuth(username, password string) smtp.Auth {
	return &loginAuth{username, password}
}

func (a *loginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", []byte{}, nil
}

// Used for AUTH LOGIN. (Maybe password should be encrypted)
func (a *loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		switch strings.ToLower(string(fromServer)) {
		case "username:":
			return []byte(a.username), nil
		case "password:":
			return []byte(a.password), nil
		default:
			return nil, errors.New("unexpected server challenge")
		}
	}
	return nil, nil
}
