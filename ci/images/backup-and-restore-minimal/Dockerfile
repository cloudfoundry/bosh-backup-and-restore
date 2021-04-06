FROM ubuntu:bionic

ENV go_version 1.16.3
ENV cf_cli_version 6.46.0
ENV bosh_cli_version 6.0.0
ENV om_cli_version 0.42.0
ENV om_cli_6_version 6.2.0

RUN apt-get update && apt-get install -y \
  apt-utils \
  curl \
  gcc \
  git \
  gnupg \
  g++ \
  jq \
  make \
  parallel \
  sed \
  sshuttle \
  sudo \
  unzip \
  vim \
  wget \
  netcat-openbsd \
  && rm -rf /var/lib/apt/lists/*

RUN \
  git config --global user.name "root" && \
  git config --global user.email "root"

ENV GOPATH /go
ENV PATH /go/bin:/usr/local/go/bin:$PATH
RUN \
  wget https://golang.org/dl/go${go_version}.linux-amd64.tar.gz -P /tmp && \
  tar xzvf /tmp/go${go_version}.linux-amd64.tar.gz -C /usr/local && \
  mkdir ${GOPATH} && \
  rm -rf /tmp/*

RUN go get github.com/onsi/ginkgo/ginkgo
RUN go get github.com/onsi/gomega

# dep
RUN \
  wget https://github.com/golang/dep/releases/download/v0.5.0/dep-linux-amd64 --output-document="/usr/bin/dep" && \
  chmod +x /usr/bin/dep

# OM-cli
RUN \
  wget https://github.com/pivotal-cf/om/releases/download/${om_cli_version}/om-linux --output-document="/usr/bin/om" && \
  chmod +x /usr/bin/om

# OM-cli-6
RUN \
  wget https://github.com/pivotal-cf/om/releases/download/${om_cli_6_version}/om-linux-6.2.0 --output-document="/usr/bin/om-6" && \
  chmod +x /usr/bin/om-6

# bosh-cli
RUN \
  wget https://s3.amazonaws.com/bosh-cli-artifacts/bosh-cli-${bosh_cli_version}-linux-amd64 --output-document="/usr/bin/bosh" && \
  chmod +x /usr/bin/bosh && \
  cp /usr/bin/bosh /usr/bin/bosh-cli && \
  chmod +x /usr/bin/bosh-cli

# cf-cli
WORKDIR /tmp
RUN \
  wget -q -O cf.deb "https://cli.run.pivotal.io/stable?release=debian64&version=${cf_cli_version}&source=github-rel" && \
  dpkg -i cf.deb && \
  rm cf.deb
