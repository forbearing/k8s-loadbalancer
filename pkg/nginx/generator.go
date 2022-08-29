package nginx

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/forbearing/k8s-loadbalancer/pkg/args"
	"github.com/sirupsen/logrus"
)

// GenerateNginxConf generate /etc/nginx/nginx.conf config file.
// it will return true, if /etc/nginx/nginx.conf changed
func GenerateNginxConf() (error, bool) {
	return generateFile(nginxConfFile, TemplateNginxConf)
}

// GenerateVirtualHostConf generate /etc/nginx/sites-enabled/xxx.conf config file for proxy traffic.
func GenerateVirtualHostConf(service *Service) (error, bool) {
	var changed bool

	// if upstream host is empty, skip generate nginx config file.
	if len(args.GetUpstream()) == 0 {
		logrus.Warn("upstream is empty, skip generate nginx config")
		return nil, false
	}
	logrus.Debugf("upstream host are: %v", args.GetUpstream())

	for _, port := range service.Ports {
		upstreamName := fmt.Sprintf("%s.%s.%s", service.Namespace, service.Name, port.Name)
		var upstreamHosts strings.Builder
		for _, host := range args.GetUpstream() {
			upstreamHosts.WriteString(fmt.Sprintf("    server %s:%d;\n", host, port.NodePort))
		}
		configFile := filepath.Join(tcpConfDir, fmt.Sprintf("%s.%s", strings.ToLower(port.Protocol), upstreamName))
		configData := fmt.Sprintf(TemplateTCP, upstreamName, upstreamHosts.String(), port.Port, upstreamName, upstreamName)

		switch service.Action {
		case ActionTypeDel:
			// if action is ActionTypeDel, it means that k8s service object was deleted,
			// and we should delete the corresponding nginx configuration file.
			logrus.Debugf("remove nginx config: %s", configFile)
			if err := os.Remove(configFile); err != nil {
				logrus.Errorf("remove %s failed", err)
				return err, false
			}
			changed = true
		case ActionTypeAdd:
			// if action is ActionTypeAdd, it means that k8s service object exists.
			// we should create the corresponding nginx configuration file.
			//logrus.Debugf(configFile)
			err, isChanged := generateFile(configFile, configData)
			if err != nil {
				return err, false
			}
			if isChanged {
				changed = true
			}
		}
	}
	return nil, changed
}

// generateFile
func generateFile(configFile, configData string) (error, bool) {
	var (
		err                      error
		oldData, newData         []byte
		oldHashCode, newHashCode string
	)

	// if config file not exist, create it.
	if _, err = os.Stat(configFile); errors.Is(err, os.ErrNotExist) {
		logrus.Debugf("%s not exist, create it", configFile)
		file, err := os.Create(configFile)
		if err != nil {
			return err, false
		}
		if _, err := file.WriteString(configData); err != nil {
			return err, false
		}
		file.Close()
		return nil, true
	} else if err != nil {
		return err, false
	}

	// calculate the nginx config file hash
	if oldData, err = ioutil.ReadFile(configFile); err != nil {
		return err, false
	}
	newData = []byte(configData)
	if oldHashCode, err = genHashCode(oldData); err != nil {
		return err, false
	}
	if newHashCode, err = genHashCode(newData); err != nil {
		return err, false
	}

	logrus.Debugf("%s hash before: %s", configFile, oldHashCode)
	logrus.Debugf("%s hash after:  %s", configFile, newHashCode)

	// if config file hash not the same, generate the nginx config and overwirte it.
	if oldHashCode != newHashCode {
		logrus.Debugf("%s hash is not the same, generate it.", configFile)
		file, err := os.Create(configFile)
		if err != nil {
			return err, false
		}
		if _, err := file.WriteString(configData); err != nil {
			return err, false
		}
		file.Close()
		return nil, true
	}
	logrus.Debugf("%s hash is same, skip generate it.", configFile)
	return nil, false
}

// genHashCode
func genHashCode(data []byte) (string, error) {
	hash := sha256.New()
	if _, err := hash.Write(data); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
