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
	//if err == nil {
	//logrus.Error(err)
	n.err = err
	//}
}

// There are four steps will be done by Do function.
// * call TestConf() to test nginx configuration if nginx already installed.
func (n *Nginx) Do(proto protocol) bool {
	locker.Lock()
	defer locker.Unlock()
	var err error
	var changed bool

	// prepare nginx
	// it will check whether nginx config dir exist
	if err = GetCmdErrMsg(Prepare()); err != nil {
		n.setErr(err)
		return false
	}
	// install nginx if nginx not installed
	if err = GetCmdErrMsg(Install()); err != nil {
		n.setErr(err)
		return false
	}
	// enable nginx
	if err = GetCmdErrMsg(Enabled()); err != nil {
		n.setErr(err)
		return false
	}

	// generate nginx config
	if err, changed = GenerateNginxConf(); err != nil {
		n.setErr(err)
		return false
	}
	// if /etc/nginx/nginx.conf changed, test nginx config and reload nginx.
	if changed {
		// test nginx configuration
		if err = GetCmdErrMsg(TestConf()); err != nil {
			n.setErr(err)
			return false
		}
		// reload nginx
		if err = GetCmdErrMsg(Reload()); err != nil {
			n.setErr(err)
			return false
		}
	}

	// generate nginx virtual host config
	switch proto {
	case ProtocolTCP:
		if err, changed = GenerateTCPConf("", nil, 0); err != nil {
			n.setErr(err)
			return false
		}
	case ProtocolUDP:
		if err, changed = GenerateUDPConf(); err != nil {
			n.setErr(err)
			return false
		}
	case ProtocolHTTP:
		if err, changed = GenerateHTTPConf(); err != nil {
			n.setErr(err)
			return false
		}
	case ProtocolHTTPS:
		if err, changed = GenerateHTTPSConf(); err != nil {
			n.setErr(err)
			return false
		}
	default:
		n.setErr(errors.New("protocol must be 'TCP|UDP|HTTP|HTTPS'"))
		return false
	}
	// if nginx virtual host config changed, test nginx config and reload nginx.
	if changed {
		// test nginx configuration
		if err = GetCmdErrMsg(TestConf()); err != nil {
			n.setErr(err)
			return false
		}
		// reload nginx
		if err = GetCmdErrMsg(Reload()); err != nil {
			n.setErr(err)
			return false
		}
	}

	// everything is done.
	return false
}

// Prepare will do check before processing nginx.
// You should always call Prepare() before do anything to nginx
func Prepare() (error, int, string) {
	return executeCommand([]string{"bash", "-c", NGINX_PREPARE},
		logger.New().WriterLevel(logrus.DebugLevel),
		&bytes.Buffer{})
}

// Install will intall the nginx package in linux.
func Install() (error, int, string) {
	return executeCommand([]string{"bash", "-c", NGINX_INSTALL},
		logger.New().WriterLevel(logrus.DebugLevel),
		&bytes.Buffer{})
}

// Remove() will uninstall the nginx package in linux.
func Remove() (error, int, string) {
	return executeCommand([]string{"bash", "-c", NGINX_REMOVE},
		logger.New().WriterLevel(logrus.DebugLevel),
		&bytes.Buffer{})
}

// Start will start nginx daemon by systemctl.
func Start() (error, int, string) {
	return executeCommand([]string{"bash", "-c", NGINX_START},
		logger.New().WriterLevel(logrus.DebugLevel),
		&bytes.Buffer{})
}

// Stop will stop nginx daemon by systemctl.
func Stop() (error, int, string) {
	return executeCommand([]string{"bash", "-c", NGINX_STOP},
		logger.New().WriterLevel(logrus.DebugLevel),
		&bytes.Buffer{})
}

// Reload will reload nginx daemon by systemctl.
func Reload() (error, int, string) {
	return executeCommand([]string{"bash", "-c", NGINX_RELOAD},
		logger.New().WriterLevel(logrus.DebugLevel),
		&bytes.Buffer{})
}

// Restart will restart nginx daemon by systemctl.
func Restart() (error, int, string) {
	return executeCommand([]string{"bash", "-c", NGINX_RESTART},
		logger.New().WriterLevel(logrus.DebugLevel),
		&bytes.Buffer{})
}

// EnabledNow will enabled and start nginx daemon by systemctl.
func EnabledNow() (error, int, string) {
	return executeCommand([]string{"bash", "-c", NGINX_ENABLENOW},
		logger.New().WriterLevel(logrus.DebugLevel),
		&bytes.Buffer{})
}

// Enabled will enabled nginx daemon by systemctl.
func Enabled() (error, int, string) {
	return executeCommand([]string{"bash", "-c", NGINX_ENABLE},
		logger.New().WriterLevel(logrus.DebugLevel),
		&bytes.Buffer{})
}

// TestConf will test nginx configuration file.
func TestConf() (error, int, string) {
	return executeCommand([]string{"bash", "-c", NGINX_TESTCONF},
		logger.New().WriterLevel(logrus.DebugLevel),
		&bytes.Buffer{})
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
	//if len(errMsg) != 0 {
	//    return errors.New(errMsg)
	//}
	return nil
}
