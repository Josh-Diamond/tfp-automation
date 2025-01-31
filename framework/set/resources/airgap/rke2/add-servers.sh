#!/bin/bash

USER=$1
GROUP=$2
RKE2_SERVER_ONE_IP=$3
RKE2_NEW_SERVER_IP=$4
RKE2_TOKEN=$5
REGISTRY=$6
RANCHER_IMAGE=$7
RANCHER_TAG_VERSION=$8
STAGING_RANCHER_AGENT_IMAGE=${9:-""}
PRIME_RANCHER_AGENT_IMAGE=${10}
PEM_FILE=/home/$USER/airgap.pem

set -e

runSSH() {
  local server="$1"
  local cmd="$2"
  
  ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i "$PEM_FILE" "$USER@$server" \
  "export USER=${USER}; \
   export GROUP=${GROUP}; \
   export RKE2_SERVER_ONE_IP=${RKE2_SERVER_ONE_IP}; \
   export RKE2_TOKEN=${RKE2_TOKEN}; \
   export REGISTRY=${REGISTRY}; \
   export REGISTRY_USERNAME=${REGISTRY_USERNAME}; \
   export REGISTRY_PASSWORD=${REGISTRY_PASSWORD}; $cmd"
}

setupConfig() {
    sudo mkdir -p /etc/rancher/rke2
    sudo tee /etc/rancher/rke2/config.yaml > /dev/null << EOF
server: https://${RKE2_SERVER_ONE_IP}:9345
token: ${RKE2_TOKEN}
tls-san:
  - ${RKE2_SERVER_ONE_IP}
EOF
}

setupRegistry() {
  sudo mkdir -p /etc/rancher/rke2
  sudo tee -a /etc/rancher/rke2/registries.yaml > /dev/null << EOF
mirrors:
  docker.io:
    endpoint:
      - "https://${REGISTRY}"
configs:
  "${REGISTRY}":
    tls:
      insecure_skip_verify: true
EOF
}

setupDockerDaemon() {
  sudo tee -a /etc/docker/daemon.json > /dev/null << EOF
{
  "insecure-registries" : [ "${REGISTRY}" ]
}
EOF
}

configFunction=$(declare -f setupConfig)
runSSH "${RKE2_NEW_SERVER_IP}" "${configFunction}; setupConfig"

setupRegistryFunction=$(declare -f setupRegistry)
runSSH "${RKE2_NEW_SERVER_IP}" "${setupRegistryFunction}; setupRegistry"

runSSH "${RKE2_NEW_SERVER_IP}" "sudo INSTALL_RKE2_ARTIFACT_PATH=/home/${USER} sh install.sh"
runSSH "${RKE2_NEW_SERVER_IP}" "sudo systemctl enable rke2-server"
runSSH "${RKE2_NEW_SERVER_IP}" "sudo systemctl start rke2-server"

if [ -n "$STAGING_RANCHER_AGENT_IMAGE" ] || [ -n "$PRIME_RANCHER_AGENT_IMAGE" ]; then
  setupDaemonFunction=$(declare -f setupDockerDaemon)
  runSSH "${RKE2_NEW_SERVER_IP}" "${setupDaemonFunction}; setupDockerDaemon"
  runSSH "${RKE2_NEW_SERVER_IP}" "sudo systemctl restart docker && sudo systemctl daemon-reload"

  runSSH "${RKE2_NEW_SERVER_IP}" "sudo docker pull ${REGISTRY}/${RANCHER_IMAGE}:${RANCHER_TAG_VERSION}"

  if [ -n "$STAGING_RANCHER_AGENT_IMAGE" ]; then
    runSSH "${RKE2_NEW_SERVER_IP}" "sudo docker pull ${REGISTRY}/${STAGING_RANCHER_AGENT_IMAGE}:${RANCHER_TAG_VERSION}"
  fi

  if [ -n "$PRIME_RANCHER_AGENT_IMAGE" ]; then
    runSSH "${RKE2_NEW_SERVER_IP}" "sudo docker pull ${REGISTRY}/${PRIME_RANCHER_AGENT_IMAGE}:${RANCHER_TAG_VERSION}"
  fi
  
  runSSH "${RKE2_NEW_SERVER_IP}" "sudo systemctl restart rke2-server"
fi

kubectl get nodes