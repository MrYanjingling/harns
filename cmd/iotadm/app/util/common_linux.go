package util

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"strconv"
	"syscall"

	// "github.com/coreos/go-semver/semver"
	"strings"
)

var (
	IsDeleteEdgeIotData = true
	IotUsers            = []string{"edgeiot", "manager"}
	IotServices         = []string{"iot-model-manager", "iot-data-collector", "iot-data-broker", "iot-data-query", "iot-installation-manager"}
	Installations       = []string{"modelmanager", "fleetmanager"}

	OfferMap = map[string]string{
		"event":        "iot-event-manager",
		"rule":         "iot-rule-manager",
		"notification": "iot-notification-manager",
		"control":      "iot-control-manager",
	}
	InstallationMap = map[string]string{
		"rule":         "ruleengine",
		"notification": "notifymanager",
		"control":      "controlmanager",
	}

	IoTInstallations = []string{"public", "launchpad", "modelmanager", "fleetmanager", "ruleengine", "notifymanager", "controlmanager"}
)

func (co *Common) SetOSInterface(intf OSTypeInstaller) {
	co.OSTypeInstaller = intf
}

func GetPackageManager() string {
	cmd := NewCommand("command -v apt || command -v yum")
	err := cmd.Exec()
	if err != nil {
		return ""
	}

	if strings.HasSuffix(cmd.GetStdOut(), APT) {
		return APT
	} else if strings.HasSuffix(cmd.GetStdOut(), YUM) {
		return YUM
	} else {
		return ""
	}
}

func GetOSInterface() OSTypeInstaller {
	switch GetPackageManager() {
	case APT:
		return &DebOS{}
	case YUM:
		return &RpmOS{}
	default:
		return nil
	}
}

func IsProcessRunning(proc string) (bool, error) {
	procRunning := fmt.Sprintf("pidof %s 2>&1", proc)
	cmd := NewCommand(procRunning)

	err := cmd.Exec()
	if cmd.ExitCode == 0 {
		return true, nil
	} else if cmd.ExitCode == 1 {
		return false, nil
	}

	return false, err
}

func IsUserRoot() bool {
	// NOTE The first call will cache the current user information. Subsequent calls will return the cached value and will not reflect changes to the current user.
	u, _ := user.Current()
	return u.Username == "root"
}

// func CopyFile(src, dst, flag string) error {
// 	cmd := NewCommand(fmt.Sprintf("cp %s %s %s", flag, src, dst))
// 	return cmd.Exec()
// }

func RegisterSystemd(svcUnit *SvcUnit, ioStreams IOStreams) error {
	if !IsFileExist(GetSystemSvcPath()) {
		// TODO perm
		err := os.Mkdir(GetSystemSvcPath(), 0755)
		if err != nil {
			_, _ = fmt.Fprintf(ioStreams.ErrOut, "Failed to create system service path\n")
			return err
		}
	}

	svcFile := filepath.Join(svcUnit.Path, svcUnit.SvcFileName)
	for _, cmd := range []string{
		fmt.Sprintf("ln -f %s %s", svcFile, filepath.Join(GetSystemSvcPath(), svcUnit.SvcFileName)),
		fmt.Sprintf("systemctl enable %s", svcUnit.SvcFullName()),
	} {
		command := NewCommand(cmd)
		if err := command.Exec(); err != nil {
			_, _ = fmt.Fprintf(ioStreams.ErrOut, "Failed to register service %s\n", svcUnit.SvcName)
			return err
		}
	}
	_, _ = fmt.Fprintf(ioStreams.Out, "Registered service %s\n", svcUnit.SvcName)

	command := NewCommand("systemctl daemon-reload")
	return command.Exec()
}

func CancelSystemd(svcUnit *SvcUnit, ioStreams IOStreams) error {
	for _, cmd := range []string{
		fmt.Sprintf("systemctl disable %s", svcUnit.SvcFullName()),
		fmt.Sprintf("rm -f %s", path.Join(GetSystemSvcPath(), svcUnit.SvcFileName)),
		"systemctl daemon-reload",
		"systemctl reset-failed",
	} {
		cmd := NewCommand(cmd)
		if err := cmd.Exec(); err != nil {
			_, _ = fmt.Fprintf(ioStreams.ErrOut, "Failed to remove %s service unit\n", svcUnit.SvcName)
			return err
		}
	}
	_, _ = fmt.Fprintf(ioStreams.Out, "Removed %s service unit\n", svcUnit.SvcName)
	return nil
}

func StartSvc(svcUnit *SvcUnit) error {
	command := NewCommand(fmt.Sprintf("systemctl start %s", svcUnit.SvcFullName()))
	if err := command.Exec(); err != nil {
		return err
	}
	return nil
}

func StopSvc(svcUnit *SvcUnit) error {
	command := NewCommand(fmt.Sprintf("systemctl stop %s", svcUnit.SvcFullName()))
	if err := command.Exec(); err != nil {
		return err
	}
	return nil
}

func Decompress(tarFile, dst string) error {
	srcFile, err := os.Open(tarFile)
	if err != nil {
		return err
	}
	defer func(srcFile *os.File) {
		err := srcFile.Close()
		if err != nil {
			fmt.Println(2)
		}
	}(srcFile)

	gr, err := gzip.NewReader(srcFile)
	if err != nil {
		return err
	}
	defer func(gr *gzip.Reader) {
		err := gr.Close()
		if err != nil {
			fmt.Println(3)
		}
	}(gr)

	tr := tar.NewReader(gr)

	for {
		hdr, err := tr.Next()
		switch {
		case err == io.EOF:
			return nil
		case err != nil:
			return err
		case hdr == nil:
			continue
		}

		dstFileDir := filepath.Join(dst, hdr.Name)

		switch hdr.Typeflag {
		case tar.TypeDir:
			if b := existDir(dstFileDir); !b {
				if err := os.MkdirAll(dstFileDir, 0775); err != nil {
					return err
				}
			}
		case tar.TypeReg:
			file, err := os.OpenFile(dstFileDir, os.O_CREATE|os.O_RDWR, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
			_, err = io.Copy(file, tr)
			if err != nil {
				return err
			}
			err = file.Close()
			if err != nil {
				return err
			}
		}
	}
}

func existDir(dirname string) bool {
	fi, err := os.Stat(dirname)
	return (err == nil || os.IsExist(err)) && fi.IsDir()
}

func CreateGroup(groupName string) error {
	g, err := user.LookupGroup(groupName)
	if err != nil {
		_, ok := err.(user.UnknownGroupError)
		if !ok {
			return err
		}
	}

	if g == nil {
		cmd := NewCommand(fmt.Sprintf("groupadd %s", groupName))
		if err = cmd.Exec(); err != nil {
			return err
		}
	}

	return nil
}

func CreateUser(username, groupName string, ioStreams IOStreams) error {
	if err := CreateGroup(groupName); err != nil {
		_, _ = fmt.Fprintf(ioStreams.ErrOut, "Failed to create group %s\n", groupName)
		return err
	}

	if u, _ := user.Lookup(username); u != nil {
		cmd := NewCommand(fmt.Sprintf("usermod -a -G %s %s", groupName, username))
		if err := cmd.Exec(); err != nil {
			_, _ = fmt.Fprintf(ioStreams.ErrOut, "Failed to add user %s group %s\n", username, groupName)
			return err
		}
	} else {
		cmd := NewCommand(fmt.Sprintf("useradd -g %s %s", groupName, username))
		if err := cmd.Exec(); err != nil {
			_, _ = fmt.Fprintf(ioStreams.ErrOut, "Failed to create user %s\n", groupName)
		} else {
			cmd := NewCommand(fmt.Sprintf("echo '%s:%s' | chpasswd", username, username))
			if err = cmd.Exec(); err != nil {
				_, _ = fmt.Fprintf(ioStreams.ErrOut, "Failed to set password for user %s\n", username)
			} else {
				_, _ = fmt.Fprintf(ioStreams.Out, "Created user %s, with password %s\n", username, username)
			}
		}
	}
	return nil
}

func IsFileUser(path, username string) bool {
	if stat, err := os.Stat(path); err == nil {
		fileSys := stat.Sys()
		fileUid := fmt.Sprint(fileSys.(*syscall.Stat_t).Uid)
		if edgeIot, err := user.Lookup(username); err == nil && edgeIot.Uid == fileUid {
			return true
		}
	}
	return false
}

func AskForConfirm(ioStreams IOStreams) bool {
	var s string
	fmt.Print("[Y/n]: ")
	if _, err := fmt.Fscanln(ioStreams.In, &s); err != nil {
		return false
	}
	if strings.ToUpper(s) == "Y" {
		return true
	}
	return false
}

func GetMemTotal() (uint64, error) {
	file, err := os.Open(MemInfoConfigPath)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		i := strings.IndexRune(line, ':')
		if i < 0 {
			continue
		}
		fld := line[:i]
		if strings.TrimSpace(fld) == MemTotalField {
			val := strings.TrimSpace(strings.TrimRight(line[i+1:], "kB"))
			return strconv.ParseUint(val, 10, 64)
		}
	}
	return 0, nil
}

const (
	dataPath      = "/var/lib/"
	runtimePath   = "/usr/local/"
	configPath    = "/etc/"
	systemSvcPath = "/usr/lib/systemd/system"
)

func GetAbsoluteDataPath(path string) string {
	return filepath.Join(dataPath, path)
}

func GetAbsoluteRuntimePath(path string) string {
	return filepath.Join(runtimePath, path)
}

func GetAbsoluteConfigPath(path string) string {
	return filepath.Join(configPath, path)
}

func GetSystemSvcPath() string {
	return systemSvcPath
}
