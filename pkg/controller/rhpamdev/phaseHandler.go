package rhpamdev

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	rhpamv1alpha1 "github.com/gpte-integr8ly/rhpam-dev-operator/pkg/apis/rhpam/v1alpha1"
	common "github.com/gpte-integr8ly/rhpam-dev-operator/pkg/controller/common"
	keycloak "github.com/gpte-integr8ly/rhpam-dev-operator/pkg/keycloak"
	appsv1 "github.com/openshift/api/apps/v1"
	"github.com/pkg/errors"
	"k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

type phaseHandler struct {
	client          client.Client
	scheme          *runtime.Scheme
	keycloakFactory *keycloak.KeycloakFactory
}

func NewPhaseHandler(c client.Client, s *runtime.Scheme) *phaseHandler {
	return &phaseHandler{
		client:          c,
		scheme:          s,
		keycloakFactory: keycloak.NewKeycloakFactory(),
	}
}

func (ph *phaseHandler) Initialize(rhpam *rhpamv1alpha1.RhpamDev) (*rhpamv1alpha1.RhpamDev, error) {
	log.Info("Phase Initialize")
	// fill in any defaults that are not set
	rhpam.Defaults()

	// validate
	if err := rhpam.Validate(); err != nil {
		log.Error(err, "Validation failed", "phase", "Initialize")
		return nil, err
	}

	//get sso username, password, admin url
	if err := ph.readSSOSecret(); err != nil {
		log.Error(err, "Error reading RHSSO secret")
		return nil, err
	}

	rhpam.Status.Phase = rhpamv1alpha1.PhaseAccepted
	rhpam.Status.Version = RhpamVersion
	return rhpam, nil
}

func (ph *phaseHandler) ProvisionRealm(rhpam *rhpamv1alpha1.RhpamDev) (*rhpamv1alpha1.RhpamDev, error) {
	log.Info("Phase Provision Realm")

	//TODO: check if realm exits -> check for secret

	//create realm in keycloak
	ssoClient, err := ph.keycloakFactory.AuthenticatedClient()
	if err != nil {
		return nil, err
	}
	realmHelper := keycloak.NewRealmHelper()

	realmId := rhpam.Name + "-" + generateToken(6)
	log.Info("Creating Realm", "Realm Id", realmId)
	realmParams := keycloak.RealmParameters{RealmId: realmId}
	json, err := realmHelper.LoadRealmTemplate(realmParams)
	if err != nil {
		return nil, err
	}

	if err := ssoClient.CreateRealm(json); err != nil {
		return nil, errors.Wrap(err, "Error creating realm in rhsso")
	}

	//create realm clients in keycloak
	//business-central
	bcClient := "rhpambc"
	bcRealmClientParams := keycloak.RealmClientParameters{ClientId: bcClient}
	bcRealmClientParams.RootUrl = "https://" + BusinessCentralDeployment + "-" + rhpam.Namespace + "." + rhpam.Spec.Domain
	bcRealmClientParams.AdminUrl = bcRealmClientParams.RootUrl
	bcRealmClientParams.RedirectUris = "\"" + bcRealmClientParams.RootUrl + "/*\""
	bcRealmClientParams.WebOrigin = "\"" + bcRealmClientParams.RootUrl + "\""
	bcRealmClientParams.BearerOnly = "false"
	bcRealmClientParams.ImplicitFlowEnabled = "false"
	bcRealmClientParams.DirectAcessGrantEnabled = "true"
	bcRealmClientParams.PublicClient = "false"
	json1, err := realmHelper.LoadRealmClientTemplate(bcRealmClientParams)
	if err != nil {
		return nil, err
	}
	if err := ssoClient.CreateClient(json1, realmId); err != nil {
		return nil, errors.Wrap(err, "Error creating realm in rhsso")
	}

	//kie-server
	ksClient := "rhpamks"
	ksRealmClientParams := keycloak.RealmClientParameters{ClientId: ksClient}
	ksRealmClientParams.RootUrl = "https://" + KieServerDeployment + "-" + rhpam.Namespace + "." + rhpam.Spec.Domain
	ksRealmClientParams.AdminUrl = ksRealmClientParams.RootUrl
	ksRealmClientParams.RedirectUris = "\"" + ksRealmClientParams.RootUrl + "/*\""
	ksRealmClientParams.WebOrigin = "\"" + ksRealmClientParams.RootUrl + "\""
	ksRealmClientParams.BearerOnly = "true"
	ksRealmClientParams.ImplicitFlowEnabled = "false"
	ksRealmClientParams.DirectAcessGrantEnabled = "true"
	ksRealmClientParams.PublicClient = "false"
	json2, err := realmHelper.LoadRealmClientTemplate(ksRealmClientParams)
	if err != nil {
		return nil, err
	}
	if err := ssoClient.CreateClient(json2, realmId); err != nil {
		return nil, errors.Wrap(err, "Error creating realm in rhsso")
	}

	//kie-server direct
	ksdRealmClientParams := keycloak.RealmClientParameters{ClientId: "rhpamks-direct"}
	ksdRealmClientParams.RootUrl = "https://" + KieServerDeployment + "-" + rhpam.Namespace + "." + rhpam.Spec.Domain
	ksdRealmClientParams.AdminUrl = ksdRealmClientParams.RootUrl
	ksdRealmClientParams.RedirectUris = "\"" + ksdRealmClientParams.RootUrl + "/*\""
	ksdRealmClientParams.WebOrigin = "\"" + ksdRealmClientParams.RootUrl + "\""
	ksdRealmClientParams.BearerOnly = "false"
	ksdRealmClientParams.ImplicitFlowEnabled = "false"
	ksdRealmClientParams.DirectAcessGrantEnabled = "true"
	ksdRealmClientParams.PublicClient = "true"
	json3, err := realmHelper.LoadRealmClientTemplate(ksdRealmClientParams)
	if err != nil {
		return nil, err
	}
	if err := ssoClient.CreateClient(json3, realmId); err != nil {
		return nil, errors.Wrap(err, "Error creating realm in rhsso")
	}

	//create secret with realm details
	clientSecret, err := ph.getClientSecret(ssoClient, realmId, bcClient)
	if err != nil {
		return nil, err
	}
	err = ph.createRealmSecret(rhpam, BusinessCentralRealmSecret, ph.keycloakFactory.AdminUrl+"/auth", realmId, bcClient, clientSecret)
	if err != nil {
		return nil, err
	}

	clientSecret, err = ph.getClientSecret(ssoClient, realmId, ksClient)
	if err != nil {
		return nil, err
	}
	err = ph.createRealmSecret(rhpam, KieServerRealmSecret, ph.keycloakFactory.AdminUrl+"/auth", realmId, ksClient, clientSecret)
	if err != nil {
		return nil, err
	}

	rhpam.Status.Phase = rhpamv1alpha1.PhaseRealmProvisioned
	rhpam.Status.Realm = realmId
	rhpam.Status.Version = RhpamVersion
	return rhpam, nil

}

func (ph *phaseHandler) Prepare(rhpam *rhpamv1alpha1.RhpamDev) (*rhpamv1alpha1.RhpamDev, error) {
	log.Info("Phase Prepare")

	if err := ph.createResources(rhpam, []Resource{ServiceAccountResource}); err != nil {
		return nil, err
	}

	rhpam.Status.Phase = rhpamv1alpha1.PhasePrepared
	rhpam.Status.Version = RhpamVersion
	return rhpam, nil
}

func (ph *phaseHandler) InstallDatabase(rhpam *rhpamv1alpha1.RhpamDev) (*rhpamv1alpha1.RhpamDev, error) {
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

	rhpam.Status.Phase = rhpamv1alpha1.PhaseDatabaseInstalled
	rhpam.Status.Version = RhpamVersion
	return rhpam, nil
}

func (ph *phaseHandler) InstallBusinessCentral(rhpam *rhpamv1alpha1.RhpamDev) (*rhpamv1alpha1.RhpamDev, error) {
	log.Info("Phase InstallBusinessCentral")

	if err := ph.createResources(rhpam, []Resource{BusinessCentralPvcResource, BusinessCentralServiceResource,
		BusinessCentralRouteResource, BusinessCentralDeploymentResource}); err != nil {
		return nil, err
	}

	rhpam.Status.Phase = rhpamv1alpha1.PhaseBusinessCentralInstalled
	rhpam.Status.Version = RhpamVersion
	return rhpam, nil
}

func (ph *phaseHandler) WaitForDatabase(rhpam *rhpamv1alpha1.RhpamDev) (bool, *rhpamv1alpha1.RhpamDev, error) {

	ready, err := ph.isDatabaseReady(rhpam)
	if err != nil {
		return false, nil, fmt.Errorf("Error checking for database : %s", err)
	}

	if ready {
		rhpam.Status.Phase = rhpamv1alpha1.PhaseDatabaseReady
		rhpam.Status.Version = RhpamVersion
	}

	return ready, rhpam, nil
}

func (ph *phaseHandler) InstallKieServer(rhpam *rhpamv1alpha1.RhpamDev) (*rhpamv1alpha1.RhpamDev, error) {
	log.Info("Phase InstallKieServer")

	if err := ph.createResources(rhpam, []Resource{KieServerServiceResource, KieServerRouteResource, KieServerDeploymentResource}); err != nil {
		return nil, err
	}

	rhpam.Status.Phase = rhpamv1alpha1.PhaseComplete
	rhpam.Status.Version = RhpamVersion
	return rhpam, nil
}

func (ph *phaseHandler) readSSOSecret() error {

	secret, err := common.ReadSSOSecret(ph.client)
	if err != nil {
		return err
	}

	ph.keycloakFactory.AdminUser = string(secret.Data["SSO_ADMIN_USERNAME"])
	ph.keycloakFactory.AdminPassword = string(secret.Data["SSO_ADMIN_PASSWORD"])
	ph.keycloakFactory.AdminUrl = string(secret.Data["SSO_ADMIN_URL"])

	return nil
}

func (ph *phaseHandler) getClientSecret(ssoClient keycloak.KeycloakInterface, realmId string, clientId string) (string, error) {
	clients, err := ssoClient.ListClients(realmId)
	if err != nil {
		return "", err
	}
	id := ""
	for _, c := range clients {
		if c.ClientID == clientId {
			id = c.ID
			break
		}
	}
	clientSecret, err := ssoClient.GetClientSecret(id, realmId)
	if err != nil {
		return "", err
	}
	return clientSecret, nil
}

func (ph *phaseHandler) createRealmSecret(rhpam *rhpamv1alpha1.RhpamDev, secret string, url string, realmId string, client string, clientSecret string) error {
	err := ph.createSecret(rhpam, secret, map[string][]byte{"sso-url": []byte(url), "realm": []byte(realmId), "client": []byte(client), "client-secret": []byte(clientSecret)})
	if err != nil {
		log.Error(err, "Create secret failed.", "secret", secret)
		return err
	}
	return nil
}

func (ph *phaseHandler) createResources(cr *rhpamv1alpha1.RhpamDev, resources []Resource) error {
	for _, resource := range resources {
		err := ph.createResource(cr, resource)
		if err != nil {
			log.Error(err, "Create Resource Failed", "resource", resource.name)
			return err
		}
	}
	return nil
}

func (ph *phaseHandler) createResource(cr *rhpamv1alpha1.RhpamDev, res Resource) error {
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
	if !k8serrors.IsNotFound(err) {
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

func (ph *phaseHandler) createSecret(cr *rhpamv1alpha1.RhpamDev, name string, data map[string][]byte) error {

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

func (ph *phaseHandler) createConfigmap(cr *rhpamv1alpha1.RhpamDev, name string, data map[string]string) error {

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

func (ph *phaseHandler) isDatabaseReady(cr *rhpamv1alpha1.RhpamDev) (bool, error) {
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
	rand.Seed(time.Now().Unix())
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
