package nginx

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
	fi; ;;
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

	NGINX_START      = "systemctl start nginx"
	NGINX_STOP       = "systemctl stop nginx"
	NGINX_RESTART    = "systemctl restart nginx"
	NGINX_RELOAD     = "systemctl reload nginx"
	NGINX_ENABLED    = "systemctl enabled nginx"
	NGINX_ENABLEDNOW = "systemctl enabled --now nginx"
	NGINX_TESTCONF   = "nginx -t"
)
