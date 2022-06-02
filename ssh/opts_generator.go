package ssh

import (
	"github.com/cloudfoundry/bosh-cli/v6/director"
	"github.com/cloudfoundry/bosh-utils/uuid"
)

//go:generate counterfeiter -o fakes/fake_opts_generator.go . SSHOptsGenerator
type SSHOptsGenerator func(uuidGen uuid.Generator) (director.SSHOpts, string, error)
