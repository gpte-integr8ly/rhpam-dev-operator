package rhpamuser

import (
	"context"

	rhpamv1alpha1 "github.com/gpte-integr8ly/rhpam-dev-operator/pkg/apis/rhpam/v1alpha1"
	"github.com/gpte-integr8ly/rhpam-dev-operator/pkg/controller/common"
	"github.com/gpte-integr8ly/rhpam-dev-operator/pkg/keycloak"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type phaseHandler struct {
	client          client.Client
	keycloakFactory *keycloak.KeycloakFactory
}

func NewPhaseHandler(c client.Client) *phaseHandler {
	return &phaseHandler{
		client:          c,
		keycloakFactory: keycloak.NewKeycloakFactory(),
	}
}

func (ph *phaseHandler) Initialize(rhpamuser *rhpamv1alpha1.RhpamUser) (*rhpamv1alpha1.RhpamUser, error) {
	log.Info("Phase Initialize")

	//get sso username, password, admin url
	if err := ph.readSSOSecret(); err != nil {
		log.Error(err, "Error reading RHSSO secret")
		return nil, err
	}

	rhpamuser.Status.Phase = rhpamv1alpha1.PhaseAccepted
	return rhpamuser, nil
}

func (ph *phaseHandler) Accepted(rhpamuser *rhpamv1alpha1.RhpamUser) (*rhpamv1alpha1.RhpamUser, error) {
	log.Info("Phase Accepted")

	// look for a rhpamdev object
	rhpamdevlist := &rhpamv1alpha1.RhpamDevList{}
	// Try to find the resource, it may already exist
	opts := &client.ListOptions{}
	//List(ctx context.Context, opts *ListOptions, list runtime.Object) error
	err := ph.client.List(context.TODO(), opts, rhpamdevlist)
	if err != nil {
		// Error reading the object
		log.Error(err, "Error when getting rhpamdev list")
		return nil, err
	}

	for _, rhpamdev := range rhpamdevlist.Items {
		log.Info("Rhpamdev found", "Name", rhpamdev.Name, "Status", rhpamdev.Status.Phase)
		if rhpamdev.Status.Phase == rhpamv1alpha1.PhaseComplete {
			// set to phase reconcilereconcile
		}
		if rhpamdev.Status.Phase == rhpamv1alpha1.PhaseRealmProvisioned {
			// create default users
			// update rhpamdev
			// set to phase reconcile
			ssoClient, err := ph.authenticatedClient()
			if err != nil {
				return nil, err
			}
			// create default roles
			for _, role := range defaultRoles {
				if err := ph.createRole(ssoClient, rhpamdev.Status.Realm, role); err != nil {
					return nil, err
				}
			}
			// create default users
			for _, user := range defaultUsers {
				if err := ph.createUser(ssoClient, rhpamdev.Status.Realm, user.Username, user.Password); err != nil {
					return nil, err
				}
				for _, role := range user.Roles {
					if err := ph.createUserRole(ssoClient, rhpamdev.Status.Realm, user.Username, role); err != nil {
						return nil, err
					}
				}
			}
			// create roles
			for _, role := range rhpamuser.Spec.Roles {
				if err := ph.createRole(ssoClient, rhpamdev.Status.Realm, role.Name); err != nil {
					return nil, err
				}
			}
			// create users
			for _, user := range rhpamuser.Spec.Users {
				if err := ph.createUser(ssoClient, rhpamdev.Status.Realm, user.Username, user.Password); err != nil {
					return nil, err
				}
				for _, role := range user.Roles {
					if err := ph.createUserRole(ssoClient, rhpamdev.Status.Realm, user.Username, role); err != nil {
						return nil, err
					}
				}
			}
			//set to phase Reconcile
			rhpamuser.Status.Phase = rhpamv1alpha1.PhaseReconcile
		}
	}

	return rhpamuser, nil
}

func (ph *phaseHandler) Reconcile(rhpamuser *rhpamv1alpha1.RhpamUser) (*rhpamv1alpha1.RhpamUser, error) {

	return rhpamuser, nil
}

func (ph *phaseHandler) authenticatedClient() (keycloak.KeycloakInterface, error) {
	//get sso username, password, admin url
	if err := ph.readSSOSecret(); err != nil {
		log.Error(err, "Error reading RHSSO secret")
		return nil, err
	}
	ssoClient, err := ph.keycloakFactory.AuthenticatedClient()
	if err != nil {
		return nil, err
	}
	return ssoClient, nil
}

func (ph *phaseHandler) createRole(ssoClient keycloak.KeycloakInterface, realm, role string) error {
	_, err := ssoClient.GetRole(role, realm)
	if err == nil {
		return nil
	}
	roleParams := keycloak.RoleParameters{RoleName: role}
	roleHelper := keycloak.NewRoleHelper()
	json, err := roleHelper.LoadRoleTemplate(roleParams)
	if err != nil {
		return err
	}
	if err := ssoClient.CreateRole(json, realm); err != nil {
		return errors.Wrap(err, "Error creating role in rhsso")
	}

	return nil
}

func (ph *phaseHandler) createUser(ssoClient keycloak.KeycloakInterface, realm, username, password string) error {
	_, err := ssoClient.FindUserByUsername(username, realm)
	if err == nil {
		return nil
	}
	userParams := keycloak.UserParameters{Username: username, Password: password}
	userHelper := keycloak.NewUserHelper()
	json, err := userHelper.LoadUserTemplate(userParams)
	if err != nil {
		return err
	}
	log.Info("Creating User")
	if err := ssoClient.CreateUser(json, realm); err != nil {
		return errors.Wrap(err, "Error creating user in rhsso")
	}

	return nil
}

func (ph *phaseHandler) createUserRole(ssoClient keycloak.KeycloakInterface, realm, username, rolename string) error {
	user, err := ssoClient.FindUserByUsername(username, realm)
	if err != nil {
		return err
	}
	roleMappings, err := ssoClient.GetRoleMappings(user.ID, realm)
	if err != nil {
		return err
	}
	roleMap := make(map[string]*keycloak.KeycloakRole)
	for _, role := range roleMappings {
		roleMap[role.Name] = role
	}
	_, ok := roleMap[rolename]
	if !ok {
		role, err := ssoClient.GetRole(rolename, realm)
		if err != nil {
			return err
		}
		userRoleParams := keycloak.UserRoleParameters{RoleName: rolename, RoleId: role.ID}
		userRoleHelper := keycloak.NewUserRoleHelper()
		json, err := userRoleHelper.LoadUserRoleTemplate(userRoleParams)
		if err := ssoClient.CreateUserRole(json, user.ID, realm); err != nil {
			return errors.Wrap(err, "Error creating userrole in rhsso")
		}
	}

	return nil

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
