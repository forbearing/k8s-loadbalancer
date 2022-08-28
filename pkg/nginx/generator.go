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
	// if nginx config file not exist, generate it.
	if _, err := os.Stat(nginxConfFile); errors.Is(err, os.ErrNotExist) {
		file, err := os.Create(nginxConfFile)
		if err != nil {
			return err, false
		}
		if _, err := file.WriteString(TemplateNginxConf); err != nil {
			return err, false
		}
		file.Close()
		return nil, true
	} else if err != nil { // os.Stat error
		return err, false
	}

	// calculate the nginx config file hash.
	var (
		err                      error
		oldData, newData         []byte
		oldHashCode, newHashCode string
	)
	if oldData, err = ioutil.ReadFile(nginxConfFile); err != nil {
		return err, false
	}
	newData = []byte(TemplateNginxConf)
	if oldHashCode, err = genHashCode(oldData); err != nil {
		return err, false
	}
	if newHashCode, err = genHashCode(newData); err != nil {
		return err, false
	}
	logrus.Debugf("%s hash before: %s", nginxConfFile, oldHashCode)
	logrus.Debugf("%s hash after:  %s", nginxConfFile, newHashCode)

	// if nginx config file hash code not same, generate the nginx config and overwirte it.
	if oldHashCode != newHashCode {
		logrus.Debugf("%s hash is not the same, generate it.", nginxConfFile)
		file, err := os.Create(nginxConfFile)
		if err != nil {
			return err, false
		}
		if _, err := file.Write(newData); err != nil {
			return err, false
		}
		file.Close()
		return nil, true
	}
	logrus.Debugf("%s hash is same, skip generate it.", nginxConfFile)
	return nil, false
}

// GenerateTCPConf generate /etc/nginx/sites-enabled/xxx.conf config file for proxy tcp traffic.
func GenerateVirtualHostConf(service *Service) (error, bool) {
	upstreamName := fmt.Sprintf("%s.%s.%s", service.Namespace, service.Name, service.Ports[0].Name)

	var hostRecord strings.Builder
	logrus.Debugf("upstream host are: %v", args.GetUpstream())
	for _, host := range args.GetUpstream() {
		hostRecord.WriteString(fmt.Sprintf("    server %s:%d;\n", host, service.Ports[0].NodePort))
	}
	configData := fmt.Sprintf(TemplateTCP, upstreamName, hostRecord.String(), service.Ports[0].Name, upstreamName, upstreamName)
	configFile := filepath.Join(tcpConfDir, "tcp."+upstreamName)

	// if action is ActionDel, it means that k8s service object was deleted,
	// and we should delete the corresponding nginx configuration file.

	// if action is ActionAdd, it means that k8s service object exists.
	// we should create the corresponding nginx configuration file.
	//
	// if config file not exist, create it.
	if _, err := os.Stat(configFile); errors.Is(err, os.ErrNotExist) {
		logrus.Debugf("create nginx config: %s", configFile)
		file, err := os.Create(configFile)
		if err != nil {
			return err, false
		}
		if _, err := file.WriteString(configData); err != nil {
			return err, false
		}
		file.Close()
		return nil, true
	} else if err != nil { // os.Stat() error
		return err, false
	}

	logrus.Debugf("nginx config file: %s already exist, skip generate", configFile)
	// calculate the nginx config file hash
	var (
		err                      error
		oldData, newData         []byte
		oldHashCode, newHashCode string
	)
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
		if _, err := file.Write(newData); err != nil {
			return err, false
		}
		file.Close()
		return nil, true
	}
	logrus.Debugf("%s hash is same, skip generate it.", configFile)
	return nil, false
}

// GenerateUDPConf generate /etc/nginx/sites-enabled/xxx.conf config file for proxy udp traffic.
func GenerateUDPConf() (error, bool) {
	return nil, false
}

// GenerateHTTPConf generate /etc/nginx/sites-enabled/xxx.conf config file for proxy http traffic.
func GenerateHTTPConf() (error, bool) {
	return nil, false
}

// GenerateHTTPConf generate /etc/nginx/sites-enabled/xxx.conf config file for proxy https traffic.
func GenerateHTTPSConf() (error, bool) {
	return nil, false
}

func genHashCode(data []byte) (string, error) {
	hash := sha256.New()
	if _, err := hash.Write(data); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
