name="k8s-loadbalancer"

install:
	go build -o /usr/local/bin/k8s-loadbalancer .
	cat > /lib/systemd/system/k8s-loadbalancer.service <<EOF
		[Unit]
		Description=k8s loadbalancer
		Documentation=https://github.com/forbearing/k8s-loadbalancer
		After=network.target nginx.service

		[Service]
		Type=simple
		ExecStart=/usr/local/bin/k8s-loadbalancer
		ExecStop=/bin/kill -s HUP $MAINPID
		RestartSec=5s
		User=root

		[Install]
		WantedBy=multi-user.target
		EOF
	#systemctl enable --now k8s-loadbalancer
