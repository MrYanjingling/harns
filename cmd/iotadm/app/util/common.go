package util

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/fs"
	"lightiot/pkg/client"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"strconv"
	text "text/template"
	"time"
)

type Common struct {
	OSTypeInstaller
	OSVersion string
	IOStreams
	// ToolVersion semver.Version
	KubeConfig string
	Master     string
}

// OSTypeInstaller interface for methods to be executed over a specified OS distribution type
type OSTypeInstaller interface {
	InstallGateway() error
}

func CopyFileWithRecursive(src, dst string, ioStreams IOStreams) error {
	var (
		err     error
		des     []os.DirEntry
		srcInfo os.FileInfo
	)

	if srcInfo, err = os.Stat(src); err != nil {
		return err
	}

	if err = os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	if des, err = os.ReadDir(src); err != nil {
		return err
	}
	for _, de := range des {
		srcDe := path.Join(src, de.Name())
		dstDe := path.Join(dst, de.Name())

		if de.IsDir() {
			if err = CopyFileWithRecursive(srcDe, dstDe, ioStreams); err != nil {
				_, _ = fmt.Fprintf(ioStreams.ErrOut, "Failed to copy dir from %s to %s because %s\n", srcDe, dstDe, err)
			}
		} else {
			if err = CopyFile(srcDe, dstDe); err != nil {
				_, _ = fmt.Fprintf(ioStreams.ErrOut, "Failed to copy file from %s to %s because %s\n", srcDe, dstDe, err)
			}
		}
	}
	return nil
}

func CopyFile(src, dst string) error {
	var (
		err     error
		srcInfo os.FileInfo
		srcFile *os.File
		dstFile *os.File
	)
	srcInfo, err = os.Stat(src)
	if err != nil {
		return err
	}
	srcFile, err = os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()
	dstFile, err = os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()
	if _, err = io.Copy(dstFile, srcFile); err != nil {
		return err
	}
	return os.Chmod(dst, srcInfo.Mode())
}

type SvcUnit struct {
	SvcName     string
	SvcFileName string
	Path        string
	Replicas    int
	File        []FileDesc
}

func NewSvcUnit(svcName string, svcPath string, replicas int, file []FileDesc) *SvcUnit {
	var svcFileName string
	if replicas > 1 {
		svcFileName = fmt.Sprintf("%s@.service", svcName)
	} else {
		svcFileName = fmt.Sprintf("%s.service", svcName)
	}
	return &SvcUnit{
		SvcName:     svcName,
		Path:        svcPath,
		SvcFileName: svcFileName,
		Replicas:    replicas,
		File:        file,
	}
}

func (su *SvcUnit) IsMultiInstance() bool {
	return su.Replicas > 1
}

func (su *SvcUnit) SvcFullName() string {
	if su.IsMultiInstance() {
		return fmt.Sprintf("%s@{0..%d}", su.SvcName, su.Replicas-1)
	} else {
		return su.SvcName
	}
}

type FileDesc struct {
	Path     string
	Template *text.Template
	Data     interface{}
	Perm     fs.FileMode
	Desc     string
}

type ApiSvcConfig struct {
	StartCmd string
	User     string
	Group    string
}

func GenerateFiles(fds []FileDesc, ioStreams IOStreams) error {
	for _, fd := range fds {
		var perm fs.FileMode
		if fd.Perm != 0 {
			perm = fd.Perm
		} else {
			perm = 0644
		}

		dir := filepath.Dir(fd.Path)
		if !IsFileExist(dir) {
			if err := os.MkdirAll(dir, 0754); err != nil {
				_, _ = fmt.Fprintf(ioStreams.ErrOut, "Failed to create dir %s\n", dir)
				return err
			}
		}

		if err := func() error {
			file, err := os.OpenFile(fd.Path, os.O_CREATE|os.O_RDWR|os.O_TRUNC, perm)
			if err != nil {
				_, _ = fmt.Fprintf(ioStreams.ErrOut, "Failed to create file %s\n", fd.Path)
				return err
			}
			defer file.Close()
			if err = fd.Template.Execute(file, fd.Data); err != nil {
				_, _ = fmt.Fprintf(ioStreams.ErrOut, "Failed to generate file %s\n", fd.Path)
				return err
			}
			return nil
		}(); err != nil {
			return err
		}

		_, _ = fmt.Fprintf(ioStreams.Out, "Generated file %s\n", fd.Path)
	}

	return nil
}

func IsFileExist(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}

func GetAllFiles(path string, ioStreams IOStreams) ([]string, error) {
	var fis []string
	des, err := os.ReadDir(path)
	if err != nil {
		_, _ = fmt.Fprintf(ioStreams.ErrOut, "Failed to read path %s\n", path)
		return nil, err
	}

	for _, de := range des {
		if !de.IsDir() {
			fis = append(fis, filepath.Join(path, de.Name()))
		}
	}
	return fis, nil
}

// ChownFile with recursive
func ChownFile(username, root string, ioStreams IOStreams) error {
	var (
		uid, gid int
	)
	if u, err := user.Lookup(username); err != nil {
		_, _ = fmt.Fprintf(ioStreams.ErrOut, "Failed to look up user %s\n", username)
		return err
	} else {
		uid, _ = strconv.Atoi(u.Uid)
		gid, _ = strconv.Atoi(u.Gid)
	}
	return filepath.Walk(root, func(path string, info fs.FileInfo, err error) error {
		if err := os.Chown(path, uid, gid); err != nil {
			_, _ = fmt.Fprintf(ioStreams.ErrOut, "Failed to change owner of %s\n", path)
		}
		return nil
	})
}

func ChmodFile(root string, perm fs.FileMode, ioStreams IOStreams) error {
	return filepath.Walk(root, func(path string, info fs.FileInfo, err error) error {
		if err := os.Chmod(path, perm); err != nil {
			_, _ = fmt.Fprintf(ioStreams.ErrOut, "Failed to change permission of %s\n", path)
		}
		return nil
	})
}

func AddExecutablePermission(root string, ioStreams IOStreams) error {
	return filepath.Walk(root, func(path string, info fs.FileInfo, err error) error {
		perm := info.Mode()&fs.ModePerm | 0111
		if err := os.Chmod(path, perm); err != nil {
			_, _ = fmt.Fprintf(ioStreams.ErrOut, "Failed to add executable permission of %s\n", path)
			return err
		}
		return nil
	})
}

func CreateHttpClient(endpoint string, clientCAFile string, ioStreams IOStreams) (client.Client, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}

	tp := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	if u.Scheme == "https" {
		var caCertPool *x509.CertPool
		if len(clientCAFile) != 0 {
			caCert, err := os.ReadFile(clientCAFile)
			if err != nil {
				_, _ = fmt.Fprintf(ioStreams.ErrOut, "Failed to read client CA file from %s\n", clientCAFile)
				return nil, err
			}
			caCertPool = x509.NewCertPool()
			caCertPool.AppendCertsFromPEM(caCert)
		}

		tp.TLSClientConfig = &tls.Config{
			RootCAs: caCertPool,
		}
	}

	return client.NewClient(u, 60*time.Second, tp), nil
}

// leadingInt consumes the leading [0-9]* from s.
func leadingInt(s string) (x int64, rem string, err error) {
	i := 0
	for ; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			break
		}
		if x > (1<<63-1)/10 {
			// overflow
			return 0, "", fmt.Errorf("errLeadingInt")
		}
		x = x*10 + int64(c) - '0'
		if x < 0 {
			// overflow
			return 0, "", fmt.Errorf("errLeadingInt")
		}
	}
	return x, s[i:], nil
}

func leadingFraction(s string) (x int64, scale float64, rem string) {
	i := 0
	scale = 1
	overflow := false
	for ; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			break
		}
		if overflow {
			continue
		}
		if x > (1<<63-1)/10 {
			// It's possible for overflow to give a positive number, so take care.
			overflow = true
			continue
		}
		y := x*10 + int64(c) - '0'
		if y < 0 {
			overflow = true
			continue
		}
		x = y
		scale *= 10
	}
	return x, scale, s[i:]
}

var unitMap = map[string]int64{
	"h": int64(OneHour),
	"d": int64(OneDayInHours),
	"w": int64(OneWeekInHours),
}

// _M_d_h_m
func ParseDuration(s string) (int64, error) {
	// [-+]?([0-9]*(\.[0-9]*)?[a-z]+)+
	orig := s
	var d int64

	if s == "" {
		return 0, fmt.Errorf("The duration string is empty\n")
	}
	for s != "" {
		var (
			v, f  int64       // integers before, after decimal point
			scale float64 = 1 // value = v + f/scale
		)

		var err error

		// The next character must be [0-9.]
		if !(s[0] == '.' || '0' <= s[0] && s[0] <= '9') {
			return 0, fmt.Errorf("Invalid duration %s\n", orig)
		}
		// Consume [0-9]*
		pl := len(s)
		v, s, err = leadingInt(s)
		if err != nil {
			return 0, fmt.Errorf("Invalid duration %s\n", orig)
		}
		pre := pl != len(s) // whether we consumed anything before a period

		// Consume (\.[0-9]*)?
		post := false
		if s != "" && s[0] == '.' {
			s = s[1:]
			pl := len(s)
			f, scale, s = leadingFraction(s)
			post = pl != len(s)
		}
		if !pre && !post {
			// no digits (e.g. ".s" or "-.s")
			return 0, fmt.Errorf("Invalid duration %s\n", orig)
		}

		// Consume unit.
		i := 0
		for ; i < len(s); i++ {
			c := s[i]
			if c == '.' || '0' <= c && c <= '9' {
				break
			}
		}
		if i == 0 {
			return 0, fmt.Errorf("Missing unit in duration %s\n", orig)
		}
		u := s[:i]
		s = s[i:]
		unit, ok := unitMap[u]
		if !ok {
			return 0, fmt.Errorf("Unknown unit %s in duration %s\n", u, orig)
		}
		if v > (1<<63-1)/unit {
			// overflow
			return 0, fmt.Errorf("Invalid duration %s\n", orig)
		}
		v *= unit
		if f > 0 {
			// float64 is needed to be nanosecond accurate for fractions of hours.
			// v >= 0 && (f*unit/scale) <= 3.6e+12 (ns/h, h is the largest unit)
			v += int64(float64(f) * (float64(unit) / scale))
			if v < 0 {
				// overflow
				return 0, fmt.Errorf("Invalid duration %s\n", orig)
			}
		}
		d += v
		if d < 0 {
			// overflow
			return 0, fmt.Errorf("Invalid duration %s\n", orig)
		}
	}

	return d, nil
}
