package nginx

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/ioutil"
	"os"

	"github.com/sirupsen/logrus"
)

var (
	nginxConfFile = "/etc/nginx/nginx.conf"
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
		if _, err := file.Write([]byte(TemplateNginxConf)); err != nil {
			return err, false
		}
		file.Close()
	} else if err != nil { // os.Stat error
		logrus.Error("os.State error")
		return err, false
	}

	// calculate the nginx config file hash.
	var (
		err                  error
		data1, data2         []byte
		hashCode1, hashCode2 string
	)
	if data1, err = ioutil.ReadFile(nginxConfFile); err != nil {
		logrus.Error("ioutil.ReadFile error", err)
		return err, false
	}
	data2 = []byte(TemplateNginxConf)
	if hashCode1, err = genHashCode(data1); err != nil {
		return err, false
	}
	if hashCode2, err = genHashCode(data2); err != nil {
		return err, false
	}

	logrus.Debugf("%s hash before: %s", nginxConfFile, hashCode1)
	logrus.Debugf("%s hash after:  %s", nginxConfFile, hashCode2)

	// if nginx config file hash code not same, generate the nginx config and overwirte it.
	if hashCode1 != hashCode2 {
		logrus.Debug("nginx config hash is not the same, generate it.")
		file, err := os.Create(nginxConfFile)
		if err != nil {
			return err, false
		}
		if _, err := file.Write(data2); err != nil {
			return err, false
		}
		file.Close()
		return nil, true
	}
	logrus.Debug("nginx config file hash is same, skip generate it.")
	return nil, false
}

// GenerateTCPConf generate /etc/nginx/sites-enabled/xxx.conf config file for proxy tcp traffic.
func GenerateTCPConf() {}

// GenerateUDPConf generate /etc/nginx/sites-enabled/xxx.conf config file for proxy udp traffic.
func GenerateUDPConf() {}

// GenerateHTTPConf generate /etc/nginx/sites-enabled/xxx.conf config file for proxy http traffic.
func GenerateHTTPConf() {}

// GenerateHTTPConf generate /etc/nginx/sites-enabled/xxx.conf config file for proxy https traffic.
func GenerateHTTPSConf() {}

func genHashCode(data []byte) (string, error) {
	hash := sha256.New()
	if _, err := hash.Write(data); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
