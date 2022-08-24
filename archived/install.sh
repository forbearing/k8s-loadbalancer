#!/usr/bin/env bash

os=$(uname -s)
if [[  $os != "Linux"  ]]; then
    echo "Not Support OS: $os"
    exit 1
fi

# uninstall k8s-loadbalancer
if [[ $1 == "-u" ]]; then
    systemctl stop k8s-loadbalancer &> /dev/null
    rm -rf /etc/k8s-loadbalancer
    rm -rf /usr/local/bin/k8s-loadbalancer.sh
    rm -rf /usr/local/bin/kubectl
    systemctl disable --now k8s-loadbalancer
    rm -rf /lib/systemd/system/k8s-loadbalancer.service
    exit 0
fi

# copy file
cp k8s-loadbalancer.sh /usr/local/bin/
chmod u+x /usr/local/bin/k8s-loadbalancer.sh
mkdir -p /etc/k8s-loadbalancer
cp -r conf.d/* /etc/k8s-loadbalancer/
cp k8s-loadbalancer.conf /etc/k8s-loadbalancer/
cp k8s-loadbalancer.service /lib/systemd/system/
cp bin/kubectl-linux-1.23.1 /usr/local/bin/kubectl
chmod u+x /usr/local/bin/kubectl
mkdir -p ~/.kube

# enabled k8s-loadbalancer
systemctl daemon-reload
systemctl enable --now k8s-loadbalancer
