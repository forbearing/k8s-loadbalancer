#!/usr/bin/env bash

# Copyright 2021 hybfkuf 
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

EXIT_SUCCESS=0
EXIT_FAILURE=1
RETURN_SUCCESS=0
RETURN_FAILURE=1

TRACE=true
DEBUG=true
WARN=true

LOOP_TIME="30"
configDir="/etc/k8s-loadbalancer"
configPath="${configDir}/k8s-loadbalancer.conf"
NGINX_CONFIG_AVAILABLE_DIR="/etc/nginx/sites-available"
NGINX_CONFIG_ENABLED_HTTP_DIR="/etc/nginx/sites-enabled"
NGINX_CONFIG_ENABLED_HTTPS_DIR="/etc/nginx/sites-enabled"
NGINX_CONFIG_ENABLED_TCP_DIR="/etc/nginx/sites-stream"
NGINX_CONFIG_ENABLED_UDP_DIR="/etc/nginx/sites-stream"
CHANNEL="/tmp/k8s-loadbalancer"

#declare -ax upstreamIP upstreamAvailIP FIREWAL_WHITELIST
upstreamIP=(
    10.240.3.21
    10.240.3.22
    10.240.3.23)
upstreamAvailIP=()
FIREWALL_WHITELIST=(10.240.0.100)

# ip_hash           哈希算法
# fair              按后端服务器响应时间来分配请求，响应时间短的优先分配
# url_hash          按访问url的hash结果来分配请求。使每个url定向到同一个（对应的）后端服务器，后端服务器为缓存时比较有效。
# least_conn        最小连接数
UPSTREAM_LOADBALANCER="least_conn"


# nginxListenPort:          Nginx 打算监听的端口
# serviceNamespace:         要代理的 k8s service 所在的 namespace
# serviceName:              要代理的 k8s service 的名字
# servicePort:              要代理的 k8s service 对外开放的某个端口
# serviceProtocol:          要代理的 k8s service 对外开放的某个端口的协议
# serviceNodeport           要代理的 k8s service 对外开放的某个端口对应的 nodeport
declare -x nginxListenPort serviceNamespace serviceName servicePort serviceProtocol serviceNodeport


function _test_var(){
    [[ $DEBUG ]] && {   # ========== BEGIN DEBUG
        echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- nginxListenPort:   ${nginxListenPort}"
        echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- serviceName:       ${serviceName}"
        echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- servicePort:  ${servicePort}"
        echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- serviceNamespace:  ${serviceNamespace}"
        echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- serviceProtocol:   ${serviceProtocol}"
        echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- serviceNodeport:   ${serviceNodeport}"
    }                   # ========== END DEBUG
}

function 0_prepare_k8s() {
    # 检查 kubectl 工具是否存在，否则退出
    if ! command -v kubectl &> /dev/null; then
        echo "kubectl: command not found, exit..."; exit $EXIT_FAILURE; fi

    # 检查是否能连接到 k8s，否则退出
    local count=1
    local k8s_ok=""
    while true; do
        kubectl get node &> /dev/null
        local kubectl_rc=$?
        if [[ $kubectl_rc -eq 0 ]]; then k8s_ok="true"; break; fi
        if [[ ${count} -gt 3 ]]; then break; fi
        sleep 3; (( count++ ))
    done
    if [[ ${k8s_ok} != "true" ]]; then
        echo "can't connect k8s cluter, exit failed."; 
        exit $EXIT_FAILURE; fi
}

function 0_prepare_nginx() {
    # 检查相关目录和文件是否存在，否则创建
    if [[ ! -d ${configDir} ]]; then
        echo "k8s-loadbalancer configure directory: ${configDir} not found, exit failed."
        exit $EXIT_FAILURE; fi
    if [[ ! -f ${configPath} ]]; then
        echo "k8s-loadbalancer configure file: ${configPath} not found, exit failed."
        exit $EXIT_FAILURE; fi
    # 如果没有, 则创建命名管道。管道用来进程间通信
    if  [[ ! -p ${CHANNEL} ]]; then mkfifo ${CHANNEL}; fi


    # 检查是否是支持的 Linux 系统
    #   ubuntu 20 code name: focal
    #   ubuntu 18 code name: bionic
    #   debian 11 code name: bullseye
    #   deiban 10 coce name: buster
    source /etc/os-release
    local linux_id linux_version
    linux_id=$ID
    linux_version=$( echo $VERSION | awk -F'[.| ]' '{print $1}' )

    # 检查 nginx 是否安装，否则就安装
    # 拷贝自定义的 nginx 主配置文件 /etc/nginx/nginx.conf
    # nginx
    #   deb http://nginx.org/packages/mainline/debian/ stretch nginx
    #   deb http://mirrors.163.com/nginx/packages/mainline/debian/ stretch nginx
    #   curl -fsSL http://nginx.org/keys/nginx_signing.key | apt-key add - 
    #   curl -fsSL http://mirrors.163.com/nginx/keys/nginx_signing.key | apt-key add -
    case $linux_id in 
    ubuntu)
        if ! command -v nginx &> /dev/null; then
            export DEBIAN_FRONTEND=noninteractive
            add-apt-repository -y ppa:nginx/stable
            apt-get update -y
            apt-get install -y nginx
            unlink /etc/nginx/sites-enabled/default &> /dev/null
            cp -f /etc/nginx/nginx.conf /etc/nginx/nginx.conf.$(date +%Y%m%d%H%M%S)
            cp -f ${configDir}/nginx.conf /etc/nginx/nginx.conf
            systemctl enable --now nginx
        fi ;;
    debian)
        if ! command -v nginx &> /dev/null; then
            export DEBIAN_FRONTEND=noninteractive
            apt-get update -y
            apt-get install -y curl gnupg netcat ufw
            cp -f /etc/apt/sources.list /etc/apt/sources.list.$(date +%Y%m%d%H%M%S)
            sed -i '/nginx/d' /etc/apt/sources.list
            echo "deb http://nginx.org/packages/mainline/debian/ stretch nginx" >> /etc/apt/sources.list
            curl -fsSL http://nginx.org/keys/nginx_signing.key | apt-key add -
            apt-get update -y
            apt-get install -y nginx
            cp -f /etc/nginx/nginx.conf /etc/nginx/nginx.conf.$(date +%Y%m%d%H%M%S)
            cp -f ${configDir}/nginx.conf /etc/nginx/nginx.conf
            mv /etc/nginx/conf.d/default.conf /etc/nginx/conf.d/default.conf.bak &> /dev/null
            systemctl enable --now nginx
        fi ;;
    *)
        echo "Not Support Linux: ${linux_id}"
        exit $EXIT_FAILURE
        ;;
    esac

    if [[ ! -d ${NGINX_CONFIG_AVAILABLE_DIR} ]]; then
        echo "create directory ${NGINX_CONFIG_AVAILABLE_DIR}."
        mkdir -p ${NGINX_CONFIG_AVAILABLE_DIR}; fi
    if [[ ! -d ${NGINX_CONFIG_ENABLED_HTTP_DIR} ]]; then
        echo "create directory ${NGINX_CONFIG_ENABLED_HTTP_DIR}."
        mkdir -p ${NGINX_CONFIG_ENABLED_HTTP_DIR}; fi
    if [[ ! -d ${NGINX_CONFIG_ENABLED_HTTPS_DIR} ]]; then
        echo "create directory ${NGINX_CONFIG_ENABLED_HTTPS_DIR}."
        mkdir -p ${NGINX_CONFIG_ENABLED_HTTPS_DIR}; fi
    if [[ ! -d ${NGINX_CONFIG_ENABLED_TCP_DIR} ]]; then
        echo "create directory ${NGINX_CONFIG_ENABLED_TCP_DIR}.";
        mkdir -p ${NGINX_CONFIG_ENABLED_TCP_DIR}; fi
    if [[ ! -d ${NGINX_CONFIG_ENABLED_UDP_DIR} ]]; then
        echo "create directory ${NGINX_CONFIG_ENABLED_UDP_DIR}."; fi
}


# Capture INT, TERM, QUIT singal
function singal_handler {
    for SIG in "$@"; do
        case $SIG in
        "INT")      # ctrl-c to stop this script, exit success.
            trap "echo Interrupt by User, exit...; exit $EXIT_SUCCESS" $SIG ;;
        "TERM")     # kill command to stop this script, exit failed
            trap "echo Killed by User, exit...; exit $EXIT_FAILURE" $SIG ;;
        "QUIT")     # systemd send singal to stop this script, exit success.
            trap "echo Finished...; exit $EXIT_SUCCESS" $SIG ;;
        "PIPE")     # ignore PIPE singal.
            trap "" $SIG ;;
        esac
    done
}


function service_handler() {
    # 检查如下变量, 如果变量为空，则直接退出
    #   nginxListenPort, serviceName, serviceNamespace, servicePort serviceProtocol
    if [[ -z ${nginxListenPort} ]]; then
        [[ $WARN ]] && echo "[WARNNING ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}]: variable nginxListenPort is empty, skip."
        return $EXIT_FAILURE; fi
    if [[ -z $serviceName ]]; then
        [[ $WARN ]] && echo "[WARNNING ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}]: variable serviceName is empty, skip."
        return $EXIT_FAILURE; fi
    if [[ -z $serviceNamespace ]]; then
        [[ $WARN ]] && echo "[WARNNING ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}]: variable serviceNamespace is empty, skip."
        return $EXIT_FAILURE; fi
    if [[ -z $servicePort ]]; then
        [[ $WARN ]] && echo "[WARNNING ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}]: variable servicePort is empty, skip."
        return $EXIT_FAILURE; fi
    if [[ -z $serviceProtocol} ]]; then
        [[ $WARN ]] && echo "[WARNNING ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}]: variable serviceProtocol is empty, skip."
        return $EXIT_FAILURE; fi


    # 循环3次，获取 k8s service nodeport
    local count=1
    while true; do
        local jsonpathString=$(echo "{.spec.ports[?(@.port==${servicePort})].nodePort}")
        serviceNodeport=$(kubectl -n ${serviceNamespace} get service ${serviceName} -o jsonpath=${jsonpathString})
        local commandLine="kubectl -n ${serviceNamespace} get service ${serviceName} -o jsonpath=\'${jsonpathString}\'"
        if [[ -n ${serviceNodeport} ]]; then break; fi
        if [[ ${count} -ge 3 ]]; then break; fi
        sleep 3; (( count++ ))
    done

    [[ $DEBUG ]] && {   #========== BEGIN DEBUG
        echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- command line:     ${commandLine}"
        echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- nginxListenPort:  ${nginxListenPort}"
        echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- serviceName:      ${serviceName}"
        echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- serviceNamespace: ${serviceNamespace}"
        echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- servicePort:      ${servicePort}"
        echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- serviceProtocol:  ${serviceProtocol}"
        echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- serviceNodeport:  ${serviceNodeport}"
        echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- ${nginxListenPort}|${serviceProtocol}|${serviceName}.${serviceNamespace}:${servicePort}"
    }                   #========== END DEBUG
}

function nginx_handler() {
    # 检查变量, 如果变量为空, 则跳过处理
    #   nginxListenPort, serviceName, serviceNamespace, servicePort serviceProtocol serviceNodeport
    if [[ -z ${nginxListenPort} ]]; then
        [[ $WARN ]] && echo "[WARNNING ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}]: variable nginxListenPort is empty, skip."
        return $EXIT_FAILURE; fi
    if [[ -z $serviceName ]]; then
        [[ $WARN ]] && echo "[WARNNING ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}]: variable serviceName is empty, skip."
        return $EXIT_FAILURE; fi
    if [[ -z $serviceNamespace ]]; then
        [[ $WARN ]] && echo "[WARNNING ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}]: variable serviceNamespace is empty, skip."
        return $EXIT_FAILURE; fi
    if [[ -z $servicePort ]]; then
        [[ $WARN ]] && echo "[WARNNING ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}]: variable servicePort is empty, skip."
        return $EXIT_FAILURE; fi
    if [[ -z $serviceProtocol} ]]; then
        [[ $WARN ]] && echo "[WARNNING ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}]: variable serviceProtocol is empty, skip."
        return $EXIT_FAILURE; fi
    if [[ -z ${serviceNodeport} ]]; then
        [[ $WARN ]] && echo "[WARNNING ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}]: variable serviceNodeport is empty, skip."
        return $EXIT_FAILURE; fi


    # 1. 简单检查哪一些上游的 k8s 节点是可用的，如果是可用的就加入到 upstreamAvailIP 数组中
    local upstreamAvailIP=()
    for ip in "${upstreamIP[@]}"; do
        case ${serviceProtocol} in
        tcp|http|https)
            [[ $DEBUG ]] && echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- nc -vz ${ip} ${serviceNodeport}"
            nc -vz ${ip} ${serviceNodeport} &> /dev/null 
            if [[ $? -eq 0 ]]; then upstreamAvailIP+=($ip); fi ;;
        udp)
            [[ $DEBUG ]] && echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- nc -u -vz ${ip} ${serviceNodeport}"
            nc -u -vz ${ip} ${serviceNodeport} &> /dev/null
            if [[ $? -eq 0 ]]; then upstreamAvailIP+=($ip); fi ;;
        *)  echo "[ERROR ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] not support service protocol: ${serviceProtocol}, exit failed." && exit $EXIT_FAILURE
        esac
    done

    # 2. 如果没有任何可用的上游 k8s 节点，则跳过处理
    if [[ ${#upstreamAvailIP[@]} -eq 0 ]]; then
        [[ $WARN ]] && echo "[WARNNING ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}]: upstream no available ip, nginx will not add config for ${serviceName}."
        return ${EXIT_FAILURE}; fi
    upstreamAvailIP=( "${upstreamAvailIP[@]}" )

    # 3. 计算 nginx 配置文件 hash 值
    local oldNginxConfigHash
    oldNginxConfigHash=$(find ${NGINX_CONFIG_ENABLED_TCP_DIR} \
                        ${NGINX_CONFIG_ENABLED_UDP_DIR} \
                        ${NGINX_CONFIG_ENABLED_HTTP_DIR} \
                        ${NGINX_CONFIG_ENABLED_HTTPS_DIR} \
                        -type l ! -name "*.swp" -exec md5sum {} \; | \
                        sort -u -k2 | md5sum | awk '{print $1}')

    # 4. 处理 nginx 配置文件模版
    cp -f ${configDir}/template-${serviceProtocol} \
        ${NGINX_CONFIG_AVAILABLE_DIR}/${serviceName}.${serviceNamespace}:${servicePort}.${serviceProtocol}
    local serverString=""
    #TODO: 处理一下 upstreamAvailIP 中的 IP 顺序
    for (( i=0; i<${#upstreamAvailIP[@]}; i++ )); do
        serverString="    server ${upstreamAvailIP[i]}:${serviceNodeport};"
        sed -i "/^upstream/a\\${serverString}"  \
            ${NGINX_CONFIG_AVAILABLE_DIR}/${serviceName}.${serviceNamespace}:${servicePort}.${serviceProtocol}
        [[ $DEBUG ]] && echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- nginx upstream server:  ${serverString}"
    done
    sed -i "s%#LISTEN_PORT#%${nginxListenPort}%g" \
        "${NGINX_CONFIG_AVAILABLE_DIR}/${serviceName}.${serviceNamespace}:${servicePort}.${serviceProtocol}"
    sed -i "s%#UPSTREAM_NAME#%${serviceName//-/_}_${serviceNamespace//-/_}_${servicePort}_${serviceProtocol}%g" \
        "${NGINX_CONFIG_AVAILABLE_DIR}/${serviceName}.${serviceNamespace}:${servicePort}.${serviceProtocol}"
    sed -i "s%#ACCESS_LOG#%${serviceName//_/-}_${serviceNamespace//-/_}_${servicePort}_${serviceProtocol}%g" \
        "${NGINX_CONFIG_AVAILABLE_DIR}/${serviceName}.${serviceNamespace}:${servicePort}.${serviceProtocol}"

    # 5. 启用 nginx 配置文件
    case ${serviceProtocol} in
    tcp)   ln -sf "${NGINX_CONFIG_AVAILABLE_DIR}/${serviceName}.${serviceNamespace}:${servicePort}.${serviceProtocol}" ${NGINX_CONFIG_ENABLED_TCP_DIR} ;;
    udp)   ln -sf "${NGINX_CONFIG_AVAILABLE_DIR}/${serviceName}.${serviceNamespace}:${servicePort}.${serviceProtocol}" ${NGINX_CONFIG_ENABLED_UDP_DIR} ;;
    http)  ln -sf "${NGINX_CONFIG_AVAILABLE_DIR}/${serviceName}.${serviceNamespace}:${servicePort}.${serviceProtocol}" ${NGINX_CONFIG_ENABLED_HTTP_DIR} ;;
    https) ln -sf "${NGINX_CONFIG_AVAILABLE_DIR}/${serviceName}.${serviceNamespace}:${servicePort}.${serviceProtocol}" ${NGINX_CONFIG_ENABLED_HTTPS_DIR} ;;
    *)     echo "[ERROR ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] not support service protocol: ${serviceProtocol}, exit failed." && exit $EXIT_FAILURE
    esac

    # 6. 再次计算 nginx 配置文件哈希值
    local newNginxConfigHash
    newNginxConfigHash=$(find ${NGINX_CONFIG_ENABLED_TCP_DIR} \
                        ${NGINX_CONFIG_ENABLED_UDP_DIR} \
                        ${NGINX_CONFIG_ENABLED_HTTP_DIR} \
                        ${NGINX_CONFIG_ENABLED_HTTPS_DIR} \
                        -type l ! -name "*.swp" -exec md5sum {} \; | \
                        sort -u -k2 | md5sum | awk '{print $1}')
    [[ $DEBUG ]] && echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} $LINENO] -- nginx config hash (before enabled ${serviceName}): ${oldNginxConfigHash}"
    [[ $DEBUG ]] && echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} $LINENO] -- nginx config hash (after enabled ${serviceName}):  ${newNginxConfigHash}"

    # 7. 如果 nginx -t 失败，则把刚才启用了的 nginx 配置文件关闭
    nginx -t -q
    local nginxRelodRC=$?
    if [[ ${nginxRelodRC} -ne 0 ]]; then
        [[ $WARN ]] && echo "[WARNNING ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- nginx config test failed. skip."
        case ${serviceProtocol} in
        tcp)   unlink "${NGINX_CONFIG_ENABLED_TCP_DIR}/${serviceName}.${serviceNamespace}:${servicePort}.${serviceProtocol}" ;;
        udp)   unlink "${NGINX_CONFIG_ENABLED_UDP_DIR}/${serviceName}.${serviceNamespace}:${servicePort}.${serviceProtocol}" ;;
        http)  unlink "${NGINX_CONFIG_ENABLED_HTTP_DIR}/${serviceName}.${serviceNamespace}:${servicePort}.${serviceProtocol}" ;;
        https) unlink "${NGINX_CONFIG_ENABLED_HTTPS_DIR}/${serviceName}.${serviceNamespace}:${servicePort}.${serviceProtocol}" ;;
        *)     echo "[ERROR ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] not support service protocol: ${serviceProtocol}, exit failed." && exit $EXIT_FAILURE
        esac
        return $RETURN_FAILURE
    fi

    # 8. 比对两次 nginx 配置文件 hash 值，如果 nginx 配置文件 hash 变化，则 reload nginx
    if [[ ${nginxRelodRC} -eq 0  &&  "${oldNginxConfigHash}" != "${newNginxConfigHash}" ]]; then 
        [[ $DEBUG ]] && echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- nginx config changed, start reload nginx."
        systemctl reload nginx; fi

    # 9. 配置防火墙
    [[ $DEBUG ]] && echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- allow ip to any port: ${FIREWALL_WHITELIST[*]}"
    for ip in "${FIREWALL_WHITELIST[@]}"; do 
        ufw allow from ${ip} comment 'whitelist ip'> /dev/null
    done
    # ufw 开放 nginxListenPort 变量值的端口
    # ufw 防火墙配置的时候只支持 tcp/udp, 当代理 k8s service 时如果使用的是不是 UDP 协议(TCP|HTTP|HTTPS), 统统默认为 TCP
    if [[ ${serviceProtocol,,} != "udp"   ]]; then serviceProtocol="tcp"; fi
    [[ $DEBUG ]] && echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- allow access port: ${nginxListenPort}/${serviceProtocol}"
    ufw allow "${nginxListenPort}/${serviceProtocol}" comment "${serviceName}.${serviceNamespace}" > /dev/null
    local commandLine="ufw allow \"${nginxListenPort}/${serviceProtocol}\" comment \"${serviceName}.${serviceNamespace}\" > /dev/null"
    [[ $DEBUG ]] && echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- command line:      ${commandLine}"

    ufw default deny incoming > /dev/null
    ufw default allow outgoing > /dev/null
    ufw --force enable > /dev/null
}


function print_info() {
    [[ $INFO ]] && {
        echo "k8s service name:             ${serviceName}"
        echo "k8s service port name:        ${servicePort}"
        echo "k8s service namespace:        ${serviceNamespace}"
        echo "k8s service protocol:         ${serviceProtocol}"
        echo "k8s service port:             ${nginxListenPort}"
        echo "k8s service nodeport:         ${serviceNodeport}"
        echo "upstream available ip:        ${upstreamAvailIP[*]}"
    }
}


function clean_nginx() {
    [[ $DEBUG ]] && echo -e "\n========== start clean nginx. =========="

    # 1. 获取环境变量中所有需要代理的 k8s service 列表，并将需要代理的 k8s service 
    #    写入到 ${tmpFile} 临时文件中
    tmpFile='/tmp/.clean_nginx.tmp'
    : > ${tmpFile} # 清空文件
    # 设置为空值
    nginxListenPort=""
    serviceName=""
    serviceNamespace=""
    servicePort=""
    serviceProtocol=""
    serviceNodeport=""

    # 解析配置文件，获取 k8s service 信息
    while read -r serviceInfo; do
        #sleep 10
        nginxListenPort=` echo ${serviceInfo} | awk -F '[.|:|/]' '{print $1}'`
        serviceName=`     echo ${serviceInfo} | awk -F '[.|:|/]' '{print $2}'`
        serviceNamespace=`echo ${serviceInfo} | awk -F '[.|:|/]' '{print $3}'`
        servicePort=`     echo ${serviceInfo} | awk -F '[.|:|/]' '{print $4}'`
        serviceProtocol=` echo ${serviceInfo} | awk -F '[.|:|/]' '{print $5}'`

        # 1. 将变量的值都转换成小写 (k8s 中的名字都不允许大写)
        # 2. 设置变量的默认值，只有: serviceProtocol 如果为空，默认值为 tcp
        # 3. 如果变量值为空则不进行下一步处理，直接跳过
        serviceName=${serviceName,,}
        servicePort=${servicePort,,}
        serviceNamespace=${serviceNamespace,,}
        serviceProtocol=${serviceProtocol,,}
        if [[ -z ${serviceProtocol} ]]; then serviceProtocol="tcp"; fi

        if [[ -n ${serviceName} && -n ${serviceNamespace} &&
              -n ${servicePort} && -n ${serviceProtocol} ]]; then
            echo ${serviceName}.${serviceNamespace}:${servicePort}.${serviceProtocol} >> ${tmpFile}
        else
            [[ $DEBUG ]] && echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] can't get k8s service info: ${serviceInfo}."
            return ${RETURN_FAILURE}
        fi
    done < <(cat ${configPath} | grep -v "^#" | sed -e 's/"//g' -e s%\'%%g -e 's/[[:space:]]//g' -e '/^$/d' | sort -u)


    # 2. 关闭相关的 nginx 配置文件
    #    当环境变量文件中的相关 k8s service 不再需要被 nginx 代理时，此时就需要将相对应的
    #    nginx 配置文件关闭掉，同时将需要关闭的 nginx 配置文件写入到 deleteNginxConfigs 数组中
    # 环境变量
    #   oldNginxConfigs:          已经启用了的 nginx 配置文件的列表
    #   newNginxConfigs:          环境变量文件中指定需要代理的 k8s service 列表
    #   copiedOldNginxConfigs:    oldNginxConfigs 的拷贝
    #   deleteNginxConfigs:       需要关闭的 nginx 配置文件的列表
    # - oldNginxConfigs, newNginxConfigs 数组都是通过进程替换得来的，一般都会有空格和换行,
    #   此时需要通过重新初始化把数组中的空格和换行去掉，防止出现不可预知的错误
    unset oldNginxConfigs newNginxConfigs copiedOldNginxConfigs deleteNginxConfigs
    local oldNginxConfigs newNginxConfigs copiedOldNginxConfigs deleteNginxConfigs

    # 2.1 获取已启用的 nginx 配置文件列表, 写入 oldNginxConfigs 数组中
    mapfile oldNginxConfigs < <( \
        find ${NGINX_CONFIG_ENABLED_TCP_DIR} \
             ${NGINX_CONFIG_ENABLED_UDP_DIR} \
             ${NGINX_CONFIG_ENABLED_HTTP_DIR} \
             ${NGINX_CONFIG_ENABLED_HTTPS_DIR} \
             -type l ! -name '*.swp' -exec basename {} \; | sort -u )

    # 2.2 获取环境变量配置文件中指定需要代理的 k8s service 列表, 写入 newNginxConfigs 数组中
    mapfile newNginxConfigs < <( cat ${tmpFile} )
    oldNginxConfigs=( ${oldNginxConfigs[@]} )
    newNginxConfigs=( ${newNginxConfigs[@]} )
    copiedOldNginxConfigs=( ${oldNginxConfigs[@]} )

    # 2.3 获取需要关闭的 nginx 配置文件，并写入到 deleteNginxConfigs 数组中
    if [[ ${#newNginxConfigs[@]} -eq 0 ]]; then deleteNginxConfigs=( ${oldNginxConfigs[@]} ); fi
    for newConf in "${newNginxConfigs[@]}"; do
        copiedOldNginxConfigs=( $(echo ${copiedOldNginxConfigs[@]} | sed 's/\<'$newConf'\>//g') )
        deleteNginxConfigs=( ${copiedOldNginxConfigs[@]} )
        [[ $TRACE ]] && echo "[TRACE ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- deleteNginxConfigs: ${deleteNginxConfigs[@]}"
    done

    [[ $DEBUG ]] && {   #========== BEGIN DEBUG
        echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- all enabled nginx config:           ${oldNginxConfigs[*]}"
        echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- all k8s service that requires lb:   ${newNginxConfigs[*]}"
        echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- the nginx config will be disabled:  ${deleteNginxConfigs[*]}"
    }                   #========== END DEBUG

    # 3. 计算 clean_nginx 处理配置文件之前的 nginx 配置文件 hash 值
    local oldNginxConfigHash
    oldNginxConfigHash=$(find ${NGINX_CONFIG_ENABLED_TCP_DIR} \
                        ${NGINX_CONFIG_ENABLED_UDP_DIR} \
                        ${NGINX_CONFIG_ENABLED_HTTP_DIR} \
                        ${NGINX_CONFIG_ENABLED_HTTPS_DIR} \
                        -type l ! -name "*.swp" -exec md5sum {} \; | \
                        sort -u -k2 | md5sum | awk '{print $1}')

    # 4. 关闭相关 k8s service 的 nginx 配置文件
    for conf in "${deleteNginxConfigs[@]}"; do
        echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- disable nginx config: ${conf}."
        unlink ${NGINX_CONFIG_ENABLED_TCP_DIR}/${conf} &> /dev/null
        unlink ${NGINX_CONFIG_ENABLED_UDP_DIR}/${conf} &> /dev/null
        unlink ${NGINX_CONFIG_ENABLED_HTTP_DIR}/${conf} &> /dev/null
        unlink ${NGINX_CONFIG_ENABLED_HTTPS_DIR}/${conf} &> /dev/null
    done

    # 5. 计算 clean_nginx 处理配置文件之后的 nginx 配置文件哈希值
    local newNginxConfigHash
    newNginxConfigHash=$(find ${NGINX_CONFIG_ENABLED_TCP_DIR} \
                        ${NGINX_CONFIG_ENABLED_UDP_DIR} \
                        ${NGINX_CONFIG_ENABLED_HTTP_DIR} \
                        ${NGINX_CONFIG_ENABLED_HTTPS_DIR} \
                        -type l ! -name "*.swp" -exec md5sum {} \; | \
                        sort -u -k2 | md5sum | awk '{print $1}')
    [[ $DEBUG ]] && echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} $LINENO] -- nginx config hash (before run clean_nginx): ${oldNginxConfigHash}"
    [[ $DEBUG ]] && echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} $LINENO] -- nginx config hash (after run clean_nginx):  ${newNginxConfigHash}"


    # 6. 如果 clean_nginx 处理前后的 nginx 配置文件 hash 值发生变化，则重新加载 nginx
    #   如果 nginx reload 前后的 nginx worker 数量相同表明 nginx reload 完毕，则跳出循环
    local oldNumWorkerProcesses newNumWorkerProcesses nginxRelodRC
    nginx -t -q
    nginxRelodRC=$?
    if [[ ${nginxRelodRC} -eq 0  &&  "${oldNginxConfigHash}" != "${newNginxConfigHash}" ]]; then 
        oldNumWorkerProcesses=$(ps -ef | grep -c "[n]ginx: worker process")
        [[ $DEBUG ]] && echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- nginx config changed, systemctl reload nginx."
        [[ $DEBUG ]] && echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- number of nginx worker processes (before reload): ${oldNumWorkerProcesses}"
        systemctl reload nginx > /dev/null
        # 等待 nginx 重新加载完毕
        while true; do
            newNumWorkerProcesses=$(ps -ef | grep -c "[n]ginx: worker process")
            [[ $DEBUG ]] && echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- number of nginx worker processes (after reload): ${newNumWorkerProcesses}"
            # 通过检查在 nginx reload 前后的 nginx worker 数量来确认 nginx 是否重新加载完毕
            # 如果 nginx 重新加载完毕则退出循环, 否则 sleep
            if [[ $newNumWorkerProcesses -eq $oldNumWorkerProcesses ]]; then 
                [[ $DEBUG ]] && echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- nginx reload successfully."
                break; fi
            sleep 0.5
        done
    fi

    # 7. nginx 重新加载完毕发送信号给 clean_firewall, 让 clean_firewall 清理防火墙端口
    #    不管 nginx 是否需要 reload，都要发送一个信号给 clean_firewall, 否则 clean_firewall
    #    会一直阻塞
    echo "clean_nginx" > ${CHANNEL} &
    [[ $DEBUG ]] && echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- send a singal to clean_firewall."
}



function clean_firewall() {
    [[ $DEBUG ]] && echo -e "\n========== start clean firewall. =========="

    # 1. 等待 clean_nginx 重新加载 nginx 完毕 (systemctl relod nginx)
    local singal=""
    read singal < ${CHANNEL}
    [[ $DEBUG ]] && echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- got a singal from ${singal}."

    # 2. 删除 nginx 不再监听的端口
    #   svc_port_list:  当前 nginx 实际监听了的端口列表
    #   ufw_port_list:  ufw 防火墙实际开放了的端口
    unset svc_port_list ufw_port_list
    local svc_port_list ufw_port_list
    mapfile svc_port_list < <(ss -luntp | grep nginx | grep '0.0.0.0' | awk '{print $5}' | awk -F':' '{print $2}' | sort -un)
    ufw_port_list=( $(ufw status | sed -n '5,$p' | awk '{print $1}' | awk -F'/'  '{print $1}' | \
        sed s'/[Anywhere]//g' | sed /^$/d | sort -un) )
    svc_port_list=( ${svc_port_list[@]} )
    ufw_port_list=( ${ufw_port_list[@]} )
    [[ $DEBUG ]] && {   #========== BEGIN DEBUG
        echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- ufw allowed access ports: ${ufw_port_list[*]}"
        echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- k8s service open ports:   ${svc_port_list[*]}"
    }                   #========== END DEBUG

    # copiedArrayOld 为 ufw_port_list 副本
    # copiedArrayNew 为 svc_port_list 副本
    # arrayAdd 为 svc_port_list 比 ufw_port_list 多的端口
    # arrayDel 为 ufw_port_list 比 svc_port_list 多的端口
    # ufw_port_list 比 svc_port_list 多的端口需要通过 ufw 命令关闭掉

    # 获取 svc_port_list 比 ufw_port_list 多的端口，并放入 arrayAdd 数组中
    unset copiedArrayOld copiedArrayNew arrayAdd arrayDel
    local copiedArrayOld copiedArrayNew arrayAdd arrayDel
    copiedArrayOld=( ${ufw_port_list[@]} )
    copiedArrayNew=( ${svc_port_list[@]} )
    if [[ ${#copiedArrayOld[@]} -eq 0 ]]; then arrayAdd=( ${copiedArrayNew[@]} ); fi
    for key in ${copiedArrayOld[@]}; do
        copiedArrayNew=( $(echo ${copiedArrayNew[@]} | sed 's/\<'$key'\>//g') )
        arrayAdd=( ${copiedArrayNew[@]} )
    done

    # 获取 ufw_port_list 比 svc_port_list 多的端口，并放入 arrayDel 数组中
    unset copiedArrayOld copiedArrayNew arrayAdd arrayDel
    local copiedArrayOld copiedArrayNew arrayAdd arrayDel
    copiedArrayOld=( ${ufw_port_list[@]} )
    copiedArrayNew=( ${svc_port_list[@]} )
    if [[ ${#copiedArrayNew[@]} -eq 0 ]]; then arrayDel=( ${copiedArrayOld[@]} ); fi
    for key in ${copiedArrayNew[@]}; do
        copiedArrayOld=( $(echo ${copiedArrayOld[@]} | sed 's/\<'$key'\>//g') )
        arrayDel=( ${copiedArrayOld[@]} )
    done

    [[ $DEBUG ]] && {   #========== BEGIN DEBUG
        echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- ports need to be allow:  ${arrayAdd[*]}"
        echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- ports need to be remove: ${arrayDel[*]}"
    }                   #========== END DEBUG

    # 删除 ufw 多余开放的端口
    for port in "${arrayDel[@]}"; do
        echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- ufw delete allow port: ${port}"
        ufw delete allow ${port} > /dev/null
        ufw delete allow ${port}/tcp > /dev/null
        ufw delete allow ${port}/udp > /dev/null
    done
}


function main() {
    # singal_handler:   信号处理, 捕捉 INT, TERM, QUIT 信号
    # 0_prepare_xxx:    各种检查
    # service_handler           # 获取 k8s service 的 port 和 nodeport
    # nginx_handler             # 处理 nginx 模版文件并启用、开启对应端口的防火墙
    # clean_nginx               # 清理不再需要的为 k8s service 代理的 nginx 配置文件
    # clean_firewall            # 清理不再需要开放的 nginx 端口对应的防火墙
    singal_handler INT TERM QUIT
    0_prepare_k8s
    0_prepare_nginx
    clean_nginx

    # 设置为空值
    nginxListenPort=""
    serviceName=""
    serviceNamespace=""
    servicePort=""
    serviceProtocol=""
    serviceNodeport=""

    # 解析配置文件，获取 k8s service 信息
    while read -r serviceInfo; do
        nginxListenPort=` echo ${serviceInfo} | awk -F '[.|:|/]' '{print $1}'`
        serviceName=`     echo ${serviceInfo} | awk -F '[.|:|/]' '{print $2}'`
        serviceNamespace=`echo ${serviceInfo} | awk -F '[.|:|/]' '{print $3}'`
        servicePort=`     echo ${serviceInfo} | awk -F '[.|:|/]' '{print $4}'`
        serviceProtocol=` echo ${serviceInfo} | awk -F '[.|:|/]' '{print $5}'`

        # 1. 将变量的值都转换成小写 (k8s 中的名字都不允许大写)
        # 2. 设置变量的默认值，只有: serviceProtocol 如果为空，默认值为 tcp
        # 3. 如果变量值为空则不进行下一步处理，直接跳过
        serviceName=${serviceName,,}
        servicePort=${servicePort,,}
        serviceNamespace=${serviceNamespace,,}
        serviceProtocol=${serviceProtocol,,}
        if [[ -z ${serviceProtocol} ]]; then serviceProtocol="tcp"; fi

        [[ $DEBUG ]] && {   #========== BEGIN DEBUG
            echo -e "\n---------- ${serviceName}.${serviceNamespace}:${servicePort}.${serviceProtocol}"
            echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- nginxListenPort:  ${nginxListenPort}"
            echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- serviceName:      ${serviceName}"
            echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- serviceNamespace: ${serviceNamespace}"
            echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- servicePort:      ${servicePort}"
            echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- serviceProtocol:  ${serviceProtocol}"
            echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- serviceNodeport:  ${serviceNodeport}"
            echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- ${nginxListenPort}|${serviceProtocol}|${serviceName}.${serviceNamespace}:${servicePort}"
        }                   #========== END DEBUG

        if [[ -n ${serviceName} && -n ${serviceNamespace} &&
              -n ${servicePort} && -n ${serviceProtocol} ]]; then
            service_handler
            nginx_handler
            # print_info
            :
       fi
    done < <(cat ${configPath} | grep -v "^#" | sed -e 's/"//g' -e s%\'%%g -e 's/[[:space:]]//g' -e '/^$/d' | sort -u)
    clean_firewall
}

while true; do
    main
    sleep ${LOOP_TIME}
done
