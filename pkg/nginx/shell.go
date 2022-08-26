package nginx

import "fmt"

var (
	NGINX_INSTALL = `
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
	fi
	if [[ -L /etc/nginx/sites-enabled/default  ]]; then
		unlink /etc/nginx/sites-enabled/default
	fi
	;;
centos|rocky)
	if ! command -v nginx &> /dev/null; then
		yum install -y nginx
	fi; ;;
esac
`

	NGINX_REMOVE = `
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
`

	NGINX_START = `
if [[ $(systemctl is-active nginx) == "active"  ]]; then exit 0; fi
echo "systemctl start nginx"
systemctl start nginx
`
	NGINX_STOP = `
echo "systemctl stop nginx"
systemctl stop nginx
`
	NGINX_RELOAD = `
echo "systemctl reload nginx"
systemctl reload nginx
`
	NGINX_RESTART = `
echo "systemctl restart nginx"
systemctl restart nginx
`
	NGINX_ENABLE = `
if [[ $(systemctl is-enabled nginx) == "enabled"  ]]; then exit 0; fi
echo "systemctl enable nginx"
systemctl enable nginx
`
	NGINX_ENABLENOW = `
echo "systemctl enable --now nginx"
systemctl enable --now nginx
`
	NGINX_TESTCONF = `
#echo "test nginx configuration"
nginx -t
`

	NGINX_PREPARE = fmt.Sprintf(`
for dir in "%s" "%s" "%s" "%s"; do
	if [[ ! -d "$dir" ]]; then
		rm -rf "$dir"
		mkdir -p "$dir"
	fi
done
`, tcpConfDir, udpConfDir, httpConfDir, httpsConfDir)
)
