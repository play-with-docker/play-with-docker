#!/bin/bash 
# Copyright 2017 Mirantis
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

set -o pipefail
set -o errtrace

apiserver_static_pod="/etc/kubernetes/manifests/kube-apiserver"

# jq filters follow.

# TODO: think about more secure possibilities
apiserver_anonymous_auth='.spec.containers[0].command|=map(select(startswith("--token-auth-file")|not))+["--token-auth-file=/etc/pki/tokens.csv"]'

# Sets etcd2 as backend
apiserver_etcd2_backend='.spec.containers[0].command|=map(select(startswith("--storage-backend")|not))+["--storage-backend=etcd2"]'

# Make apiserver accept insecure connections on port 8080
# TODO: don't use insecure port
#apiserver_insecure_bind_port='.spec.containers[0].command|=map(select(startswith("--insecure-port=")|not))+["--insecure-port=2375"]'


# Update kube-proxy CIDR, enable --masquerade-all and disable conntrack (see dind::frob-proxy below)
function dind::proxy-cidr-and-no-conntrack {
  cluster_cidr="$(ip addr show docker0 | grep -w inet | awk '{ print $2; }')"
  echo ".items[0].spec.template.spec.containers[0].command |= .+ [\"--cluster-cidr=${cluster_cidr}\", \"--masquerade-all\", \"--conntrack-max=0\", \"--conntrack-max-per-core=0\"]"
}


# Adds route to defualt eth0 interface so 10.96.x.x can go through
function dind::add-route {
   ip route add 10.96.0.0/16 dev eth0
}



function dind::join-filters {
  local IFS="|"
  echo "$*"
}

function dind::frob-apiserver {
  local -a filters=("${apiserver_anonymous_auth}")

  dind::frob-file "${apiserver_static_pod}" "${filters[@]}"
}

function dind::frob-file {
  local path_base="$1"
  shift
  local filter="$(dind::join-filters "$@")"
  local status=0
  if [[ -f ${path_base}.yaml ]]; then
    dind::yq "${filter}" "${path_base}.yaml" || status=$?
  else
    echo "${path_base}.json or ${path_base}.yaml not found" >&2
    return 1
  fi
  if [[ ${status} -ne 0 ]]; then
    echo "Failed to frob ${path_base}.yaml" >&2
    return 1
  fi
}

function dind::yq {
  local filter="$1"
  local path="$2"
  # We need to use a temp file here because if you feed an object to
  # 'kubectl convert' via stdin, you'll get a List object because
  # multiple input objects are implied
  tmp="$(mktemp tmp-XXXXXXXXXX.json)"
  kubectl convert -f "${path}" --local -o json 2>/dev/null |
    jq "${filter}" > "${tmp}"
  kubectl convert -f "${tmp}" --local -o yaml 2>/dev/null >"${path}"
  rm -f "${tmp}"
}

function dind::frob-proxy {
  # Trying to change conntrack settings fails even in priveleged containers,
  # so we need to avoid it. Here's sample error message from kube-proxy:
  # I1010 21:53:00.525940       1 conntrack.go:57] Setting conntrack hashsize to 49152
  # Error: write /sys/module/nf_conntrack/parameters/hashsize: operation not supported
  # write /sys/module/nf_conntrack/parameters/hashsize: operation not supported
  #
  # Recipe by @errordeveloper:
  # https://github.com/kubernetes/kubernetes/pull/34522#issuecomment-253248985
  local force_apply=--force
  if ! kubectl version --short >&/dev/null; then
    # kubectl 1.4 doesn't have version --short and also it doesn't support apply --force
    force_apply=
  fi
  KUBECONFIG=/etc/kubernetes/admin.conf kubectl -n kube-system get ds -l k8s-app=kube-proxy -o json |
    jq "$(dind::join-filters "$(dind::proxy-cidr-and-no-conntrack)")" | KUBECONFIG=/etc/kubernetes/admin.conf kubectl apply ${force_apply} -f -

  KUBECONFIG=/etc/kubernetes/admin.conf kubectl -n kube-system delete pods --now -l "k8s-app=kube-proxy"
}


function dind::wait-for-apiserver {
    echo -n "Waiting for api server to startup"
    local url="https://localhost:6443/api"
    local n=60
    while true; do
      if curl -k -s "${url}" >&/dev/null; then
        break
      fi
      if ((--n == 0)); then
        echo "Error: timed out waiting for apiserver to become available" >&2
      fi
      echo -n "."
      sleep 0.5
    done
    echo ""
}

function dind::frob-cluster {
  dind::frob-apiserver
  dind::wait-for-apiserver
  dind::frob-proxy
}

# Weave depends on /etc/machine-id being unique
if [[ ! -f /etc/machine-id ]]; then
  rm -f /etc/machine-id
  systemd-machine-id-setup
fi

if [[ "$@" == "init"* || "$@" == "join"* ]]; then
# Call kubeadm with params and skip flag
	/usr/bin/kubeadm "$@" --ignore-preflight-errors all
else
# Call kubeadm with params
	/usr/bin/kubeadm "$@" 
fi

# Frob cluster
if [[ "$@" == "init"* && $? -eq 0 && ! "$@" == *"--help"* ]]; then
   dind::frob-cluster
else
   dind::add-route
fi

