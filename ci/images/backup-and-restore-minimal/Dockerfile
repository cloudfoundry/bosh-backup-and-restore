FROM golang:1.23.6 as go-binary

FROM ubuntu:jammy

ENV cf_cli_version 8.10.0
ENV bosh_cli_version 7.9.3
ENV om_cli_version 7.14.0

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

RUN curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | gpg --dearmor -o /usr/share/keyrings/githubcli-archive-keyring.gpg
RUN echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" | tee /etc/apt/sources.list.d/github-cli.list
RUN apt-get update && apt-get install -y gh

RUN \
  git config --global user.name "root" && \
  git config --global user.email "root"

ENV GOPATH /go
ENV PATH /go/bin:/usr/local/go/bin:$PATH
COPY --from=go-binary /usr/local/go /usr/local/go

# dep
RUN \
  wget --quiet https://github.com/golang/dep/releases/download/v0.5.0/dep-linux-amd64 --output-document="/usr/bin/dep" && \
  chmod +x /usr/bin/dep

# OM-cli
RUN \
  wget --quiet https://github.com/pivotal-cf/om/releases/download/${om_cli_version}/om-linux --output-document="/usr/bin/om" && \
  chmod +x /usr/bin/om

# bosh-cli
RUN \
  wget --quiet https://github.com/cloudfoundry/bosh-cli/releases/download/v${bosh_cli_version}/bosh-cli-${bosh_cli_version}-linux-amd64 --output-document="/usr/bin/bosh" && \
  chmod +x /usr/bin/bosh && \
  cp /usr/bin/bosh /usr/bin/bosh-cli && \
  chmod +x /usr/bin/bosh-cli

# cf-cli
WORKDIR /tmp
RUN \
  wget --quiet -O cf.deb "https://cli.run.pivotal.io/stable?release=debian64&version=${cf_cli_version}&source=github-rel" && \
  dpkg -i cf.deb && \
  rm cf.deb

# cf-v6-cli for use with older versions of CF
WORKDIR /tmp
RUN \
    wget --quiet -O cf6.tgz "https://packages.cloudfoundry.org/stable?release=linux64-binary&source=github&version=v6" && \
    tar --extract --file=cf6.tgz cf && \
    mv cf /usr/bin/cf6 && \
    rm cf6.tgz

RUN apt-get update && apt-get install -y --no-install-recommends \
  libxml2-utils \
  && rm -rf /var/lib/apt/lists/*
