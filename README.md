# BOSH Backup and Restore

BOSH backup and restore is a CLI for orchestrating the backup and restore of BOSH deployments and BOSH directors. It orchestrates triggering the backup or restore process on the deployment or director, and transfers the backup artifact to and from the deployment or director.

## User Documentation
User documentation can be found [here](http://www.boshbackuprestore.io/). Documentation for service authors wishing to implement backup and restore flows in their release can be found [here](http://www.boshbackuprestore.io/bosh-backup-and-restore/release_author_guide.html).

## Contributing

Run tests with `make test` before committing. System tests can be run locally with `make sys-test-local`.
