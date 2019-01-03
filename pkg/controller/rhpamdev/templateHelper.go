package rhpamdev

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"text/template"

	gptev1alpha1 "github.com/gpte-naps/rhpam-dev-operator/pkg/apis/gpte/v1alpha1"
)

type TemplateHelper struct {
	parameters   RhpamdevParameters
	templatePath string
}

type RhpamdevParameters struct {
	ServiceAccount                      string
	ServiceAccountRoleBinding           string
	ApplicationName                     string
	ApplicationNamespace                string
	PostgresqlPvc                       string
	PostgresqlPvcCapacity               string
	PostgresqlService                   string
	PostgresqlDeployment                string
	PostgresqlImage                     string
	PostgresqlSecret                    string
	PostgresqlMaxConnections            string
	PostgresqlSharedBuffers             string
	PostgresqlMaxPreparedTransactions   string
	PostgresqlMemoryLimit               string
	PostgresqlInitConfigmap             string
	BusinessCentralService              string
	BusinessCentralPvc                  string
	BusinessCentralRoute                string
	BusinessCentralDeployment           string
	BusinessCentralVolumeCapacity       string
	BusinessCentralImageStreamNamespace string
	BusinessCentralImage                string
	BusinessCentralImageTag             string
	BusinessCentralMemoryLimit          string
	BusinessCentralJavaMaxMemRatio      string
	BusinessCentralJavaInitialMemRatio  string
	BusinessCentralGcMaxMetaSize        string
	KieMbeans                           string
	BusinessCentralJavaOptsAppend       string
	KieServerRoute                      string
	KieServerService                    string
	KieServerDeployment                 string
	KieServerImageStreamNamespace       string
	KieServerImage                      string
	KieServerImageTag                   string
	KieServerMemoryLimit                string
	KieServerJavaMaxMemRatio            string
	KieServerJavaInitialMemRatio        string
	KieServerDroolsFilterClasses        string
	KieServerBypassAuthUser             string
	KieServerControllerProtocol         string
	KieServerId                         string
	KieMavenUser                        string
	KieMavenPassword                    string
}

func newTemplateHelper(rhpam *gptev1alpha1.RhpamDev) *TemplateHelper {

	params := RhpamdevParameters{
		ServiceAccount:                      ServiceAccount,
		ServiceAccountRoleBinding:           ServiceAccountRoleBinding,
		ApplicationName:                     ApplicationName,
		ApplicationNamespace:                rhpam.Namespace,
		PostgresqlPvc:                       DatabasePvc,
		PostgresqlPvcCapacity:               valueOrDefault(rhpam.Spec.Config.DatabaseConfig.PersistentVolumeCapacity, DatabaseVolumeCapacity),
		PostgresqlDeployment:                DatabaseDeployment,
		PostgresqlService:                   DatabaseService,
		PostgresqlImage:                     DatabaseImage,
		PostgresqlSecret:                    DatabaseCredentialsSecret,
		PostgresqlMaxConnections:            valueOrDefault(rhpam.Spec.Config.DatabaseConfig.MaxConnections, DatabaseMaxConnections),
		PostgresqlSharedBuffers:             valueOrDefault(rhpam.Spec.Config.DatabaseConfig.SharedBuffers, DatabaseSharedBuffers),
		PostgresqlMaxPreparedTransactions:   valueOrDefault(rhpam.Spec.Config.DatabaseConfig.MaxPreparedTransactions, DatabaseMaxPreparedTransactions),
		PostgresqlMemoryLimit:               valueOrDefault(rhpam.Spec.Config.DatabaseConfig.MemoryLimit, DatabaseMemoryLimit),
		PostgresqlInitConfigmap:             DatabaseInitConfigmap,
		BusinessCentralService:              BusinessCentralService,
		BusinessCentralPvc:                  BusinessCentralPvc,
		BusinessCentralRoute:                BusinessCentralRoute,
		BusinessCentralDeployment:           BusinessCentralDeployment,
		BusinessCentralVolumeCapacity:       valueOrDefault(rhpam.Spec.Config.BusinessCentralConfig.PersistentVolumeCapacity, BusinessCentralVolumeCapacity),
		BusinessCentralImageStreamNamespace: BusinessCentralImageStreamNamespace,
		BusinessCentralImage:                BusinessCentralImage,
		BusinessCentralImageTag:             BusinessCentralImageTag,
		BusinessCentralMemoryLimit:          valueOrDefault(rhpam.Spec.Config.BusinessCentralConfig.MemoryLimit, BusinessCentralMemoryLimit),
		BusinessCentralJavaMaxMemRatio:      valueOrDefault(rhpam.Spec.Config.BusinessCentralConfig.JavaMaxMemRatio, BusinessCentralJavaMaxMemRatio),
		BusinessCentralJavaInitialMemRatio:  valueOrDefault(rhpam.Spec.Config.BusinessCentralConfig.JavaInitialMemRatio, BusinessCentralJavaInitialMemRatio),
		BusinessCentralGcMaxMetaSize:        valueOrDefault(rhpam.Spec.Config.BusinessCentralConfig.GcMaxSize, BusinessCentralGcMaxMetaSize),
		KieMbeans:                           valueOrDefault(rhpam.Spec.Config.BusinessCentralConfig.KieMbeans, KieMBeans),
		BusinessCentralJavaOptsAppend:       valueOrDefault(rhpam.Spec.Config.BusinessCentralConfig.JavaOptsAppend, BusinessCentralJavaOptsAppend),
		KieServerRoute:                      KieServerRoute,
		KieServerService:                    KieServerService,
		KieServerDeployment:                 KieServerDeployment,
		KieServerImageStreamNamespace:       KieServerImageStreamNamespace,
		KieServerImage:                      KieServerImage,
		KieServerImageTag:                   KieServerImageTag,
		KieServerMemoryLimit:                valueOrDefault(rhpam.Spec.Config.KieServerConfig.MemoryLimit, KieServerMemoryLimit),
		KieServerJavaMaxMemRatio:            valueOrDefault(rhpam.Spec.Config.KieServerConfig.JavaMaxMemRatio, KieServerJavaMaxMemRatio),
		KieServerJavaInitialMemRatio:        valueOrDefault(rhpam.Spec.Config.KieServerConfig.JavaInitialMemRatio, KieServerJavaInitialMemRatio),
		KieServerDroolsFilterClasses:        valueOrDefault(rhpam.Spec.Config.KieServerConfig.DroolsFilterClasses, KieServerDroolsFilterClasses),
		KieServerBypassAuthUser:             valueOrDefault(rhpam.Spec.Config.KieServerConfig.BypassAuthUser, KieServerBypassAuthUser),
		KieServerControllerProtocol:         KieServerControllerProtocol,
		KieServerId:                         KieServerId,
		KieMavenUser:                        KieMavenUser,
		KieMavenPassword:                    KieMavenPassword,
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
