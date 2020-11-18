package orchestrator

func NewSkipStep(logger Logger, name string) *SkipStep {
	return &SkipStep{
		logger: logger,
		name:   name,
	}
}

type SkipStep struct {
	name   string
	logger Logger
}

func (s *SkipStep) Run(session *Session) error {
	s.logger.Info("bbr", "Skipping %s for deployment", s.name)
	return nil
}
