# BOSH Backup and Restore

BOSH Backup and Restore is a CLI utility for orchestrating the backup and restore of [BOSH](https://bosh.io/) deployments and BOSH directors. It orchestrates triggering the backup or restore process on the deployment or director, and transfers the backup artifact to and from the deployment or director.

This repository contains the source code for BOSH Backup and Restore.

## CI Status

BBR build status [![BBR Build Status Badge](https://backup-and-restore.ci.cf-app.com/api/v1/teams/main/pipelines/bbr/jobs/build-rc/badge)](https://backup-and-restore.ci.cf-app.com/teams/main/pipelines/bbr)

## Developing BBR locally

We use [dep](https://github.com/golang/dep) to manage our dependencies, so run:

1. `go get github.com/cloudfoundry-incubator/bosh-backup-and-restore`
1. `cd $GOPATH/src/github.com/cloudfoundry-incubator/bosh-backup-and-restore`
1. `make setup`

You're good to go. Run tests locally with `make test`.

## Additional information

**Docs**: http://docs.cloudfoundry.org/bbr/index.html

**Slack**: #bbr on https://slack.cloudfoundry.org

Cloud Foundry Summit talk on BBR https://www.youtube.com/watch?v=rQSLNHAHgA8

Blog posts about BBR https://content.pivotal.io/blog/cloud-native-recovery-tool-bosh-backup-restore-now-available-in-public-beta


