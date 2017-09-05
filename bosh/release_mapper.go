package bosh

import (
	"fmt"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/instance"
	"github.com/cppforlife/go-patch/patch"
	"gopkg.in/yaml.v1"
)

type ManifestReleaseMapping struct {
	releaseMap map[string]map[string]string
}

func (rm ManifestReleaseMapping) FindReleaseName(instanceGroupName, jobName string) (string, error) {
	jobReleaseMap, jobReleaseMapFound := rm.releaseMap[instanceGroupName]
	if !jobReleaseMapFound {
		return "", fmt.Errorf("can't find instance group %s in release mapping, %v", instanceGroupName, rm.releaseMap)
	}
	releaseName, releaseNameFound := jobReleaseMap[jobName]
	if !releaseNameFound {
		return "", fmt.Errorf("can't find job name %s in release mapping, %v", jobName, rm.releaseMap)
	}
	return releaseName, nil
}

func NewBoshManifestReleaseMapping(manifest string, instanceNames []string) instance.ReleaseMapping {
	var parsedManifest interface{}

	err := yaml.Unmarshal([]byte(manifest), &parsedManifest)
	if err != nil {
		panic(err)
	}

	releaseMap := make(map[string]map[string]string)
	isV2Manifest := v2Manifest(parsedManifest)

	for _, igName := range instanceNames {
		instCount := instanceCount(parsedManifest, igName, isV2Manifest)
		if instCount == 0 {
			continue
		}
		jobs := jobs(parsedManifest, igName, isV2Manifest)
		for _, j := range jobs {
			rn := release(parsedManifest, igName, j, isV2Manifest)
			if _, ok := releaseMap[igName]; !ok {
				releaseMap[igName] = map[string]string{}
			}
			releaseMap[igName][j] = rn
		}
	}

	return ManifestReleaseMapping{releaseMap: releaseMap}
}

func v2Manifest(manifest interface{}) bool {
	uuidPath := patch.MustNewPointerFromString(fmt.Sprintf("/director_uuid"))
	_, err := patch.FindOp{Path: uuidPath}.Apply(manifest)
	if err != nil {
		return true
	}
	return false
}

func instanceCount(manifest interface{}, instanceName string, v2Manifest bool) int {
	var countPathStr string
	if v2Manifest {
		countPathStr = fmt.Sprintf("/instance_groups/name=%s/instances", instanceName)
	} else {
		countPathStr = fmt.Sprintf("/jobs/name=%s/instances", instanceName)
	}
	countPath := patch.MustNewPointerFromString(countPathStr)
	count, err := patch.FindOp{Path: countPath}.Apply(manifest)
	if err != nil {
		panic(err)
	}
	return count.(int)
}

func jobs(manifest interface{}, instanceName string, v2Manifest bool) []string {
	i := 0
	jobs := []string{}
	for {
		var jobPathStr string
		if v2Manifest {
			jobPathStr = fmt.Sprintf("/instance_groups/name=%s/jobs/%v/name", instanceName, i)
		} else {
			jobPathStr = fmt.Sprintf("/jobs/name=%s/templates/%v/name", instanceName, i)
		}
		jobPath := patch.MustNewPointerFromString(jobPathStr)
		j, err := patch.FindOp{Path: jobPath}.Apply(manifest)
		if err != nil {
			return jobs
		}
		jobs = append(jobs, j.(string))
		i++
	}
	return jobs
}

func release(manifest interface{}, instanceName string, jobName string, v2Manifest bool) string {
	var releasePathStr string
	if v2Manifest {
		releasePathStr = fmt.Sprintf("/instance_groups/name=%s/jobs/name=%s/release", instanceName, jobName)
	} else {
		releasePathStr = fmt.Sprintf("/jobs/name=%s/templates/name=%s/release", instanceName, jobName)
	}
	releasePath := patch.MustNewPointerFromString(releasePathStr)
	release, err := patch.FindOp{Path: releasePath}.Apply(manifest)
	if err != nil {
		panic(err)
	}
	return release.(string)
}
