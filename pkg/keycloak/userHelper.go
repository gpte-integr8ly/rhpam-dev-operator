package keycloak

type UserHelper struct {
	userTemplatePath string
}

type UserParameters struct {
	Username string
	Password string
}

func NewUserHelper() *UserHelper {
	return &UserHelper{
		userTemplatePath: templatePath(),
	}
}

func (r *UserHelper) LoadUserTemplate(params UserParameters) ([]byte, error) {
	return loadTemplate("user", r.userTemplatePath, params)
}
