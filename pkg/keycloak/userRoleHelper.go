package keycloak

type UserRoleHelper struct {
	userRoleTemplatePath string
}

type UserRoleParameters struct {
	RoleName string
	RoleId   string
}

func NewUserRoleHelper() *UserRoleHelper {
	return &UserRoleHelper{
		userRoleTemplatePath: templatePath(),
	}
}

func (r *UserRoleHelper) LoadUserRoleTemplate(params UserRoleParameters) ([]byte, error) {
	return loadTemplate("user-role", r.userRoleTemplatePath, params)
}
