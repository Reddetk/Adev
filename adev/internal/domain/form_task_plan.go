package domain

// PlanTaskType is the kind of work a task carries.
type PlanTaskType string

const (
	TaskFeature  PlanTaskType = "feature"
	TaskFix      PlanTaskType = "fix"
	TaskRefactor PlanTaskType = "refactor"
	TaskConfig   PlanTaskType = "config"
	TaskDocs     PlanTaskType = "docs"
	TaskTest     PlanTaskType = "test"
)

// PlanTaskOwner is the single agent that runs the task.
type PlanTaskOwner string

const (
	OwnerDev    PlanTaskOwner = "dev"
	OwnerDevOps PlanTaskOwner = "devops"
)

// PlanTask is one unit of work with a single owner.
type PlanTask struct {
	ID             string        `json:"id"`
	Title          Text          `json:"title"`
	TaskType       PlanTaskType  `json:"taskType"`
	ParentID       string        `json:"parentId,omitempty"`
	Owner          PlanTaskOwner `json:"owner"`
	Inputs         []Ref         `json:"inputs,omitempty"`
	ContractRefs   []string      `json:"contractRefs,omitempty"`
	Outputs        []Ref         `json:"outputs,omitempty"`
	Acceptance     []Statement   `json:"acceptance"`
	AcceptanceRefs []string      `json:"acceptanceRefs,omitempty"`
	DependsOn      []string      `json:"dependsOn,omitempty"`
}

// TaskPlan is the plan of implementation tasks with acceptance criteria.
type TaskPlan struct {
	Tasks        []PlanTask `json:"tasks"`
	Roots        []string   `json:"roots,omitempty"`
	CriticalPath []string   `json:"criticalPath,omitempty"`
}

// Texts returns every free-text value for the language checker.
func (p TaskPlan) Texts() []Text {
	var out []Text
	for _, task := range p.Tasks {
		out = append(out, task.Title)
		for _, acceptance := range task.Acceptance {
			out = append(out, Text{Value: acceptance.Text, Lang: LangSTE100})
		}
	}
	return out
}
