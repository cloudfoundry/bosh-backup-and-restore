(test)
# BOSH Backup and Restore

BOSH Backup and Restore is a CLI utility for orchestrating the backup and restore of [BOSH](https://bosh.io/) deployments and BOSH directors. It orchestrates triggering the backup or restore process on the deployment or director, and transfers the backup artifact to and from the deployment or director.

This repository contains the source code for BOSH Backup and Restore.

## Install

The latest BBR binaries for Linux and macOS are available to download on the [releases page](https://github.com/cloudfoundry-incubator/bosh-backup-and-restore/releases).

On macOS, you can install BBR using [Homebrew](http://brew.sh/):

1. `brew tap cloudfoundry/tap`
1. `brew install bbr`

## CI Status

BOSH Backup and Restore build status [![BBR Build Status Badge](https://ci.cryo.cf-app.com/api/v1/teams/bosh-backup-restore/pipelines/bbr/jobs/build-rc/badge)](https://ci.cryo.cf-app.com/teams/bosh-backup-restore/pipelines/bbr)

## Developing BBR locally

We use [go modules](https://blog.golang.org/using-go-modules) to manage our dependencies, so run:

1. `git clone git@github.com:cloudfoundry-incubator/bosh-backup-and-restore.`
1. `go mod download`

Unit tests make use of Docker containers to test networking. Ensure that Docker is running in your environment, before attempting to run tests.
These can be run with the command: `make test`.

Note: if you see timeout errors occurring in the SSH suite of unit tests this may be due to the number of nodes the tests are running on. You can run `ginkgo -nodes=<number> -r` to reduce the number of nodes and decrease the concurrent requests being made to the Docker Daemon which will slow when under load.

## Additional information

**Docs:** http://docs.cloudfoundry.org/bbr/index.html

**Slack:** `#bbr` on https://slack.cloudfoundry.org

**Talks:**
- [_Burning Down the House: How to Plan for and Deal with Disaster Recovery in CF_](https://www.youtube.com/watch?v=rQSLNHAHgA8) at Cloud Foundry Summit Europe 2017
- [_Extending the BOSH Backup and Restore Framework_](https://www.youtube.com/watch?v=LiXXqrdlXSQ) at Cloud Foundry Summit 2018
- [_Reviving the platform every day_](https://www.youtube.com/watch?v=8osX_c1XQyI) at Cloud Foundry Summit EU 2018

**Blog posts:**
- [Cloud-Native Recovery Tool, BOSH Backup & Restore, Now Available in Public Beta](https://content.pivotal.io/blog/cloud-native-recovery-tool-bosh-backup-restore-now-available-in-public-beta) on the Pivotal Blog
- [TUTORIAL: Automating ERT Backups with BBR and Concourse](https://content.pivotal.io/blog/tutorial-automating-ert-backups-with-bbr-and-concourse)
