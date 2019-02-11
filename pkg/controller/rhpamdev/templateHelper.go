package rhpamdev

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"text/template"

	rhpamv1alpha1 "github.com/gpte-integr8ly/rhpam-dev-operator/pkg/apis/rhpam/v1alpha1"
)

type TemplateHelper struct {
	parameters   RhpamdevParameters
	templatePath string
}

type RhpamdevParameters struct {
	ServiceAccount                      string
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
	BusinessCentralCpuRequest           string
	BusinessCentralCpuLimit             string
	BusinessCentralMemoryRequest        string
	BusinessCentralMemoryLimit          string
	BusinessCentralGcMaxMetaSize        string
	BusinessCentralKieMBeans            string
	BusinessCentralJavaOptsAppend       string
	KieServerRoute                      string
	KieServerService                    string
	KieServerDeployment                 string
	KieServerImageStreamNamespace       string
	KieServerImage                      string
	KieServerImageTag                   string
	KieServerCpuRequest                 string
	KieServerCpuLimit                   string
	KieServerMemoryRequest              string
	KieServerMemoryLimit                string
	KieServerGcMaxMetaSize              string
	KieServerDroolsFilterClasses        string
	KieServerBypassAuthUser             string
	KieServerControllerProtocol         string
	KieServerId                         string
	KieMavenUser                        string
	KieMavenPassword                    string
	KieServerKieMBeans                  string
}

func newTemplateHelper(rhpam *rhpamv1alpha1.RhpamDev) *TemplateHelper {

	params := RhpamdevParameters{
		ServiceAccount:                      ServiceAccount,
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
		BusinessCentralCpuRequest:           valueOrDefault(rhpam.Spec.Config.BusinessCentralConfig.CpuRequest, BusinessCentralCpuRequest),
		BusinessCentralCpuLimit:             valueOrDefault(rhpam.Spec.Config.BusinessCentralConfig.CpuLimit, BusinessCentralCpuLimit),
		BusinessCentralMemoryRequest:        valueOrDefault(rhpam.Spec.Config.BusinessCentralConfig.MemoryRequest, BusinessCentralMemoryRequest),
		BusinessCentralMemoryLimit:          valueOrDefault(rhpam.Spec.Config.BusinessCentralConfig.MemoryLimit, BusinessCentralMemoryLimit),
		BusinessCentralGcMaxMetaSize:        valueOrDefault(rhpam.Spec.Config.BusinessCentralConfig.GcMaxMetaSize, BusinessCentralGcMaxMetaSize),
		BusinessCentralJavaOptsAppend:       valueOrDefault(rhpam.Spec.Config.BusinessCentralConfig.JavaOptsAppend, BusinessCentralJavaOptsAppend),
		BusinessCentralKieMBeans:            valueOrDefault(rhpam.Spec.Config.BusinessCentralConfig.KieMBeans, BusinessCentralKieMBeans),
		KieServerRoute:                      KieServerRoute,
		KieServerService:                    KieServerService,
		KieServerDeployment:                 KieServerDeployment,
		KieServerImageStreamNamespace:       KieServerImageStreamNamespace,
		KieServerImage:                      KieServerImage,
		KieServerImageTag:                   KieServerImageTag,
		KieServerCpuRequest:                 valueOrDefault(rhpam.Spec.Config.KieServerConfig.CpuRequest, KieServerCpuRequest),
		KieServerCpuLimit:                   valueOrDefault(rhpam.Spec.Config.KieServerConfig.CpuLimit, KieServerCpuLimit),
		KieServerMemoryRequest:              valueOrDefault(rhpam.Spec.Config.KieServerConfig.MemoryRequest, KieServerMemoryRequest),
		KieServerMemoryLimit:                valueOrDefault(rhpam.Spec.Config.KieServerConfig.MemoryLimit, KieServerMemoryLimit),
		KieServerGcMaxMetaSize:              valueOrDefault(rhpam.Spec.Config.KieServerConfig.GcMaxMetaSize, KieServerGcMaxMetaSize),
		KieServerDroolsFilterClasses:        valueOrDefault(rhpam.Spec.Config.KieServerConfig.DroolsFilterClasses, KieServerDroolsFilterClasses),
		KieServerBypassAuthUser:             valueOrDefault(rhpam.Spec.Config.KieServerConfig.BypassAuthUser, KieServerBypassAuthUser),
		KieServerControllerProtocol:         KieServerControllerProtocol,
		KieServerId:                         KieServerId,
		KieMavenUser:                        KieMavenUser,
		KieMavenPassword:                    KieMavenPassword,
		KieServerKieMBeans:                  valueOrDefault(rhpam.Spec.Config.KieServerConfig.KieMBeans, KieServerKieMBeans),
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
