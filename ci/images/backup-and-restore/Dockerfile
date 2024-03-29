FROM pcfplatformrecovery/backup-and-restore-minimal

ENV YQ_VERSION 2.4.0
ENV CLOUD_SDK_VERSION 429.0.0
ENV CREDHUB_CLI_VERSION 2.9.0
ENV BBL_VERSION v8.4.44
ENV VELERO_VERSION v1.6.1
ENV GH_VERSION 2.14.4
ENV PIVNET_VERSION 3.0.0
ENV RUBY_VERSION 3.3.0

# fixes error "Error response from daemon: client is newer than server"
ENV DOCKER_API_VERSION 1.41

ENV DEBIAN_FRONTEND=noninteractive

ARG TERRAFORM_VERSION

RUN apt-get update && apt-get install -y \
  apt-transport-https \
  awscli \
  autoconf \
  patch \
  build-essential \
  rustc \
  libyaml-dev \
  libreadline6-dev \
  libreadline-dev \
  libffi-dev \
  zlib1g-dev \
  libncurses5-dev \
  libgdbm6 \
  libgdbm-dev \
  libdb-dev \
  libssl-dev \
  uuid-dev \
  default-mysql-client \
  dnsutils \
  file \
  lsb-release \
  netcat-openbsd \
  openjdk-8-jdk \
  ruby-dev \
  ca-certificates \
  gettext-base \
  && rm -rf /var/lib/apt/lists/*

# Install azure cli

# install micrsoft signing key
RUN curl -sL https://packages.microsoft.com/keys/microsoft.asc | \
    gpg --dearmor | \
    sudo tee /etc/apt/trusted.gpg.d/microsoft.asc.gpg > /dev/null

# add azure cli repo
RUN AZ_REPO=$(lsb_release -cs) && \
    echo "deb [arch=amd64] https://packages.microsoft.com/repos/azure-cli/ $AZ_REPO main" | \
    sudo tee /etc/apt/sources.list.d/azure-cli.list

RUN apt-get update && apt-get install -y \
  azure-cli

# INSTALL RUBY
#
ENV RBENV_ROOT /home/vcap/.rbenv
RUN git clone https://github.com/rbenv/rbenv.git ${RBENV_ROOT}

RUN mkdir -p ${RBENV_ROOT}/plugins
RUN git clone https://github.com/rbenv/ruby-build.git ${RBENV_ROOT}/plugins/ruby-build
RUN git clone https://github.com/rbenv/rbenv-each.git ${RBENV_ROOT}/plugins/rbenv-each

ENV PATH ${RBENV_ROOT}/bin:${RBENV_ROOT}/shims:$PATH

RUN rbenv install ${RUBY_VERSION}
RUN rbenv global ${RUBY_VERSION}

RUN echo 'gem: --no-document' > $HOME/.gemrc
RUN rbenv each gem install bundler

# gcloud
RUN curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | gpg --dearmor -o /usr/share/keyrings/cloud.google.gpg &&  \
  echo "deb [signed-by=/usr/share/keyrings/cloud.google.gpg] https://packages.cloud.google.com/apt cloud-sdk main" | sudo tee -a /etc/apt/sources.list.d/google-cloud-sdk.list && \
  apt-get update && apt-get install -y python3 google-cloud-sdk=${CLOUD_SDK_VERSION}-0 && rm -rf /var/lib/apt/lists/*

# git-crypt
RUN git clone https://github.com/AGWA/git-crypt.git && \
  cd git-crypt/ && \
  CXXFLAGS="$(TARGET_CXXFLAGS) -std=c++11 -DOPENSSL_API_COMPAT=0x30000000L" make && sudo make install

# Bash testing framework
RUN go install github.com/progrium/basht@latest

RUN git config --global url."git@github.com:".insteadOf "https://github.com/" && \
  mkdir -p ~/.ssh && ssh-keyscan -H github.com > ~/.ssh/known_hosts

RUN wget https://download.docker.com/linux/static/stable/x86_64/docker-18.09.0.tgz && \
  tar xvf docker-18.09.0.tgz && \
  chmod +x docker/* && \
  sudo cp docker/* /bin

RUN wget "https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_linux_386.zip" && \
  unzip "terraform_${TERRAFORM_VERSION}_linux_386.zip" -d /bin/

# uaac
RUN gem install cf-uaac

# credhub
RUN \
  wget https://github.com/cloudfoundry-incubator/credhub-cli/releases/download/${CREDHUB_CLI_VERSION}/credhub-linux-${CREDHUB_CLI_VERSION}.tgz -P /tmp && \
  tar xzvf /tmp/credhub-linux-${CREDHUB_CLI_VERSION}.tgz -C /usr/local/bin && \
  chmod +x /usr/local/bin/credhub

# yq
RUN \
  wget https://github.com/mikefarah/yq/releases/download/${YQ_VERSION}/yq_linux_amd64 -P /tmp && \
  mv /tmp/yq_linux_amd64 /usr/local/bin/yq && \
  chmod +x /usr/local/bin/yq

# bbl
RUN \
  wget https://github.com/cloudfoundry/bosh-bootloader/releases/download/${BBL_VERSION}/bbl-${BBL_VERSION}_linux_x86-64 -P /tmp && \
  mv /tmp/bbl-${BBL_VERSION}_linux_x86-64 /usr/local/bin/bbl && \
  chmod +x /usr/local/bin/bbl

# github cli
RUN \
  wget https://github.com/cli/cli/releases/download/v${GH_VERSION}/gh_${GH_VERSION}_linux_amd64.tar.gz -P /tmp && \
  tar xzvf /tmp/gh_${GH_VERSION}_linux_amd64.tar.gz -C /tmp/ && \
  mv /tmp/gh_${GH_VERSION}_linux_amd64/bin/gh /usr/local/bin/gh && \
  chmod +x /usr/local/bin/gh

# velero
RUN \
  wget https://github.com/vmware-tanzu/velero/releases/download/${VELERO_VERSION}/velero-${VELERO_VERSION}-linux-amd64.tar.gz -P /tmp && \
  tar xzvf /tmp/velero-${VELERO_VERSION}-linux-amd64.tar.gz -C /tmp/ && \
  mv /tmp/velero-${VELERO_VERSION}-linux-amd64/velero /usr/local/bin/velero && \
  chmod +x /usr/local/bin/velero

# kubectl
ADD https://storage.googleapis.com/kubernetes-release/release/v1.21.2/bin/linux/amd64/kubectl /usr/bin/kubectl
RUN chmod +x /usr/bin/kubectl

RUN useradd --create-home --shell /bin/bash --home-dir /home/test test

# Pivnet cli
ADD https://github.com/pivotal-cf/pivnet-cli/releases/download/v${PIVNET_VERSION}/pivnet-linux-amd64-${PIVNET_VERSION} /usr/bin/pivnet
RUN chmod +x /usr/bin/pivnet

