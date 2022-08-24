#!/usr/bin/env bash

configPath="k8s-loadbalancer.conf"

# nginxListenPort:          Nginx 打算监听的端口
# serviceNamespace:         要代理的 k8s service 所在的 namespace
# serviceName:              要代理的 k8s service 的名字
# servicePort:              要代理的 k8s service 对外开放的某个端口
# serviceProtocol:          要代理的 k8s service 对外开放的某个端口的协议
# 完成格式如下
#   22: gitea-repo/gitea-service:22
#   23: gitea-repo/gitea-service:22
#   53: kube-system/kube-dns:53
#   9000: default/example-go:8080

function _getServiceProtocol() {
    # local serviceNamespace=$1
    # local servicename=$2
    # local serviceProt=$3
    local count=1
    local jsonpathString=`echo jsonpath="{.spec.ports[?(@.port==${servicePort})].protocol}"`

    while true; do
        serviceProtocol=`kubectl -n ${serviceNamespace} get service ${serviceName} -o ${jsonpathString}`
        if [[ -n ${serviceProtocol} ]]; then break; fi
        if [[ ${count} -ge 3 ]]; then break; fi
        sleep 3
        (( count++ ))
    done
    if [[ -z ${serviceProtocol} ]]; then
        echo "can't get ${serviceNamespace}/${serviceName}:${servicePort} protocol, return failed."
        return 1
    fi

    serviceProtocol=${serviceProtocol,,}

}

declare -x nginxListenPort serviceNamespace serviceName servicePort serviceProtocol
while read -r serviceInfo; do
    nginxListenPort=` echo ${serviceInfo} | awk -F '[.|:|/]' '{print $1}'`
    serviceName=`     echo ${serviceInfo} | awk -F '[.|:|/]' '{print $2}'`
    serviceNamespace=`echo ${serviceInfo} | awk -F '[.|:|/]' '{print $3}'`
    servicePort=`     echo ${serviceInfo} | awk -F '[.|:|/]' '{print $4}'`
    serviceProtocol=` echo ${serviceInfo} | awk -F '[.|:|/]' '{print $5}'`
    if [[ -z ${serviceProtocol} ]]; then serviceProtocol=tcp; fi

    echo "${nginxListenPort}|${serviceName}.${serviceNamespace}:${servicePort}|${serviceProtocol}"
done < <(cat ${configPath} | \
    grep -v "^#" | \
    sed -e 's/"//g' -e s%\'%%g \
        -e 's/[[:space:]]//g' \
        -e '/^$/d' |\
    sort)
