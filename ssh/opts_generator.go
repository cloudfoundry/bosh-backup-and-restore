package ssh

import (
	"github.com/cloudfoundry/bosh-cli/v7/director"
	"github.com/cloudfoundry/bosh-utils/uuid"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate
//counterfeiter:generate -o fakes/fake_opts_generator.go . SSHOptsGenerator
type SSHOptsGenerator func(uuidGen uuid.Generator) (director.SSHOpts, string, error)
