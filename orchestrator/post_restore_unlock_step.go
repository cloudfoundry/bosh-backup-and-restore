package orchestrator

type PostRestoreUnlockStep struct{}

func NewPostRestoreUnlockStep() Step {
	return &PostRestoreUnlockStep{}
}

func (s *PostRestoreUnlockStep) Run(session *Session) error {
	err := session.CurrentDeployment().PostRestoreUnlock()

	if err != nil {
		return NewPostUnlockError(err.Error())
	}

	return nil
}
