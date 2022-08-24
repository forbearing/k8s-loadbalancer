package nginx

import (
	"io"
	"os/exec"

	"github.com/sirupsen/logrus"
)

var (
	stdout = logrus.New().Writer()
	stderr = logrus.New().WriterLevel(logrus.ErrorLevel)
)

// Start will start nginx daemon by systemctl.
func Start() error {
	return executeCommand([]string{"systemctl", "start", "nginx"}, stdout, stderr)
}

// Stop will stop nginx daemon by systemctl.
func Stop() error {
	return executeCommand([]string{"systemctl", "stop", "nginx"}, stdout, stderr)
}

// Restart will restart nginx daemon by systemctl.
func Restart() error {
	return executeCommand([]string{"systemctl", "restart", "nginx"}, stdout, stderr)
}

// Reload will reload nginx daemon by systemctl.
func Reload() error {
	return executeCommand([]string{"systemctl", "reload", "nginx"}, stdout, stderr)
}

// EnabledNow will enabled and start nginx daemon by systemctl.
func EnabledNow() error {
	return executeCommand([]string{"systemctl", "enabled", "--now", "nginx"}, stdout, stderr)
}

// Enabled will enabled nginx daemon by systemctl.
func Enabled() error {
	return executeCommand([]string{"systemctl", "enabled", "nginx"}, stdout, stderr)
}

// TestConf will test nginx configuration file.
func TestConf() error {
	return executeCommand([]string{"nginx", "-t"}, stdout, stderr)
}

// executeCommand execute linux command.
func executeCommand(command []string, stdout, stderr io.Writer) error {
	if _, err := exec.LookPath(command[0]); err != nil {
		return err
	}
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	if err := cmd.Wait(); err != nil {
		return err
	}
	return nil
}
