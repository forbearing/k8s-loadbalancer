package nginx

import (
	"bytes"
	"errors"
	"io"
	"os/exec"
	"strconv"
	"sync"

	"github.com/sirupsen/logrus"
)

var (
	locker sync.RWMutex

	stdout = logrus.New().WriterLevel(logrus.DebugLevel)
)

type Nginx struct {
	err error
}

// Err returns the first errors that was encountered by the Do() method.
func (n *Nginx) Err() error {
	return n.err
}

// setErr records the first error encountered by the Do() method.
func (n *Nginx) setErr(err error) {
	if n.err == nil {
		n.err = err
	}
}

// Install will intall the nginx package in linux.
func Install() (error, int, string) {
	locker.Lock()
	defer locker.Unlock()
	errBuf := &bytes.Buffer{}
	return executeCommand([]string{"bash", "-c", NGINX_INSTALL}, stdout, errBuf)
}

// Remove() will uninstall the nginx package in linux.
func Remove() (error, int, string) {
	locker.Lock()
	defer locker.Unlock()
	errBuf := &bytes.Buffer{}
	return executeCommand([]string{"bash", "-c", NGINX_REMOVE}, stdout, errBuf)
}

// Start will start nginx daemon by systemctl.
func Start() (error, int, string) {
	locker.Lock()
	defer locker.Unlock()
	errBuf := &bytes.Buffer{}
	return executeCommand([]string{"bash", "-c", NGINX_START}, stdout, errBuf)
}

// Stop will stop nginx daemon by systemctl.
func Stop() (error, int, string) {
	locker.Lock()
	defer locker.Unlock()
	errBuf := &bytes.Buffer{}
	return executeCommand([]string{"bash", "-c", NGINX_STOP}, stdout, errBuf)
}

// Restart will restart nginx daemon by systemctl.
func Restart() (error, int, string) {
	locker.Lock()
	defer locker.Unlock()
	errBuf := &bytes.Buffer{}
	return executeCommand([]string{"bash", "-c", NGINX_RESTART}, stdout, errBuf)
}

// Reload will reload nginx daemon by systemctl.
func Reload() (error, int, string) {
	locker.Lock()
	defer locker.Unlock()
	errBuf := &bytes.Buffer{}
	return executeCommand([]string{"bash", "-c", NGINX_RELOAD}, stdout, errBuf)
}

// EnabledNow will enabled and start nginx daemon by systemctl.
func EnabledNow() (error, int, string) {
	locker.Lock()
	defer locker.Unlock()
	errBuf := &bytes.Buffer{}
	return executeCommand([]string{"bash", "-c", NGINX_ENABLEDNOW}, stdout, errBuf)
}

// Enabled will enabled nginx daemon by systemctl.
func Enabled() (error, int, string) {
	locker.Lock()
	defer locker.Unlock()
	errBuf := &bytes.Buffer{}
	return executeCommand([]string{"bash", "-c", NGINX_ENABLED}, stdout, errBuf)
}

// TestConf will test nginx configuration file.
func TestConf() (error, int, string) {
	locker.Lock()
	defer locker.Unlock()
	errBuf := &bytes.Buffer{}
	return executeCommand([]string{"bash", "-c", NGINX_TESTCONF}, stdout, errBuf)
}
func Version() (error, int, string) {
	locker.Lock()
	defer locker.Unlock()
	errBuf := &bytes.Buffer{}
	return executeCommand([]string{"bash", "-c", "nginx version"}, stdout, errBuf)
}

// executeCommand execute linux command.
// if command exit code is 0, ignore command stderr output.
func executeCommand(command []string, stdout io.Writer, errBuf *bytes.Buffer) (error, int, string) {
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Stdout = stdout
	cmd.Stderr = errBuf
	err := cmd.Run()
	if err != nil {
		if msg, ok := err.(*exec.ExitError); ok {
			return err, msg.ExitCode(), errBuf.String()
		}
		return err, 0, errBuf.String()
	}
	return nil, 0, errBuf.String()
}

// CmdErrMsg returns the exec.Command error and command stderr output.
func CmdErrMsg(err error, exitCode int, errMsg string) error {
	if err != nil {
		return errors.New(err.Error() + ": " + errMsg)
	}
	if exitCode != 0 {
		return errors.New("exit status " + strconv.Itoa(exitCode) + ": " + errMsg)
	}
	return nil
}
