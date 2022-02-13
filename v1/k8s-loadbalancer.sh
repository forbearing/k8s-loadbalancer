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

LOOP_TIME="30"
CONFIG_PATH="/etc/k8s-loadbalancer/conf.d"
ENV_FILE="/etc/k8s-loadbalancer/k8s-loadbalancer.env"
NGINX_CONFIG_AVAILABLE_DIR="/etc/nginx/sites-available"
NGINX_CONFIG_ENABLED_HTTP_DIR="/etc/nginx/sites-enabled"
NGINX_CONFIG_ENABLED_HTTPS_DIR="/etc/nginx/sites-enabled"
NGINX_CONFIG_ENABLED_TCP_DIR="/etc/nginx/sites-stream"
NGINX_CONFIG_ENABLED_UDP_DIR="/etc/nginx/sites-stream"
CHANNEL="/tmp/k8s-loadbalancer"

# declare -ax UPSTREAM_IP UPSTREAM_AVAIL_IP FIREWAL_WHITELIST
UPSTREAM_IP=(
    10.240.3.21
    10.240.3.22
    10.240.3.23)
UPSTREAM_AVAIL_IP=()
FIREWALL_WHITELIST=(10.240.0.100)

# ip_hash           哈希算法
# fair              按后端服务器响应时间来分配请求，响应时间短的优先分配
# url_hash          按访问url的hash结果来分配请求。使每个url定向到同一个（对应的）后端服务器，后端服务器为缓存时比较有效。
# least_conn        最小连接数
UPSTREAM_LOADBALANCER="least_conn"

# declare -p SVC_NAME SVC_PORT_NAME SVC_NAMESPACE SVC_PROTOCOL
# declare -p SVC_PORT SVC_NODEPORT
SVC_NAME=""
SVC_PORT_NAME=""
SVC_NAMESPACE=""
SVC_PROTOCOL=""
SVC_PORT=""
SVC_NODEPORT=""
LISTEN_PORT=""



function _test_var(){
    [[ $DEBUG ]] && {   # ========== BEGIN DEBUG
        echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- SVC_NAME:       ${SVC_NAME}"
        echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- SVC_PORT_NAME:  ${SVC_PORT_NAME}"
        echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- SVC_NAMESPACE:  ${SVC_NAMESPACE}"
        echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- SVC_PROTOCOL:   ${SVC_PROTOCOL}"
        echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- SVC_PORT:       ${SVC_PORT}"
        echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- SVC_NODEPORT:   ${SVC_NODEPORT}"
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
    if [[ ! -f ${ENV_FILE} ]]; then
        echo "not found environment file: ${ENV_FILE}, exit failed."
        exit $EXIT_FAILURE; fi
    if [[ ! -d ${CONFIG_PATH} ]]; then
        echo "not found config directory: ${CONFIG_PATH}, exit failed."
        exit $EXIT_FAILURE; fi

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
    # 加载环境变量文件
    source ${ENV_FILE}

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
            cp -f ${CONFIG_PATH}/nginx.conf /etc/nginx/nginx.conf
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
            cp -f ${CONFIG_PATH}/nginx.conf /etc/nginx/nginx.conf
            mv /etc/nginx/conf.d/default.conf /etc/nginx/conf.d/default.conf.bak &> /dev/null
            systemctl enable --now nginx
        fi ;;
    *)
        echo "Not Support Linux: ${linux_id}"
        exit $EXIT_FAILURE
        ;;
    esac
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
    #   SVC_NAME, SVC_PORT_NAME, SVC_NAMESPACe, SVC_PROTOCOL
    if [[ -z $SVC_NAME ]]; then
        echo "[ERROR ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}]: variable SVC_NAME is empty, exit failed."
        exit $EXIT_FAILURE; fi
    if [[ -z $SVC_PORT_NAME ]]; then
        echo "[ERROR ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}]: variable SVC_PORT_NAME is empty, exit failed."
        exit $EXIT_FAILURE; fi
    if [[ -z $SVC_NAMESPACE ]]; then
        echo "[ERROR ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}]: variable SVC_NAMESPACE is empty, exit failed."
        exit $EXIT_FAILURE; fi
    if [[ -z $SVC_PROTOCOL} ]]; then
        echo "[ERROR ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}]: variable SVC_PROTOCOL is empty, exit failed."
        exit $EXIT_FAILURE; fi
    local svc_name=${SVC_NAME}
    local svc_port_name=${SVC_PORT_NAME}
    local svc_namespace=${SVC_NAMESPACE}
    local svc_protocol=${SVC_PROTOCOL}
    local svc_port=""
    local svc_nodeport=""

    # 循环3次，获取 k8s service port
    local count=1
    while true; do
        local jsonpath_string=$(echo jsonpath="{.spec.ports[?(@.name==\""${svc_port_name}\"")].port}")
        svc_port=$(kubectl -n ${svc_namespace} get svc ${svc_name} -o ${jsonpath_string})
        if [[ -n ${svc_port} ]]; then break; fi
        if [[ ${count} -ge 3 ]]; then break; fi
        sleep 3; (( count++ ))
    done

    # 循环3次，获取 k8s service nodeport
    local count=1
    while true; do
        local jsonpath_string=$(echo jsonpath="{.spec.ports[?(@.name==\""${svc_port_name}\"")].nodePort}")
        svc_nodeport=$(kubectl -n ${svc_namespace} get svc ${svc_name} -o ${jsonpath_string})
        if [[ -n ${svc_nodeport} ]]; then break; fi
        if [[ ${count} -ge 3 ]]; then break; fi
        sleep 3; (( count++ ))
    done

    SVC_PORT=${svc_port}
    SVC_NODEPORT=${svc_nodeport}

    [[ $DEBUG ]] && {   #========== BEGIN DEBUG
        echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- SVC_NAME:       ${SVC_NAME}"
        echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- SVC_PORT_NAME:  ${SVC_PORT_NAME}"
        echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- SVC_NAMESPACE:  ${SVC_NAMESPACE}"
        echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- SVC_PROTOCOL:   ${SVC_PROTOCOL}"
        echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- SVC_PORT:       ${SVC_PORT}"
        echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- SVC_NODEPORT:   ${SVC_NODEPORT}"
    }                   #========== END DEBUG
}

function nginx_handler() {
    # 检查变量, 如果变量为空, 则跳过处理
    #   SVC_NAME, SVC_PORT_NAME, SVC_NAMESPACE, SVC_PROTOCOL, SVC_PORT, SVC_NODEPORT
    [[ $WARN ]] && {
        if [[ -z $SVC_NAME ]]; then
            echo "[WARNNING ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}]: variable SVC_NAME is empty, skip."
            return $EXIT_FAILURE; fi
        if [[ -z $SVC_PORT_NAME ]]; then
            echo "[WARNNING ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}]: variable SVC_PORT_NAME is empty, skip."
            return $EXIT_FAILURE; fi
        if [[ -z $SVC_NAMESPACE ]]; then
            echo "[WARNNING ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}]: variable SVC_NAMESPACE is empty, skip."
            return $EXIT_FAILURE; fi
        if [[ -z $SVC_PROTOCOL} ]]; then
            echo "[WARNNING ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}]: variable SVC_PROTOCOL is empty, skip."
            return $EXIT_FAILURE; fi
        if [[ -z ${SVC_PORT} ]]; then
            echo "[WARNNING ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}]: variable SVC_PORT is empty, skip."
            return $EXIT_FAILURE; fi
        if [[ -z ${SVC_NODEPORT} ]]; then
            echo "[WARNNING ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}]: variable SVC_NODEPORT is empty, skip."
            return $EXIT_FAILURE; fi
        # 如果配置文件目录 CONFIG_PATH 不存在，直接退出
        if [[ ! -d ${CONFIG_PATH} ]]; then
            echo "[WARNNING ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}]: not found directory ${CONFIG_PATH}, skip."
            return ${EXIT_FAILURE}; fi
    }


    # 1. 简单检查哪一些上游的 k8s 节点是可用的，如果是可用的就加入到 upstreamAvailIP 数组中
    local upstreamAvailIP=()
    for ip in "${UPSTREAM_IP[@]}"; do
        case ${SVC_PROTOCOL} in
        tcp|http|https)
            [[ $DEBUG ]] && echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- nc -vz ${ip} ${SVC_NODEPORT}"
            nc -vz ${ip} ${SVC_NODEPORT} &> /dev/null 
            if [[ $? -eq 0 ]]; then upstreamAvailIP+=($ip); fi ;;
        udp)
            [[ $DEBUG ]] && echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- nc -u -vz ${ip} ${SVC_NODEPORT}"
            nc -u -vz ${ip} ${SVC_NODEPORT} &> /dev/null
            if [[ $? -eq 0 ]]; then upstreamAvailIP+=($ip); fi ;;
        *)  echo "[ERROR ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] not support service protocol: ${SVC_PROTOCOL}, exit failed." && exit $EXIT_FAILURE
        esac
    done

    # 2. 如果没有任何可用的上游 k8s 节点，则跳过处理
    if [[ ${#upstreamAvailIP[@]} -eq 0 ]]; then
        [[ $WARN ]] && echo "[WARNNING ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}]: upstream no available ip, nginx will not add config for ${SVC_NAME}."
        return 1; fi
    UPSTREAM_AVAIL_IP=( "${upstreamAvailIP[@]}" )

    # 3. 计算 nginx 配置文件 hash 值
    local oldNginxConfigHash
    oldNginxConfigHash=$(find ${NGINX_CONFIG_ENABLED_TCP_DIR} \
                        ${NGINX_CONFIG_ENABLED_UDP_DIR} \
                        ${NGINX_CONFIG_ENABLED_HTTP_DIR} \
                        ${NGINX_CONFIG_ENABLED_HTTPS_DIR} \
                        -type l ! -name "*.swp" -exec md5sum {} \; | \
                        sort -u -k2 | md5sum | awk '{print $1}')

    # 4. 处理 nginx 配置文件模版
    cp -f ${CONFIG_PATH}/template-${SVC_PROTOCOL} ${NGINX_CONFIG_AVAILABLE_DIR}/${SVC_PROTOCOL}-${SVC_NAME}
    local serverString=
    for (( i=0; i<${#upstreamAvailIP[@]}; i++ )); do
        serverString="    server ${upstreamAvailIP[i]}:${SVC_NODEPORT};"
        sed -i "/^upstream/a\\${serverString}"  \
            ${NGINX_CONFIG_AVAILABLE_DIR}/${SVC_PROTOCOL}-${SVC_NAME}
        [[ $DEBUG ]] && echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- nginx upstream server:  ${serverString}"
    done
    # 如果设置了 LISTEN_PORT 变量，则使用 LISTEN_PORT 变量的值作为 nginx 监听端口，
    # 否则使用 SVC_PORT 的值作为 nginx 监听端口
    if [[ -n ${LISTEN_PORT} ]]; then
        [[ $DEBUG ]] && echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- nginx listen port: ${LISTEN_PORT}"
        sed -i "s%#LISTEN_PORT#%${LISTEN_PORT}%g"   "${NGINX_CONFIG_AVAILABLE_DIR}/${SVC_PROTOCOL}-${SVC_NAME}"
    else
        [[ $DEBUG ]] && echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- nginx listen port: ${SVC_PORT}"
        sed -i "s%#LISTEN_PORT#%${SVC_PORT}%g"      "${NGINX_CONFIG_AVAILABLE_DIR}/${SVC_PROTOCOL}-${SVC_NAME}"
    fi
    sed -i "s%#UPSTREAM_NAME#%${SVC_NAME//-/_}%g"   "${NGINX_CONFIG_AVAILABLE_DIR}/${SVC_PROTOCOL}-${SVC_NAME}"
    sed -i "s%#ACCESS_LOG#%${SVC_NAME//_/-}%g"      "${NGINX_CONFIG_AVAILABLE_DIR}/${SVC_PROTOCOL}-${SVC_NAME}"

    # 5. 启用 nginx 配置文件
    case ${SVC_PROTOCOL} in
    tcp)   ln -sf "${NGINX_CONFIG_AVAILABLE_DIR}/${SVC_PROTOCOL}-${SVC_NAME}" ${NGINX_CONFIG_ENABLED_TCP_DIR} ;;
    udp)   ln -sf "${NGINX_CONFIG_AVAILABLE_DIR}/${SVC_PROTOCOL}-${SVC_NAME}" ${NGINX_CONFIG_ENABLED_UDP_DIR} ;;
    http)  ln -sf "${NGINX_CONFIG_AVAILABLE_DIR}/${SVC_PROTOCOL}-${SVC_NAME}" ${NGINX_CONFIG_ENABLED_HTTP_DIR} ;;
    https) ln -sf "${NGINX_CONFIG_AVAILABLE_DIR}/${SVC_PROTOCOL}-${SVC_NAME}" ${NGINX_CONFIG_ENABLED_HTTPS_DIR} ;;
    *)     echo "[ERROR ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] not support service protocol: ${SVC_PROTOCOL}, exit failed." && exit $EXIT_FAILURE
    esac

    # 6. 再次计算 nginx 配置文件哈希值
    local newNginxConfigHash
    newNginxConfigHash=$(find ${NGINX_CONFIG_ENABLED_TCP_DIR} \
                        ${NGINX_CONFIG_ENABLED_UDP_DIR} \
                        ${NGINX_CONFIG_ENABLED_HTTP_DIR} \
                        ${NGINX_CONFIG_ENABLED_HTTPS_DIR} \
                        -type l ! -name "*.swp" -exec md5sum {} \; | \
                        sort -u -k2 | md5sum | awk '{print $1}')
    [[ $DEBUG ]] && echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} $LINENO] -- nginx config hash (before enabled ${SVC_PROTOCOL}-${SVC_NAME}): ${oldNginxConfigHash}"
    [[ $DEBUG ]] && echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} $LINENO] -- nginx config hash (after enabled ${SVC_PROTOCOL}-${SVC_NAME}):  ${newNginxConfigHash}"

    # 7. 如果 nginx -t 失败，则把刚才启用了的 nginx 配置文件关闭
    nginx -t -q
    local nginxRelodRC=$?
    if [[ ${nginxRelodRC} -ne 0 ]]; then
        [[ $WARN ]] && echo "[WARNNING ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- nginx config test failed. skip."
        case ${SVC_PROTOCOL} in
        tcp)   unlink "${NGINX_CONFIG_ENABLED_TCP_DIR}"/"${SVC_PROTOCOL}-${SVC_NAME}" ;;
        udp)   unlink "${NGINX_CONFIG_ENABLED_UDP_DIR}"/"${SVC_PROTOCOL-${SVC_NAME}}" ;;
        http)  unlink "${NGINX_CONFIG_ENABLED_HTTP_DIR}"/"${SVC_PROTOCOL}-${SVC_NAME}" ;;
        https) unlink "${NGINX_CONFIG_ENABLED_HTTPS_DIR}"/"${SVC_PROTOCOL}-${SVC_NAME}" ;;
        *)     echo "[ERROR ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] not support service protocol: ${SVC_PROTOCOL}, exit failed." && exit $EXIT_FAILURE
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
    # 如果设置了 LISTEN_PORT 变量，ufw 开放 LISTEN_PORT 值的端口
    # 否则设置 SVC_PORT 变量值的端口
    if [[ -n ${LISTEN_PORT} ]]; then
        [[ $DEBUG ]] && echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- allow access port: ${LISTEN_PORT}/${SVC_PROTOCOL}"
        ufw allow "${LISTEN_PORT}/${SVC_PROTOCOL}" comment "${SVC_NAMESPACE}/${SVC_NAME}:${SVC_PORT_NAME}" > /dev/null
    else
        [[ $DEBUG ]] && echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- allow access port: ${SVC_PORT}/${SVC_PROTOCOL}"
        ufw allow "${SVC_PORT}/${SVC_PROTOCOL}" comment "${SVC_NAMESPACE}/${SVC_NAME}:${SVC_PORT_NAME}" > /dev/null
    fi
    ufw default deny incoming > /dev/null
    ufw default allow outgoing > /dev/null
    ufw --force enable > /dev/null
}


function print_info() {
    [[ $INFO ]] && {
        echo "k8s service name:             ${SVC_NAME}"
        echo "k8s service port name:        ${SVC_PORT_NAME}"
        echo "k8s service namespace:        ${SVC_NAMESPACE}"
        echo "k8s service protocol:         ${SVC_PROTOCOL}"
        echo "k8s service port:             ${SVC_PORT}"
        echo "k8s service nodeport:         ${SVC_NODEPORT}"
        echo "upstream available ip:        ${UPSTREAM_AVAIL_IP[*]}"
    }
}


function clean_nginx() {
    [[ $DEBUG ]] && echo -e "\n========== start clean nginx. =========="

    # 1. 获取环境变量中所有需要代理的 k8s service 列表，并将需要代理的 k8s service 
    #    写入到 ${tmpFile} 临时文件中
    tmpFile='/tmp/.clean_nginx.tmp'
    : > ${tmpFile} # 清空文件
    unset dict_list
    source ${ENV_FILE}
    mapfile dict_list < <(declare -Ap | grep -i LB_ | awk '{print $3}' | awk -F'=' '{print $1}')
    dict_list=( ${dict_list[@]} )
    for dict in "${dict_list[@]}"; do
        while read -r file; do
            echo "${file}" >> ${tmpFile}
        done < <( \
            declare -p "${dict}" |  \
            awk -F '[()]' '{print $2}' | \
            sed -e s'/\[//g' -e s'/\]//g' -e 's/ /\n/g' | \
            grep 'SVC_NAME=' | \
            awk -F'=' '{print $2}' | \
            sed -e 's/"//g' | sort )
    done


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
    for oldConf in "${oldNginxConfigs[@]}"; do
        for newConf in "${newNginxConfigs[@]}"; do
            if [[ "$oldConf" == "$newConf" ||
                  "$oldConf" == "tcp-$newConf" ||
                  "$oldConf" == "udp-$newConf" ||
                  "$oldConf" == "http-$newConf" ||
                  "$oldConf" == "https-$newConf" ]]; then
                copiedOldNginxConfigs=( $(echo ${copiedOldNginxConfigs[@]} | sed 's/\<'$oldConf'\>//g') )
                deleteNginxConfigs=( ${copiedOldNginxConfigs[@]} )
                [[ $TRACE ]] && echo "[TRACE ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- deleteNginxConfigs: ${deleteNginxConfigs[@]}"
            fi
        done
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


    # # 如果上游服务器不可用，关闭对应的 nginx 配置文件
    # while read -r ip port; do
    #     if ! nc -vz ${ip} ${port} &> /dev/null; then
    #         while read -r file; do
    #             #===== BEGIN DEBUG
    #             if [[ ${DEBUG,,} == "true" ]]; then
    #                 printf "[DEBUG %s %s] unlink %s\n" ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO} ${file}
    #             fi
    #             #===== END DEBUG
    #             unlink ${file}
    #         done < <( grep -Rsi "${port}" /etc/nginx/sites-available/ | sed -e s'/;//g' -e s'/:/ /g' | \
    #             awk '{print $1}' | sort | uniq )
    #     fi
    # done < <( grep -Rsi 'server.*:.*;' /etc/nginx/sites-available/ | \
    #     sed -e s'/;//g' -e s'/:/ /g' | \
    #     awk '{printf "%s %s\n", $3,$4}' )
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

    unset dict_list
    source ${ENV_FILE}
    mapfile dict_list < <(declare -Ap | grep -i LB_ | awk '{print $3}' | awk -F'=' '{print $1}')
    dict_list=( ${dict_list[@]} )       # 进程替换出来的数组，需要去除空格和换行符
    svc_list=( $( echo ${dict_list[@]} | sed -e 's/LB_//g' -e 's/_/-/g') )
    [[ $DEBUG ]] && echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- k8s service list: ${svc_list[*]}"

    # 循环获取每个 k8s service 相对应的变量文件
    # dict_list 中
    for dict in "${dict_list[@]}"; do
        declare -p ${dict} |  \
            awk -F '[()]' '{print $2}' | \
            sed -e s'/\[//g' -e s'/\]//g' > /tmp/.k8s-loadbalancer.env
        SVC_NAME=""
        SVC_PORT_NAME=""
        SVC_NAMESPACE=""
        SVC_PROTOCOL=""
        SVC_PORT=""
        SVC_NODEPORT=""
        LISTEN_PORT=""
        source /tmp/.k8s-loadbalancer.env

        [[ $DEBUG ]] && echo -e "\n========== handle ${SVC_NAME,,} =========="
        [[ $DEBUG ]] && {   #========== BEGIN DEBUG
            echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- SVC_NAME:      ${SVC_NAME}"
            echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- SVC_PORT_NAME: ${SVC_PORT_NAME}"
            echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- SVC_NAMESPACE: ${SVC_NAMESPACE}"
            echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- SVC_PROTOCOL:  ${SVC_PROTOCOL}"
            echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- SVC_PORT:      ${SVC_PORT}"
            echo "[DEBUG ${FUNCNAME[0]:+${FUNCNAME[0]}()} ${LINENO}] -- SVC_NODEPORT:  ${SVC_NODEPORT}"
        }                   #========== END DEBUG

        # 1. 将变量的值都转换成小写 (k8s 中的名字都不允许大写)
        # 2. 设置变量的默认值，只有: SVC_PROTOCOL 如果为空，默认值为 tcp
        # 3. 如果变量值为空则不进行下一步处理，直接跳过
        SVC_NAME=${SVC_NAME,,}
        SVC_PORT_NAME=${SVC_PORT_NAME,,}
        SVC_NAMESPACE=${SVC_NAMESPACE,,}
        SVC_PROTOCOL=${SVC_PROTOCOL,,}
        if [[ -z ${SVC_PROTOCOL} ]]; then SVC_PROTOCOL="tcp"; fi
        if [[ -n ${SVC_NAME} && -n ${SVC_PORT_NAME} &&
              -n ${SVC_NAMESPACE} && -n ${SVC_PROTOCOL} ]]; then
            service_handler
            nginx_handler
            print_info
       fi
    done
    clean_firewall
}

while true; do
    main
    sleep ${LOOP_TIME}
done
