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

	releaseMapping := ReleaseMapping{}
	for _, igName := range instanceNames {
		i := 0
		for {
			releasePath := patch.MustNewPointerFromString(fmt.Sprintf("/instance_groups/name=%s/jobs/%v/release", igName, i))
			release, err := patch.FindOp{Path: releasePath}.Apply(parsedManifest)
			if err != nil {
				break
			}

			namePath := patch.MustNewPointerFromString(fmt.Sprintf("/instance_groups/name=%s/jobs/%v/name", igName, i))
			job, _ := patch.FindOp{Path: namePath}.Apply(parsedManifest)

			releaseMapping = add(releaseMapping, release.(string), job.(string))
			i++
		}
	}
	return releaseMapping
}

type ReleaseMapping map[string][]string

func add(rm map[string][]string, releaseName, jobName string) map[string][]string {
	if _, ok := rm[releaseName]; ok {
		for _, jn := range rm[releaseName] {
			if jn == jobName {
				return rm
			}
		}
	}
	rm[releaseName] = append(rm[releaseName], jobName)
	return rm
}
