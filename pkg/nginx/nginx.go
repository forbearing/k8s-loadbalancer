package nginx

import (
	"io"
	"os/exec"
	"sync"

	"github.com/sirupsen/logrus"
)

var (
	stdout = logrus.New().Writer()
	stderr = logrus.New().WriterLevel(logrus.ErrorLevel)
)
var (
	lock sync.Mutex
)

// Install will intall the nginx package in linux.
func Install() error {
	lock.Lock()
	defer lock.Unlock()

	return executeCommand([]string{"bash", "-c", `
source /etc/os-release
linuxID=$ID
linuxMajorVersion=$( echo $VERSION | awk -F'[.| ]' '{print $1}' )
linuxCodeName=$VERSION_CODENAME
[ -f /etc/lsb-release ] &&  \
	linuxMinorVersion=$(cat /etc/lsb-release  | awk -F'=' '/DISTRIB_RELEASE/ {print $2}' | awk -F'.'  '{print $2}')
[ -f /etc/system-release ] && \
	linuxMinorVersion=$(cat /etc/system-release | awk '{print $4}' | awk -F'.' '{print $2}')

case $linuxID in 
ubuntu|debian)
	if ! command -v nginx &> /dev/null; then
		apt-get update
		apt-get install -y nginx
	fi; ;;
centos|rocky)
	if ! command -v nginx &> /dev/null; then
		yum install -y nginx
	fi; ;;
esac

	`}, stdout, stderr)
}

// Uninstall() will uninstall the nginx package in linux.
func Uninstall() error {
	lock.Lock()
	defer lock.Unlock()

	return executeCommand([]string{"bash", "-c", `
source /etc/os-release
linuxID=$ID
linuxMajorVersion=$( echo $VERSION | awk -F'[.| ]' '{print $1}' )
linuxCodeName=$VERSION_CODENAME
[ -f /etc/lsb-release ] &&  \
	linuxMinorVersion=$(cat /etc/lsb-release  | awk -F'=' '/DISTRIB_RELEASE/ {print $2}' | awk -F'.'  '{print $2}')
[ -f /etc/system-release ] && \
	linuxMinorVersion=$(cat /etc/system-release | awk '{print $4}' | awk -F'.' '{print $2}')

case $linuxID in 
ubuntu|debian)
	if command -v nginx &> /dev/null; then
		systemctl disable --now nginx &> /dev/null
		apt-get purge -y nginx*
	fi; ;;
centos|rocky)
	if command -v nginx &> /dev/null; then
		systemctl disable --now nginx &> /dev/null
		yum remove -y nginx*
	fi; ;;
esac
	`}, stdout, stderr)
}

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
