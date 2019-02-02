package keycloak

type RealmHelper struct {
	realmTemplatePath string
}

type RealmParameters struct {
	RealmId string
}

type RealmClientParameters struct {
	ClientId                string
	RootUrl                 string
	AdminUrl                string
	RedirectUris            string
	WebOrigin               string
	BearerOnly              string
	ImplicitFlowEnabled     string
	DirectAcessGrantEnabled string
	PublicClient            string
}

func NewRealmHelper() *RealmHelper {
	return &RealmHelper{
		realmTemplatePath: templatePath(),
	}
}

func (r *RealmHelper) LoadRealmTemplate(params RealmParameters) ([]byte, error) {
	return loadTemplate("realm", r.realmTemplatePath, params)
}

func (r *RealmHelper) LoadRealmClientTemplate(params RealmClientParameters) ([]byte, error) {
	return loadTemplate("realm-client", r.realmTemplatePath, params)
}
