package nginx

import (
	"bytes"
	"errors"
	"io"
	"os/exec"
	"strconv"
	"sync"

	"github.com/forbearing/k8s-loadbalancer/pkg/logger"
	"github.com/sirupsen/logrus"
)

var (
	locker sync.RWMutex
)

type Nginx struct {
	err error
}

// Err returns the first errors that was encountered by the Do() function.
func (n *Nginx) Err() error {
	return n.err
}

// setErr records the first error encountered by the Do() function.
func (n *Nginx) setErr(err error) {
	if err == nil {
		n.err = err
	}
}

// There are four steps will be done by Do function.
// * call TestConf() to test nginx configuration if nginx already installed.
// *
func (n *Nginx) Do() bool {
	locker.Lock()
	defer locker.Unlock()

	// install nginx if nginx not installed
	if err := GetCmdErrMsg(Install()); err != nil {
		n.setErr(err)
		return false
	}

	// test nginx configuration
	if err := GetCmdErrMsg(TestConf()); err != nil {
		n.setErr(err)
		return false
	}

	// enable nginx
	if err := GetCmdErrMsg(Enabled()); err != nil {
		n.setErr(err)
		return false
	}

	// reload nginx
	if err := GetCmdErrMsg(Reload()); err != nil {
		n.setErr(err)
		return false
	}

	// everything is done.
	return false
}

// Install will intall the nginx package in linux.
func Install() (error, int, string) {
	stdout := logger.New().WriterLevel(logrus.DebugLevel)
	errBuf := &bytes.Buffer{}
	return executeCommand([]string{"bash", "-c", NGINX_INSTALL}, stdout, errBuf)
}

// Remove() will uninstall the nginx package in linux.
func Remove() (error, int, string) {
	stdout := logger.New().WriterLevel(logrus.DebugLevel)
	errBuf := &bytes.Buffer{}
	return executeCommand([]string{"bash", "-c", NGINX_REMOVE}, stdout, errBuf)
}

// Start will start nginx daemon by systemctl.
func Start() (error, int, string) {
	stdout := logger.New().WriterLevel(logrus.DebugLevel)
	errBuf := &bytes.Buffer{}
	return executeCommand([]string{"bash", "-c", NGINX_START}, stdout, errBuf)
}

// Stop will stop nginx daemon by systemctl.
func Stop() (error, int, string) {
	stdout := logger.New().WriterLevel(logrus.DebugLevel)
	errBuf := &bytes.Buffer{}
	return executeCommand([]string{"bash", "-c", NGINX_STOP}, stdout, errBuf)
}

// Reload will reload nginx daemon by systemctl.
func Reload() (error, int, string) {
	stdout := logger.New().WriterLevel(logrus.DebugLevel)
	errBuf := &bytes.Buffer{}
	return executeCommand([]string{"bash", "-c", NGINX_RELOAD}, stdout, errBuf)
}

// Restart will restart nginx daemon by systemctl.
func Restart() (error, int, string) {
	stdout := logger.New().WriterLevel(logrus.DebugLevel)
	errBuf := &bytes.Buffer{}
	return executeCommand([]string{"bash", "-c", NGINX_RESTART}, stdout, errBuf)
}

// EnabledNow will enabled and start nginx daemon by systemctl.
func EnabledNow() (error, int, string) {
	stdout := logger.New().WriterLevel(logrus.DebugLevel)
	errBuf := &bytes.Buffer{}
	return executeCommand([]string{"bash", "-c", NGINX_ENABLENOW}, stdout, errBuf)
}

// Enabled will enabled nginx daemon by systemctl.
func Enabled() (error, int, string) {
	stdout := logger.New().WriterLevel(logrus.DebugLevel)
	errBuf := &bytes.Buffer{}
	return executeCommand([]string{"bash", "-c", NGINX_ENABLE}, stdout, errBuf)
}

// TestConf will test nginx configuration file.
func TestConf() (error, int, string) {
	stdout := logger.New().WriterLevel(logrus.DebugLevel)
	errBuf := &bytes.Buffer{}
	return executeCommand([]string{"bash", "-c", NGINX_TESTCONF}, stdout, errBuf)
}

// Version get nginx version
func Version() (error, int, string) {
	stdout := logger.New().WriterLevel(logrus.DebugLevel)
	errBuf := &bytes.Buffer{}
	return executeCommand([]string{"whoami"}, stdout, errBuf)
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

// GetCmdErrMsg returns the exec.Command error and command stderr output.
func GetCmdErrMsg(err error, exitCode int, errMsg string) error {
	if err != nil {
		return errors.New(err.Error() + ": " + errMsg)
	}
	if exitCode != 0 {
		return errors.New("exit status " + strconv.Itoa(exitCode) + ": " + errMsg)
	}
	return nil
}
