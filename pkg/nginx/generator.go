package nginx

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/ioutil"
	"os"

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
		if _, err := file.Write([]byte(TemplateNginxConf)); err != nil {
			return err, false
		}
		file.Close()
		return nil, true
	} else if err != nil { // os.Stat error
		return err, false
	}

	// calculate the nginx config file hash.
	var (
		err                  error
		data1, data2         []byte
		hashCode1, hashCode2 string
	)
	if data1, err = ioutil.ReadFile(nginxConfFile); err != nil {
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
		logrus.Debugf("%%s hash is not the same, generate it.", nginxConfFile)
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
	logrus.Debugf("%s hash is same, skip generate it.", nginxConfFile)
	return nil, false
}

// GenerateTCPConf generate /etc/nginx/sites-enabled/xxx.conf config file for proxy tcp traffic.
func GenerateTCPConf(upstreamName string, upstreanHost []string, listenPort int) (error, bool) {
	configFile := upstreamName + ".tcp.conf"
	_ = configFile

	// if config file not exist, create it.

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
