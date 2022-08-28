package nginx

import "path/filepath"

var (
	nginxDir = "/etc/nginx"

	tcpConfDir   = filepath.Join(nginxDir, "sites-stream")
	udpConfDir   = filepath.Join(nginxDir, "sites-stream")
	httpConfDir  = filepath.Join(nginxDir, "sites-enabled")
	httpsConfDir = filepath.Join(nginxDir, "sites-enabled")

	nginxConfFile = filepath.Join(nginxDir, "nginx.conf")
)

type Protocol string

const (
	ProtocolTCP   Protocol = "TCP"
	ProtocolUDP   Protocol = "UDP"
	ProtocolHTTP  Protocol = "HTTP"
	ProtocolHTTPS Protocol = "HTTPS"
)

type ActionType string

const (
	ActionTypeAdd    = "ADD"
	ActionTypeDelete = "DELETE"
)

type Service struct {
	Action    ActionType
	Namespace string
	Name      string
	Ports     []Port
}

type Port struct {
	Protocol string
	Name     string
	NodePort int32
}
