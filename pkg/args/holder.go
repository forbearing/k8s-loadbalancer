package args

import "net"

var lbHolder = &holder{}

type holder struct {
	port        int
	bindAddress net.IP
	kubeconfig  string
	logLevel    string
	logFormat   string
	logFile     string
	upstream    []string
	numWorker   int
}

func GetPort() int           { return lbHolder.port }
func GetBindAddress() net.IP { return lbHolder.bindAddress }
func GetKubeconfig() string  { return lbHolder.kubeconfig }
func GetLogLevel() string    { return lbHolder.logLevel }
func GetLogFormat() string   { return lbHolder.logFormat }
func GetLogFile() string     { return lbHolder.logFile }
func GetUpstream() []string {
	var upstream []string
	for _, host := range lbHolder.upstream {
		upstream = append(upstream, host)
	}
	return upstream
}
func GetNumWorker() int { return lbHolder.numWorker }
