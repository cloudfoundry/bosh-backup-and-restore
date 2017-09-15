package orchestrator

import (
	"time"
)

type AddFinishTimeStep struct {
	nowFunc func() time.Time
}

func NewAddFinishTimeStep(nowFunc func() time.Time) Step {
	return &AddFinishTimeStep{
		nowFunc: nowFunc,
	}
}

func (s *AddFinishTimeStep) Run(session *Session) error {
	if session.CurrentArtifact() != nil {
		return session.CurrentArtifact().AddFinishTime(s.nowFunc())
	}

	return nil
}
