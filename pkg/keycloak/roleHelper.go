package keycloak

type RoleHelper struct {
	roleTemplatePath string
}

type RoleParameters struct {
	RoleName string
}

func NewRoleHelper() *RoleHelper {
	return &RoleHelper{
		roleTemplatePath: templatePath(),
	}
}

func (r *RoleHelper) LoadRoleTemplate(params RoleParameters) ([]byte, error) {
	return loadTemplate("role", r.roleTemplatePath, params)
}
