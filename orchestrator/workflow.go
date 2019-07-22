package orchestrator

type Workflow struct {
	StartingNode *Node
	Nodes        []*Node
}

func NewWorkflow() *Workflow {
	return &Workflow{}
}

func (workflow *Workflow) Run(session *Session) Error {
	var errs Error
	currentNode := workflow.StartingNode

	for currentNode != nil {
		err := currentNode.step.Run(session)
		if err != nil {
			errs = append(errs, err)
			currentNode = workflow.findNode(currentNode.failStep)
		} else {
			currentNode = workflow.findNode(currentNode.successStep)
		}
	}

	return errs
}

func (workflow *Workflow) findNode(step Step) *Node {
	if step == nil {
		return nil
	}
	for _, value := range workflow.Nodes {
		if value.step == step {
			return value
		}
	}
	//TODO: replace with something else
	panic("node not found")
}

func (workflow *Workflow) Add(step Step) *Node {
	node := NewNode(step)
	workflow.Nodes = append(workflow.Nodes, node)
	return node
}

func (workflow *Workflow) StartWith(step Step) *Node {
	node := workflow.Add(step)
	workflow.StartingNode = node
	return node
}

type Step interface {
	Run(*Session) error
}

type Node struct {
	step        Step
	successStep Step
	failStep    Step
}

func NewNode(step Step) *Node {
	return &Node{step: step}
}
func (node *Node) OnSuccessOrFailure(step Step) *Node {
	return node.OnSuccess(step).OnFailure(step)
}

func (node *Node) OnFailure(failStep Step) *Node {
	node.failStep = failStep
	return node
}

func (node *Node) OnSuccess(successStep Step) *Node {
	node.successStep = successStep
	return node
}
