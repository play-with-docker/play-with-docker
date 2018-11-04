#!/bin/bash

set -e

function wait_for_url {
    # Wait for docker daemon to be ready
    while ! curl -k -sS $1 > /dev/null; do
        sleep 1;
    done
}

function deploy_ucp {
    wait_for_url "https://localhost:2376"
    docker run --rm -i  --name ucp \
        -v /var/run/docker.sock:/var/run/docker.sock \
        docker/ucp:3.0.5 install --debug --force-insecure-tcp \
        --san *.direct.${PWD_HOST_FQDN} \
        --license $(cat $HOME/workshop.lic) \
        --swarm-port 2375 \
        --admin-username admin \
        --admin-password admin1234

    rm $HOME/workshop.lic
    echo "Finished deploying UCP"
}

function get_instance_ip {
    ip -o -4 a s eth1 | awk '{print $4}' | cut -d '/' -f1
}

function get_node_routable_ip {
    curl -sS https://${PWD_HOST_FQDN}/sessions/${SESSION_ID} | jq -r '.instances[] | select(.hostname == "'$1'") | .routable_ip'
}

function get_direct_url_from_ip {
    local ip_dash="${1//./-}"
    local url="https://ip${ip_dash}-${SESSION_ID}.direct.${PWD_HOST_FQDN}"
    echo $url
}

function deploy_dtr {
    if [ $# -lt 1 ]; then
        echo "DTR node hostname"
        return
    fi


    local dtr_ip=$(get_node_routable_ip $1)
    local ucp_ip=$(get_instance_ip)

    local dtr_url=$(get_direct_url_from_ip $dtr_ip)
    local ucp_url=$(get_direct_url_from_ip $ucp_ip)

    docker run -i --rm docker/dtr:2.5.5 install \
      --dtr-external-url $dtr_url \
      --ucp-node $1 \
      --ucp-username admin \
      --ucp-password admin1234 \
      --ucp-insecure-tls \
      --ucp-url $ucp_url
}

function setup_dtr_certs {
    if [ $# -lt 1 ]; then
        echo "DTR node hostname is missing"
        return
    fi


    local dtr_ip=$(get_node_routable_ip $1)
    local dtr_url=$(get_direct_url_from_ip $dtr_ip)
    local dtr_hostname="${dtr_url/https:\/\/}"

    wait_for_url "$dtr_url/ca"

    curl -kfsSL $dtr_url/ca -o /usr/local/share/ca-certificates/$dtr_hostname.crt
    update-ca-certificates
}


case "$1" in
    deploy)
            deploy_ucp
            deploy_dtr $2
            setup_dtr_certs $2
            ;;
    setup-certs)
            setup_dtr_certs $2
            ;;
    *)
            echo "Illegal option $1"
            ;;
esac

