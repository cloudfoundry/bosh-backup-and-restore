package bosh

import (
	"fmt"

	"github.com/cppforlife/go-patch/patch"
	yaml "gopkg.in/yaml.v1"
)

func NewReleaseMapping(manifest string, instanceNames []string) ReleaseMapping {
	var parsedManifest interface{}

	err := yaml.Unmarshal([]byte(manifest), &parsedManifest)
	if err != nil {
		panic(err)
	}

	releaseMapping := make(ReleaseMapping)
	for _, igName := range instanceNames {
		instCount := instanceCount(parsedManifest, igName)
		if instCount == 0 {
			continue
		}
		jobs := jobs(parsedManifest, igName)
		for _, j := range jobs {
			rn := release(parsedManifest, igName, j)
			if _, ok := releaseMapping[igName]; !ok {
				releaseMapping[igName] = map[string]string{}
			}
			releaseMapping[igName][j] = rn
		}
	}
	return releaseMapping
}

type ReleaseMapping map[string]map[string]string

func instanceCount(manifest interface{}, instanceName string) int {
	countPath := patch.MustNewPointerFromString(fmt.Sprintf("/instance_groups/name=%s/instances", instanceName))
	count, err := patch.FindOp{Path: countPath}.Apply(manifest)
	if err != nil {
		panic(err)
	}
	return count.(int)
}

func jobs(manifest interface{}, instanceName string) []string {
	i := 0
	jobs := []string{}
	for {
		jobPath := patch.MustNewPointerFromString(fmt.Sprintf("/instance_groups/name=%s/jobs/%v/name", instanceName, i))
		j, err := patch.FindOp{Path: jobPath}.Apply(manifest)
		if err != nil {
			return jobs
		}
		jobs = append(jobs, j.(string))
		i++
	}
	return jobs
}

func release(manifest interface{}, instanceName string, jobName string) string {
	releasePath := patch.MustNewPointerFromString(fmt.Sprintf("/instance_groups/name=%s/jobs/name=%s/release", instanceName, jobName))
	release, err := patch.FindOp{Path: releasePath}.Apply(manifest)
	if err != nil {
		panic(err)
	}
	return release.(string)
}
