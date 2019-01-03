package rhpamdev

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"text/template"

	gptev1alpha1 "github.com/gpte-naps/rhpam-dev-operator/pkg/apis/gpte/v1alpha1"
)

const (
	ServiceAccountResource            = "rhpamdev-service-account"
	ServiceAccountRoleBindingResource = "rhpamdev-sa-role-binding"
	DatabasePvcResource               = "rhpamdev-postgresql-pvc"
	DatabaseServiceResource           = "rhpamdev-postgresql-service"
	DatabaseDeploymentResource        = "rhpamdev-postgresql-dc"
)

type TemplateHelper struct {
	parameters   RhpamdevParameters
	templatePath string
}

type RhpamdevParameters struct {
	ServiceAccount                    string
	ServiceAccountRoleBinding         string
	ApplicationName                   string
	ApplicationNamespace              string
	PostgresqlPvc                     string
	PostgresqlPvcCapacity             string
	PostgresqlService                 string
	PostgresqlDeployment              string
	PostgresqlImage                   string
	PostgresqlSecret                  string
	PostgresqlMaxConnections          string
	PostgresqlSharedBuffers           string
	PostgresqlMaxPreparedTransactions string
	PostgresqlMemoryLimit             string
	PostgresqlInitConfigmap           string
}

func newTemplateHelper(rhpam *gptev1alpha1.RhpamDev) *TemplateHelper {

	params := RhpamdevParameters{
		ServiceAccount:                    ServiceAccount,
		ServiceAccountRoleBinding:         ServiceAccountRoleBinding,
		ApplicationName:                   ApplicationName,
		ApplicationNamespace:              rhpam.Namespace,
		PostgresqlPvc:                     DatabasePvc,
		PostgresqlPvcCapacity:             valueOrDefault(rhpam.Spec.Config.DatabaseConfig.PersistentVolumeCapacity, DatabaseVolumeCapacity),
		PostgresqlDeployment:              DatabaseDeployment,
		PostgresqlService:                 DatabaseService,
		PostgresqlImage:                   DatabaseImage,
		PostgresqlSecret:                  DatabaseCredentialsSecret,
		PostgresqlMaxConnections:          valueOrDefault(rhpam.Spec.Config.DatabaseConfig.MaxConnections, DatabaseMaxConnections),
		PostgresqlSharedBuffers:           valueOrDefault(rhpam.Spec.Config.DatabaseConfig.SharedBuffers, DatabaseSharedBuffers),
		PostgresqlMaxPreparedTransactions: valueOrDefault(rhpam.Spec.Config.DatabaseConfig.MaxPreparedTransactions, DatabaseMaxPreparedTransactions),
		PostgresqlMemoryLimit:             valueOrDefault(rhpam.Spec.Config.DatabaseConfig.MemoryLimit, DatabaseMemoryLimit),
		PostgresqlInitConfigmap:           DatabaseInitConfigmap,
	}

	templatePath := os.Getenv("TEMPLATE_PATH")
	if templatePath == "" {
		templatePath = "./templates"
	}

	return &TemplateHelper{
		parameters:   params,
		templatePath: templatePath,
	}
}

func (h *TemplateHelper) loadTemplate(name string) ([]byte, error) {
	path := fmt.Sprintf("%s/%s.yaml", h.templatePath, name)
	tpl, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	parsed, err := template.New("rhpamdev").Parse(string(tpl))
	if err != nil {
		return nil, err
	}

	var buffer bytes.Buffer
	err = parsed.Execute(&buffer, h.parameters)
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func valueOrDefault(value string, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}
