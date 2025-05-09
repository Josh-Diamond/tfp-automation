FROM registry.suse.com/bci/golang:1.23

ENV GOPATH /root/go
ENV PATH ${PATH}:/root/go/bin

ENV WORKSPACE ${GOPATH}/src/github.com/rancher/tfp-automation

WORKDIR $WORKSPACE

COPY . ./
SHELL ["/bin/bash", "-c"]

RUN go mod download && \
    go install gotest.tools/gotestsum@latest

ARG QASE_TEST_RUN_ID
ARG TERRAFORM_VERSION
ARG RKE_PROVIDER_VERSION
ARG RANCHER2_PROVIDER_VERSION
ARG LOCALS_PROVIDER_VERSION
ARG AWS_PROVIDER_VERSION
ARG LINODE_PROVIDER_VERSION
ARG KUBERNETES_PROVIDER_VERSION
ARG HARVESTER_PROVIDER_VERSION

ENV QASE_TEST_RUN_ID=${QASE_TEST_RUN_ID}
ENV TERRAFORM_VERSION=${TERRAFORM_VERSION}
ENV RKE_PROVIDER_VERSION=${RKE_PROVIDER_VERSION}
ENV RANCHER2_PROVIDER_VERSION=${RANCHER2_PROVIDER_VERSION}
ENV LOCALS_PROVIDER_VERSION=${LOCALS_PROVIDER_VERSION}
ENV AWS_PROVIDER_VERSION=${AWS_PROVIDER_VERSION}
ENV LINODE_PROVIDER_VERSION=${LINODE_PROVIDER_VERSION}
ENV HARVESTER_PROVIDER_VERSION=${HARVESTER_PROVIDER_VERSION}
ENV KUBERNETES_PROVIDER_VERSION=${KUBERNETES_PROVIDER_VERSION}

RUN zypper install -y openssh wget unzip > /dev/null

RUN wget https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_linux_amd64.zip -q && zypper --non-interactive update && \
    unzip terraform_${TERRAFORM_VERSION}_linux_amd64.zip > /dev/null && \ 
    rm terraform_${TERRAFORM_VERSION}_linux_amd64.zip > /dev/null && \
    chmod a+x terraform > /dev/null && mv terraform /usr/local/bin/terraform > /dev/null

ARG CONFIG_FILE
COPY $CONFIG_FILE /config.yml

RUN mkdir /root/.ssh && chmod 600 .ssh/jenkins-*
RUN for pem_file in .ssh/jenkins-*; do \
      ssh-keygen -f "$pem_file" -y > "/root/.ssh/$(basename "$pem_file").pub"; \
    done

RUN if [[ "$RANCHER2_PROVIDER_VERSION" == *"-rc"* ]]; then \
      chmod +x ./scripts/setup-provider.sh && ./scripts/setup-provider.sh rancher2 v${RANCHER2_PROVIDER_VERSION} ; \
    fi;

RUN if [[ "$RKE_PROVIDER_VERSION" == *"-rc"* ]]; then \
      chmod +x ./scripts/setup-provider.sh && ./scripts/setup-provider.sh rke v${RKE_PROVIDER_VERSION} ; \
    fi;