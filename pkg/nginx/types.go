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
	ActionTypeAdd = "ADD"
	ActionTypeDel = "DEL"
)

type Service struct {
	Action    ActionType
	Name      string
	Namespace string

	MeetType        bool
	MeetAnnotations bool

	Ports []ServicePort
}

type ServicePort struct {
	Name     string
	Port     int32
	NodePort int32
	Protocol string

	ListenPort int32
}
