package domain

// Role is an agent role. A role selects the projection of a form.
type Role string

const (
	RoleOperator  Role = "operator"
	RoleAnalyst   Role = "analyst"
	RoleArchitect Role = "architect"
	RoleQA        Role = "qa"
	RoleDev       Role = "dev"
	RoleDevOps    Role = "devops"
)

// Valid reports whether the role is known.
func (r Role) Valid() bool {
	switch r {
	case RoleOperator, RoleAnalyst, RoleArchitect, RoleQA, RoleDev, RoleDevOps:
		return true
	}
	return false
}
