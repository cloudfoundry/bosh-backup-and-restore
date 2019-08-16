package bosh

import (
	"fmt"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/instance"
	"github.com/cppforlife/go-patch/patch"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type BoshManifestQuerier struct {
	manifest   interface{}
	v2Manifest bool
}

func (mq BoshManifestQuerier) FindReleaseName(instanceGroupName, jobName string) (string, error) {
	var releasePath string
	if mq.v2Manifest {
		releasePath = fmt.Sprintf("/instance_groups/name=%s/jobs/name=%s/release", instanceGroupName, jobName)
	} else {
		releasePath = fmt.Sprintf("/jobs/name=%s/templates/name=%s/release", instanceGroupName, jobName)
	}

	releasePointer, err := patch.NewPointerFromString(releasePath)
	if err != nil {
		return "", errors.Wrap(err, fmt.Sprintf("error finding release name for job %s in instance group %s", jobName, instanceGroupName))
	}

	release, err := patch.FindOp{Path: releasePointer}.Apply(mq.manifest)
	if err != nil {
		return "", errors.Wrap(err, fmt.Sprintf("error finding release name for job %s in instance group %s", jobName, instanceGroupName))
	}

	return release.(string), nil
}

func NewBoshManifestQuerier(manifest string) (instance.ManifestQuerier, error) {
	var parsedManifest interface{}

	err := yaml.Unmarshal([]byte(manifest), &parsedManifest)
	if err != nil {
		return nil, errors.Wrap(err, "error unmarshalling manifest yaml")
	}

	v2Manifest := isV2Manifest(parsedManifest)

	return BoshManifestQuerier{manifest: parsedManifest, v2Manifest: v2Manifest}, nil
}

func (mq BoshManifestQuerier) IsJobBackupOneRestoreAll(instanceGroupName, jobName string) (bool, error) {
	var jobPath, backupOneRestoreAllPropertyPath string
	if mq.v2Manifest {
		jobPath = fmt.Sprintf("/instance_groups/name=%s/jobs/name=%s", instanceGroupName, jobName)
		backupOneRestoreAllPropertyPath = fmt.Sprintf("%s/properties/bbr/backup_one_restore_all", jobPath)

	} else {
		jobPath = fmt.Sprintf("/jobs/name=%s/templates/name=%s", instanceGroupName, jobName)
		backupOneRestoreAllPropertyPath = fmt.Sprintf("/jobs/name=%s/properties/bbr/backup_one_restore_all", instanceGroupName)
	}

	jobPathPointer, _ := patch.NewPointerFromString(jobPath)
	_, err := patch.FindOp{Path: jobPathPointer}.Apply(mq.manifest)
	if err != nil {
		return false, errors.Wrap(err, fmt.Sprintf("error finding job %s in instance group %s", jobName, instanceGroupName))
	}

	backupOneRestoreAllPropertyPointer, _ := patch.NewPointerFromString(backupOneRestoreAllPropertyPath)
	backupOneRestoreAll, err := patch.FindOp{Path: backupOneRestoreAllPropertyPointer}.Apply(mq.manifest)
	if err != nil {
		return false, nil
	}

	return backupOneRestoreAll.(bool), nil
}

func isV2Manifest(manifest interface{}) bool {
	instanceGroupPath := patch.MustNewPointerFromString(fmt.Sprintf("/instance_groups"))
	_, err := patch.FindOp{Path: instanceGroupPath}.Apply(manifest)
	if err != nil {
		return false
	}
	return true
}
