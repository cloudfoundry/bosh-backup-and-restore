package orchestrator

type CleanupPreviousStep struct{}

func NewCleanupPreviousStep() Step {
	return &CleanupPreviousStep{}
}

func (s *CleanupPreviousStep) Run(session *Session) error {
	return session.CurrentDeployment().CleanupPrevious()
}
