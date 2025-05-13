package client

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/spf13/pflag"
	"k8s.io/klog/v2"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"
)

const (
	_defaultReadTimeout    = 60 * time.Second
	_defaultConnectTimeout = 10 * time.Second
	_defaultClientCAFile   = ""
)

type Options struct {
	ReadTimeout    time.Duration `json:"read-timeout"`
	ConnectTimeout time.Duration `json:"connect-timeout"`
	ClientCAFile   string        `json:"client-ca-file"`
}

func NewDefaultOptions() *Options {
	return &Options{
		ReadTimeout:    _defaultReadTimeout,
		ConnectTimeout: _defaultConnectTimeout,
		ClientCAFile:   _defaultClientCAFile,
	}
}

func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fs.DurationVar(&o.ReadTimeout, "read-timeout", o.ReadTimeout, "Http client read timeout")
	fs.DurationVar(&o.ConnectTimeout, "connect-timeout", o.ConnectTimeout, "Http client connect timeout")
	fs.StringVar(&o.ClientCAFile, "client-ca-file", o.ClientCAFile, ""+
		"If set, any request presenting a client certificate signed by one of the authorities in the client-ca-file "+
		"is authenticated with an identity corresponding to the CommonName of the client certificate.")
}

func (o *Options) Validate() []error {
	var errs []error
	if o.ConnectTimeout >= o.ReadTimeout {
		errs = append(errs, fmt.Errorf("--connect-timeout %v must be less than --read-timeout %v", o.ConnectTimeout, o.ReadTimeout))
	}
	return errs
}

func (o *Options) CreateHttpClient(endpoint string) (Client, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		klog.ErrorS(err, "URL not valid", "rawURL", endpoint)
		return nil, err
	}

	tp := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   o.ConnectTimeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   o.ConnectTimeout,
		ExpectContinueTimeout: 1 * time.Second,
	}

	if u.Scheme == "https" {
		var caCertPool *x509.CertPool
		if len(o.ClientCAFile) != 0 {
			caCert, err := os.ReadFile(o.ClientCAFile)
			if err != nil {
				klog.ErrorS(err, "Failed to read client CA file", "filePath", o.ClientCAFile)
				return nil, err
			}
			caCertPool = x509.NewCertPool()
			caCertPool.AppendCertsFromPEM(caCert)
		}

		tp.TLSClientConfig = &tls.Config{
			RootCAs: caCertPool,
		}
	}

	return NewClient(u, o.ReadTimeout, tp), nil
}
