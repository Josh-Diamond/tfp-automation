#!/bin/bash

USER=$1
GROUP=$2
RKE2_SERVER_ONE_IP=$3
RKE2_TOKEN=$4
REGISTRY=$5
RANCHER_IMAGE=$6
RANCHER_TAG_VERSION=$7
RANCHER_AGENT_IMAGE=${8}
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
  sudo tee /etc/docker/daemon.json > /dev/null << EOF
{
  "insecure-registries" : [ "${REGISTRY}" ]
}
EOF
}

runSSH "${RKE2_SERVER_ONE_IP}" "sudo mv /home/${USER}/kubectl /usr/local/bin/"

configFunction=$(declare -f setupConfig)
runSSH "${RKE2_SERVER_ONE_IP}" "${configFunction}; setupConfig"

setupRegistryFunction=$(declare -f setupRegistry)
runSSH "${RKE2_SERVER_ONE_IP}" "${setupRegistryFunction}; setupRegistry"

runSSH "${RKE2_SERVER_ONE_IP}" "sudo INSTALL_RKE2_ARTIFACT_PATH=/home/${USER} sh install.sh"
runSSH "${RKE2_SERVER_ONE_IP}" "sudo systemctl enable rke2-server"
runSSH "${RKE2_SERVER_ONE_IP}" "sudo systemctl start rke2-server"

if [ -n "$RANCHER_AGENT_IMAGE" ]; then
  setupDaemonFunction=$(declare -f setupDockerDaemon)
  runSSH "${RKE2_SERVER_ONE_IP}" "${setupDaemonFunction}; setupDockerDaemon"
  runSSH "${RKE2_SERVER_ONE_IP}" "sudo systemctl restart docker && sudo systemctl daemon-reload"

  runSSH "${RKE2_SERVER_ONE_IP}" "sudo docker pull ${REGISTRY}/${RANCHER_IMAGE}:${RANCHER_TAG_VERSION}"
  runSSH "${RKE2_SERVER_ONE_IP}" "sudo docker pull ${REGISTRY}/${RANCHER_AGENT_IMAGE}:${RANCHER_TAG_VERSION}"
  runSSH "${RKE2_SERVER_ONE_IP}" "sudo systemctl restart rke2-server"
fi

runSSH "${RKE2_SERVER_ONE_IP}" "sudo mkdir -p /home/${USER}/.kube"
runSSH "${RKE2_SERVER_ONE_IP}" "sudo cp /etc/rancher/rke2/rke2.yaml /home/${USER}/.kube/config"
runSSH "${RKE2_SERVER_ONE_IP}" "sudo chown -R ${USER}:${GROUP} /home/${USER}/.kube"

mkdir -p ~/.kube
ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i ${PEM_FILE} ${USER}@${RKE2_SERVER_ONE_IP} "sudo cat /home/${USER}/.kube/config" > ~/.kube/config
sed -i "s|server: https://127.0.0.1:6443|server: https://${RKE2_SERVER_ONE_IP}:6443|" ~/.kube/config
kubectl get nodes