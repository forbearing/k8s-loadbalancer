package nginx

import "path/filepath"

type protocol string

const (
	ProtocolTCP protocol = "TCP"
	ProtocolUDP protocol= "UDP"
	ProtocolHTTP protocol = "HTTP"
	ProtocolHTTPS protocol ="HTTPS"
)

var (
	nginxDir = "/etc/nginx"

	tcpConfDir   = filepath.Join(nginxDir, "sites-stream")
	udpConfDir   = filepath.Join(nginxDir, "sites-stream")
	httpConfDir  = filepath.Join(nginxDir, "sites-enabled")
	httpsConfDir = filepath.Join(nginxDir, "sites-enabled")

	nginxConfFile = filepath.Join(nginxDir, "nginx.conf")
)
