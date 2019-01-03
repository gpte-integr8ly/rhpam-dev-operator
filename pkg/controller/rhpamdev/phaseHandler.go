package rhpamdev

import (
	"context"
	"fmt"
	"math/rand"

	gptev1alpha1 "github.com/gpte-naps/rhpam-dev-operator/pkg/apis/gpte/v1alpha1"
	appsv1 "github.com/openshift/api/apps/v1"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

type phaseHandler struct {
	client client.Client
	scheme *runtime.Scheme
}

func NewPhaseHandler(c client.Client, s *runtime.Scheme) *phaseHandler {
	return &phaseHandler{
		client: c,
		scheme: s,
	}
}

func (ph *phaseHandler) Initialize(rhpam *gptev1alpha1.RhpamDev) (*gptev1alpha1.RhpamDev, error) {
	log.Info("Phase Initialize")
	// fill in any defaults that are not set
	rhpam.Defaults()

	// validate
	if err := rhpam.Validate(); err != nil {
		log.Error(err, "Validation failed", "phase", "Initialize")
		return nil, err
	}

	rhpam.Status.Phase = gptev1alpha1.PhaseInitialized
	rhpam.Status.Version = RhpamVersion
	return rhpam, nil
}

func (ph *phaseHandler) Prepare(rhpam *gptev1alpha1.RhpamDev) (*gptev1alpha1.RhpamDev, error) {
	log.Info("Phase Prepare")

	if err := ph.createResources(rhpam, []Resource{ServiceAccountResource}); err != nil {
		return nil, err
	}

	rhpam.Status.Phase = gptev1alpha1.PhasePrepared
	rhpam.Status.Version = RhpamVersion
	return rhpam, nil
}

func (ph *phaseHandler) InstallDatabase(rhpam *gptev1alpha1.RhpamDev) (*gptev1alpha1.RhpamDev, error) {
	log.Info("Phase InstallDatabase")

	//pg credentials secret
	dbUser := "user" + generateToken(4)
	dbPassword := generateToken(10)
	err := ph.createSecret(rhpam, DatabaseCredentialsSecret, map[string][]byte{"database-user": []byte(dbUser), "database-password": []byte(dbPassword), "database-name": []byte(DatabaseName)})
	if err != nil {
		log.Error(err, "Create secret failed.", "secret", DatabaseCredentialsSecret)
		return nil, err
	}

	//pg init configmap
	configmapHelper := newConfigmapHelper()
	configmapData, err := configmapHelper.data(initConfigmapResources())
	if err != nil {
		log.Error(err, "Create configmap failed.", "configmap", DatabaseInitConfigmap)
		return nil, err
	}
	err1 := ph.createConfigmap(rhpam, DatabaseInitConfigmap, configmapData)
	if err1 != nil {
		log.Error(err, "Create configmap failed.", "configmap", DatabaseInitConfigmap)
		return nil, err1
	}

	//pg database
	if err := ph.createResources(rhpam, []Resource{DatabasePvcResource, DatabaseServiceResource, DatabaseDeploymentConfigResource}); err != nil {
		return nil, err
	}

	rhpam.Status.Phase = gptev1alpha1.PhaseDatabaseInstalled
	rhpam.Status.Version = RhpamVersion
	return rhpam, nil
}

func (ph *phaseHandler) installBusinessCentral(rhpam *gptev1alpha1.RhpamDev) (*gptev1alpha1.RhpamDev, error) {
	log.Info("Phase InstallBusinessCentral")

	if err := ph.createResources(rhpam, []Resource{BusinessCentralPvcResource, BusinessCentralServiceResource,
		BusinessCentralRouteResource, BusinessCentralDeploymentResource}); err != nil {
		return nil, err
	}

	rhpam.Status.Phase = gptev1alpha1.PhaseBusinessCentralInstalled
	rhpam.Status.Version = RhpamVersion
	return rhpam, nil
}

func (ph *phaseHandler) WaitForDatabase(rhpam *gptev1alpha1.RhpamDev) (bool, *gptev1alpha1.RhpamDev, error) {

	ready, err := ph.isDatabaseReady(rhpam)
	if err != nil {
		return false, nil, fmt.Errorf("Error checking for database : %s", err)
	}

	if ready {
		rhpam.Status.Phase = gptev1alpha1.PhaseDatabaseReady
		rhpam.Status.Version = RhpamVersion
	}

	return ready, rhpam, nil
}

func (ph *phaseHandler) installKieServer(rhpam *gptev1alpha1.RhpamDev) (*gptev1alpha1.RhpamDev, error) {
	log.Info("Phase InstallKieServer")

	if err := ph.createResources(rhpam, []Resource{KieServerServiceResource, KieServerRouteResource, KieServerDeploymentResource}); err != nil {
		return nil, err
	}

	rhpam.Status.Phase = gptev1alpha1.PhaseComplete
	rhpam.Status.Version = RhpamVersion
	return rhpam, nil
}

func (ph *phaseHandler) createResources(cr *gptev1alpha1.RhpamDev, resources []Resource) error {
	for _, resource := range resources {
		err := ph.createResource(cr, resource)
		if err != nil {
			log.Error(err, "Create Resource Failed", "resource", resource.name)
			return err
		}
	}
	return nil
}

func (ph *phaseHandler) createResource(cr *gptev1alpha1.RhpamDev, res Resource) error {
	resourceHelper := newResourceHelper(cr)
	resource, err := resourceHelper.createResource(res)

	if err != nil {
		return fmt.Errorf("Error parsing templates: %s", err)
	}

	// Try to find the resource, it may already exist
	selector := types.NamespacedName{
		Namespace: cr.Namespace,
		Name:      res.name,
	}
	err = ph.client.Get(context.TODO(), selector, resource)

	// The resource exists, do nothing
	if err == nil {
		return nil
	}

	//Resource does not exist or something went wrong
	if !errors.IsNotFound(err) {
		return fmt.Errorf("Error reading resource '%s': %s", res.name, err)
	}

	// Set the CR as the owner of this resource so that when
	// the CR is deleted this resource also gets removed
	err = controllerutil.SetControllerReference(cr, resource.(metav1.Object), ph.scheme)
	if err != nil {
		return fmt.Errorf("Error setting the custom resource as owner: %s", err)
	}

	err = ph.client.Create(context.TODO(), resource)
	if err != nil {
		return fmt.Errorf("Error creating resource: %s", err)
	}
	return nil
}

func (ph *phaseHandler) createSecret(cr *gptev1alpha1.RhpamDev, name string, data map[string][]byte) error {

	secret := &v1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels:    map[string]string{"app": ApplicationName},
			Name:      name,
			Namespace: cr.Namespace,
		},
		Data: data,
		Type: "Opaque",
	}

	// Try to find the resource, it may already exist
	selector := types.NamespacedName{
		Namespace: cr.Namespace,
		Name:      name,
	}
	if err := ph.client.Get(context.TODO(), selector, secret); err == nil {
		return nil
	}

	controllerutil.SetControllerReference(cr, secret, ph.scheme)
	return ph.client.Create(context.TODO(), secret)
}

func (ph *phaseHandler) createConfigmap(cr *gptev1alpha1.RhpamDev, name string, data map[string]string) error {

	configmap := &v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels:    map[string]string{"app": ApplicationName},
			Name:      name,
			Namespace: cr.Namespace,
		},
		Data: data,
	}

	// Try to find the resource, it may already exist
	selector := types.NamespacedName{
		Namespace: cr.Namespace,
		Name:      name,
	}
	if err := ph.client.Get(context.TODO(), selector, configmap); err == nil {
		return nil
	}

	controllerutil.SetControllerReference(cr, configmap, ph.scheme)
	return ph.client.Create(context.TODO(), configmap)
}

func (ph *phaseHandler) isDatabaseReady(cr *gptev1alpha1.RhpamDev) (bool, error) {
	resource := appsv1.DeploymentConfig{}

	selector := types.NamespacedName{
		Namespace: cr.Namespace,
		Name:      DatabaseDeployment,
	}

	if err := ph.client.Get(context.TODO(), selector, &resource); err != nil {
		return false, err
	}

	return resource.Status.ReadyReplicas == 1, nil
}

func generateToken(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
