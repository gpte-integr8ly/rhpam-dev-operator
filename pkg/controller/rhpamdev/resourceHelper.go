package rhpamdev

import (
	yaml "github.com/ghodss/yaml"
	gptev1alpha1 "github.com/gpte-naps/rhpam-dev-operator/pkg/apis/gpte/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	ServiceAccountTemplate            = "rhpamdev-service-account"
	ServiceAccountRoleBindingTemplate = "rhpamdev-sa-role-binding"
	DatabasePvcTemplate               = "rhpamdev-postgresql-pvc"
	DatabaseServiceTemplate           = "rhpamdev-postgresql-service"
	DatabaseDeploymentTemplate        = "rhpamdev-postgresql-dc"
	BusinessCentralRouteTemplate      = "rhpamdev-bc-route"
	BusinessCentralServiceTemplate    = "rhpamdev-bc-service"
	BusinessCentralPvcTemplate        = "rhpam-bc-pvc"
	BusinessCentralDeploymentTemplate = "rhpam-bc-dc"
)

type Resource struct {
	name     string
	template string
}

var (
	ServiceAccountResource            Resource = Resource{name: ServiceAccount, template: ServiceAccountTemplate}
	ServiceAccountRoleBindingResource Resource = Resource{name: ServiceAccountRoleBinding, template: ServiceAccountRoleBindingTemplate}
	DatabasePvcResource               Resource = Resource{name: DatabasePvc, template: DatabasePvcTemplate}
	DatabaseServiceResource           Resource = Resource{name: DatabaseService, template: DatabaseServiceTemplate}
	DatabaseDeploymentConfigResource  Resource = Resource{name: DatabaseDeployment, template: DatabaseDeploymentTemplate}
	BusinessCentralDeploymentResource Resource = Resource{name: BusinessCentralDeployment, template: BusinessCentralDeploymentTemplate}
	BusinessCentralServiceResource    Resource = Resource{name: BusinessCentralService, template: BusinessCentralServiceTemplate}
	BusinessCentralRouteResource      Resource = Resource{name: BusinessCentralRoute, template: BusinessCentralRouteTemplate}
	BusinessCentralPvcResource        Resource = Resource{name: BusinessCentralPvc, template: BusinessCentralPvcTemplate}
)

type ResourceHelper struct {
	cr             *gptev1alpha1.RhpamDev
	templateHelper *TemplateHelper
}

func newResourceHelper(cr *gptev1alpha1.RhpamDev) *ResourceHelper {
	return &ResourceHelper{
		cr:             cr,
		templateHelper: newTemplateHelper(cr),
	}
}

func (r *ResourceHelper) createResource(res Resource) (runtime.Object, error) {

	tpl, err := r.templateHelper.loadTemplate(res.template)

	if err != nil {
		return nil, err
	}

	resource := unstructured.Unstructured{}
	err = yaml.Unmarshal(tpl, &resource)

	if err != nil {
		return nil, err
	}

	return &resource, nil
}
