package rhpamdev

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"text/template"
)

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

	realmPath := os.Getenv("TEMPLATE_PATH")
	if realmPath == "" {
		realmPath = "./templates/rhsso"
	}

	return &RealmHelper{
		realmTemplatePath: realmPath,
	}
}

func (r *RealmHelper) loadRealmTemplate(params RealmParameters) ([]byte, error) {
	return loadTemplate("realm", r.realmTemplatePath, params)
}

func (r *RealmHelper) loadRealmClientTemplate(params RealmClientParameters) ([]byte, error) {
	return loadTemplate("realm-client", r.realmTemplatePath, params)
}

func loadTemplate(name string, templatePath string, data interface{}) ([]byte, error) {
	path := fmt.Sprintf("%s/%s.json", templatePath, name)
	tpl, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	parsed, err := template.New("realm").Parse(string(tpl))
	if err != nil {
		return nil, err
	}
	var buffer bytes.Buffer
	err = parsed.Execute(&buffer, data)
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}
