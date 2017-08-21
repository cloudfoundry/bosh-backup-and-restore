# BOSH Backup and Restore

BOSH Backup and Restore is a CLI utility for orchestrating the backup and restore of [BOSH](https://bosh.io/) deployments and BOSH directors. It orchestrates triggering the backup or restore process on the deployment or director, and transfers the backup artifact to and from the deployment or director.

This repository contains the source code for BOSH Backup and Restore.

## Developing BBR locally

Running `go get` on the BBR repo will not work, as we use [Glide](https://github.com/Masterminds/glide) to manage our dependencies. Instead:

1. `mkdir -p $GOPATH/src/github.com/cloudfoundry-incubator`
1. `git clone git@github.com:cloudfoundry-incubator/bosh-backup-and-restore.git`
1. `cd $GOPATH/src/github.com/cloudfoundry-incubator/bosh-backup-and-restore`
1. `make setup`

You're good to go. Run tests locally with `make test`.

## Additional information

**Docs**: http://docs.cloudfoundry.org/bbr/index.html

**Slack**: #bbr on https://slack.cloudfoundry.org

Cloud Foundry Summit talk on BBR https://www.youtube.com/watch?v=HlO9L9iE9T8

Blog post about BBR https://content.pivotal.io/blog/cloud-native-recovery-tool-bosh-backup-restore-now-available-in-public-beta
