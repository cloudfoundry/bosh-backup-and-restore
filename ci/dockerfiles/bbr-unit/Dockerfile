FROM alpine AS third-party-deps
RUN apk --update add curl --no-cache
RUN curl -L https://download.docker.com/linux/static/stable/x86_64/docker-20.10.9.tgz | tar xzv

FROM golang:1.21
COPY --from=third-party-deps /docker/* /usr/local/bin/

COPY ./ /bosh-backup-and-restore
WORKDIR /bosh-backup-and-restore
