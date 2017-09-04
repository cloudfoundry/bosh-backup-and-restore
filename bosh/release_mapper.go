package bosh

import (
	"fmt"

	"github.com/cppforlife/go-patch/patch"
	yaml "gopkg.in/yaml.v1"
)

//go:generate counterfeiter -o fakes/fake_release_mapper.go . ReleaseMapper
type ReleaseMapper interface {
	NewReleaseMapping(manifest string, instanceNames []string) ReleaseMapping
}

type releaseMapper struct{}

func NewReleaseMapper() *releaseMapper {
	return &releaseMapper{}
}

func (rm releaseMapper) NewReleaseMapping(manifest string, instanceNames []string) ReleaseMapping {
	var parsedManifest interface{}

	err := yaml.Unmarshal([]byte(manifest), &parsedManifest)
	if err != nil {
		panic(err)
	}

	releaseMapping := make(ReleaseMapping)
	isV2Manifest := v2Manifest(parsedManifest)

	for _, igName := range instanceNames {
		instCount := instanceCount(parsedManifest, igName, isV2Manifest)
		if instCount == 0 {
			continue
		}
		jobs := jobs(parsedManifest, igName, isV2Manifest)
		for _, j := range jobs {
			rn := release(parsedManifest, igName, j, isV2Manifest)
			if _, ok := releaseMapping[igName]; !ok {
				releaseMapping[igName] = map[string]string{}
			}
			releaseMapping[igName][j] = rn
		}
	}
	return releaseMapping
}

type ReleaseMapping map[string]map[string]string

func v2Manifest(manifest interface{}) bool {
	uuidPath := patch.MustNewPointerFromString(fmt.Sprintf("/director_uuid"))
	_, err := patch.FindOp{Path: uuidPath}.Apply(manifest)
	if err != nil {
		return true
	}
	return false
}

func instanceCount(manifest interface{}, instanceName string, v2 bool) int {
	var countPathStr string
	if v2 {
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

func jobs(manifest interface{}, instanceName string, v2 bool) []string {
	i := 0
	jobs := []string{}
	for {
		var jobPathStr string
		if v2 {
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

func release(manifest interface{}, instanceName string, jobName string, v2 bool) string {
	var releasePathStr string
	if v2 {
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
