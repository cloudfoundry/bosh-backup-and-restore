package ssh

import (
	"github.com/cloudfoundry/bosh-utils/uuid"
	"github.com/cloudfoundry/bosh-cli/director"
)

//go:generate counterfeiter -o fakes/fake_opts_generator.go . SSHOptsGenerator
type SSHOptsGenerator func(uuidGen uuid.Generator) (director.SSHOpts, string, error)
