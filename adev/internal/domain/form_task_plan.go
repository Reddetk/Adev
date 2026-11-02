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
	// ContainerRefs are the HLD containers the task changes or calls. The system
	// copies only the relations that touch them into the plan.
	ContainerRefs []string `json:"containerRefs,omitempty"`
	DependsOn     []string `json:"dependsOn,omitempty"`
}

// TaskPlan is the plan of implementation tasks with acceptance criteria.
type TaskPlan struct {
	Tasks        []PlanTask `json:"tasks"`
	Roots        []string   `json:"roots,omitempty"`
	CriticalPath []string   `json:"criticalPath,omitempty"`
	// ContextRelations are the HLD edges the tasks touch. The system writes the
	// subset: a relation whose endpoint no task names is not a test premise.
	ContextRelations []ContextRelation `json:"contextRelations,omitempty"`
}

// RelationsForTasks keeps the HLD edges whose endpoint is a container named by a
// task. A task with no container reference has no integration premise, so it gets
// no relation; a relation between two untouched containers stays out.
func RelationsForTasks(h HLD, tasks []PlanTask) []ContextRelation {
	wanted := map[string]bool{}
	for _, task := range tasks {
		for _, containerID := range task.ContainerRefs {
			wanted[containerID] = true
		}
	}
	relations := make([]ContextRelation, 0)
	if len(wanted) == 0 {
		return relations
	}
	seen := map[string]bool{}
	for _, relation := range h.ContextRelations {
		if !wanted[relation.From] && !wanted[relation.To] {
			continue
		}
		key := relation.From + "->" + relation.To + "|" + relation.CauseRef
		if seen[key] {
			continue
		}
		seen[key] = true
		relations = append(relations, relation)
	}
	return relations
}

// Texts returns every free-text value for the language checker.
func (p TaskPlan) Texts() []Text {
	var out []Text
	for _, task := range p.Tasks {
		out = append(out, task.Title)
		for _, acceptance := range task.Acceptance {
			out = append(out, Text{Value: acceptance.Text})
		}
	}
	return out
}
