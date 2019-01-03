package rhpamdev

import (
	yaml "github.com/ghodss/yaml"
	gptev1alpha1 "github.com/gpte-naps/rhpam-dev-operator/pkg/apis/gpte/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
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

func (r *ResourceHelper) createResource(template string) (runtime.Object, error) {
	tpl, err := r.templateHelper.loadTemplate(template)

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
