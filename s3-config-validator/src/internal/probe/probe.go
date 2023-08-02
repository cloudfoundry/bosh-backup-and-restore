package probe

import (
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/s3-config-validator/src/internal/s3"
)

type Probe func(bucket string) error

type NamedProbe struct {
	Name  string
	Probe Probe
}

type Set []NamedProbe

func NewSet(s3 s3.Client, readOnly bool, versioned bool) Set {

	var probeSet Set

	if versioned {
		probeSet = []NamedProbe{
			{
				Name:  "Bucket is versioned",
				Probe: s3.IsVersioned,
			},
			{
				Name:  "Can list object versions",
				Probe: s3.CanListObjectVersions,
			},
			{
				Name:  "Can get object versions",
				Probe: s3.CanGetObjectVersions,
			},
		}
	} else {
		probeSet = []NamedProbe{
			{
				Name:  "Bucket is not versioned",
				Probe: s3.IsUnversioned,
			},
			{
				Name:  "Can list objects",
				Probe: s3.CanListObjects,
			},
			{
				Name:  "Can get objects",
				Probe: s3.CanGetObjects,
			},
		}
	}

	if !readOnly {
		probeSet = append(
			probeSet,
			NamedProbe{
				Name:  "Can put objects",
				Probe: s3.CanPutObjects,
			},
		)
	}

	return probeSet
}
