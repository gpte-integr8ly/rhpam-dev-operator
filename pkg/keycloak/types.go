package keycloak

type KeycloakProtocolMapper struct {
	ID              string            `json:"id,omitempty"`
	Name            string            `json:"name,omitempty"`
	Protocol        string            `json:"protocol,omitempty"`
	ProtocolMapper  string            `json:"protocolMapper,omitempty"`
	ConsentRequired bool              `json:"consentRequired,omitempty"`
	ConsentText     string            `json:"consentText"`
	Config          map[string]string `json:"config"`
}

type KeycloakClient struct {
	*KeycloakApiClient
	OutputSecret *string `json:"outputSecret,omitempty"`
}

type KeycloakApiClient struct {
	ID                        string                   `json:"id,omitempty"`
	ClientID                  string                   `json:"clientId,omitempty"`
	Secret                    string                   `json:"secret"`
	Name                      string                   `json:"name"`
	BaseURL                   string                   `json:"baseUrl"`
	AdminURL                  string                   `json:"adminUrl"`
	RootURL                   string                   `json:"rootUrl"`
	Description               string                   `json:"description"`
	SurrogateAuthRequired     bool                     `json:"surrogateAuthRequired"`
	Enabled                   bool                     `json:"enabled"`
	ClientAuthenticatorType   string                   `json:"clientAuthenticatorType"`
	DefaultRoles              []string                 `json:"defaultRoles,omitempty"`
	RedirectUris              []string                 `json:"redirectUris,omitempty"`
	WebOrigins                []string                 `json:"webOrigins,omitempty"`
	NotBefore                 int                      `json:"notBefore"`
	BearerOnly                bool                     `json:"bearerOnly"`
	ConsentRequired           bool                     `json:"consentRequired"`
	StandardFlowEnabled       bool                     `json:"standardFlowEnabled"`
	ImplicitFlowEnabled       bool                     `json:"implicitFlowEnabled"`
	DirectAccessGrantsEnabled bool                     `json:"directAccessGrantsEnabled"`
	ServiceAccountsEnabled    bool                     `json:"serviceAccountsEnabled"`
	PublicClient              bool                     `json:"publicClient"`
	FrontchannelLogout        bool                     `json:"frontchannelLogout"`
	Protocol                  string                   `json:"protocol,omitempty"`
	Attributes                map[string]string        `json:"attributes,omitempty"`
	FullScopeAllowed          bool                     `json:"fullScopeAllowed"`
	NodeReRegistrationTimeout int                      `json:"nodeReRegistrationTimeout"`
	ProtocolMappers           []KeycloakProtocolMapper `json:"protocolMappers,omitempty"`
	UseTemplateConfig         bool                     `json:"useTemplateConfig"`
	UseTemplateScope          bool                     `json:"useTemplateScope"`
	UseTemplateMappers        bool                     `json:"useTemplateMappers"`
	Access                    map[string]bool          `json:"access"`
}