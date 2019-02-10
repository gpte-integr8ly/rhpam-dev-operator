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
					if err := ph.createUserRealmRole(ssoClient, rhpamdev.Status.Realm, user.Username, role); err != nil {
						return nil, err
					}
				}
			}
			//set to phase Reconcile
			rhpamuser.Status.Phase = rhpamv1alpha1.PhaseReconcile
			rhpamuser.Status.Realm = rhpamdev.Status.Realm
		}
	}

	return rhpamuser, nil
}

func (ph *phaseHandler) Reconcile(rhpamuser *rhpamv1alpha1.RhpamUser) (*rhpamv1alpha1.RhpamUser, error) {
	ssoClient, err := ph.authenticatedClient()
	if err != nil {
		return nil, err
	}
	if err := ph.reconcileRoles(ssoClient, rhpamuser); err != nil {
		return nil, err
	}
	if err := ph.reconcileUsers(ssoClient, rhpamuser); err != nil {
		return nil, err
	}

	return rhpamuser, nil
}

func (ph *phaseHandler) reconcileRoles(ssoClient keycloak.KeycloakInterface, rhpamuser *rhpamv1alpha1.RhpamUser) error {
	roles, err := ssoClient.ListRoles(rhpamuser.Status.Realm)
	if err != nil {
		return err
	}
	rolesPairList := map[string]*keycloak.KeycloakRolePair{}
	for _, role := range roles {
		rolesPairList[role.Name] = &keycloak.KeycloakRolePair{KcRole: role.Name, SpecRole: ""}
	}
	for _, role := range rhpamuser.Spec.Roles {
		if _, ok := rolesPairList[role.Name]; ok {
			rolesPairList[role.Name].SpecRole = role.Name
		} else {
			rolesPairList[role.Name] = &keycloak.KeycloakRolePair{KcRole: "", SpecRole: role.Name}
		}
	}
	for _, rolePair := range rolesPairList {
		if err := ph.reconcileRole(ssoClient, rhpamuser.Status.Realm, rolePair.KcRole, rolePair.SpecRole); err != nil {
			return err
		}
	}
	return nil
}

func (ph *phaseHandler) reconcileUsers(ssoClient keycloak.KeycloakInterface, rhpamuser *rhpamv1alpha1.RhpamUser) error {
	users, err := ssoClient.ListUsers(rhpamuser.Status.Realm)
	if err != nil {
		return err
	}
	userPairList := map[string]*keycloak.KeycloakUserPair{}
	for _, user := range users {
		roleMappings, err := ssoClient.GetRealmRoleMappings(user.ID, rhpamuser.Status.Realm)
		if err != nil {
			return err
		}
		for _, role := range roleMappings {
			user.RealmRoles = append(user.RealmRoles, role.Name)
		}
		userPairList[user.UserName] = &keycloak.KeycloakUserPair{KcUser: user, SpecUser: nil}
	}
	for _, user := range rhpamuser.Spec.Users {
		keycloakUser := &keycloak.KeycloakUser{KeycloakApiUser: &keycloak.KeycloakApiUser{UserName: user.Username, RealmRoles: user.Roles}, Password: &user.Password}
		if _, ok := userPairList[user.Username]; ok {
			userPairList[user.Username].SpecUser = keycloakUser
		} else {
			userPairList[user.Username] = &keycloak.KeycloakUserPair{KcUser: nil, SpecUser: keycloakUser}
		}
	}
	for _, userPair := range userPairList {
		if err := ph.reconcileUser(ssoClient, rhpamuser.Status.Realm, userPair.KcUser, userPair.SpecUser); err != nil {
			return err
		}
	}
	return nil
}

func (ph *phaseHandler) reconcileRole(ssoClient keycloak.KeycloakInterface, realm, kcRole, specRole string) error {
	if specRole == "" && !isDefaultRole(kcRole) {
		return ssoClient.DeleteRole(kcRole, realm)
	}
	if kcRole == "" {
		return ph.createRole(ssoClient, realm, specRole)
	}
	return nil
}

func (ph *phaseHandler) reconcileUser(ssoClient keycloak.KeycloakInterface, realm string, kcUser, specUser *keycloak.KeycloakUser) error {
	if specUser == nil {
		if !isDefaultUser(kcUser.UserName) {
			ph.deleteUser(ssoClient, realm, kcUser.UserName)
		}
	} else if kcUser == nil {
		user := &rhpamv1alpha1.User{Username: specUser.UserName, Password: *specUser.Password, Roles: specUser.RealmRoles}
		return ph.createUserWithRealmRoles(ssoClient, realm, user)
	} else {
		// reconcile roles
		rolePairList := map[string]*keycloak.KeycloakRolePair{}
		for _, kcUserRole := range kcUser.RealmRoles {
			rolePairList[kcUserRole] = &keycloak.KeycloakRolePair{KcRole: kcUserRole, SpecRole: ""}
		}
		for _, specUserRole := range specUser.RealmRoles {
			if _, ok := rolePairList[specUserRole]; ok {
				rolePairList[specUserRole].SpecRole = specUserRole
			} else {
				rolePairList[specUserRole] = &keycloak.KeycloakRolePair{KcRole: "", SpecRole: specUserRole}
			}
		}
		for _, rolePair := range rolePairList {
			if err := ph.reconcileUserRealmRole(ssoClient, realm, kcUser.UserName, rolePair.KcRole, rolePair.SpecRole); err != nil {
				return err
			}
		}
	}

	return nil

}

func (ph *phaseHandler) reconcileUserRealmRole(ssoClient keycloak.KeycloakInterface, realm, user, kcRole, specRole string) error {
	if kcRole == "" {
		return ph.createUserRealmRole(ssoClient, realm, user, specRole)
	} else if specRole == "" {
		return ph.deleteUserRealmRole(ssoClient, realm, user, kcRole)
	}
	return nil
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

func (ph *phaseHandler) createUserWithRealmRoles(ssoClient keycloak.KeycloakInterface, realm string, user *rhpamv1alpha1.User) error {
	if err := ph.createUser(ssoClient, realm, user.Username, user.Password); err != nil {
		return err
	}
	for _, role := range user.Roles {
		if err := ph.createUserRealmRole(ssoClient, realm, user.Username, role); err != nil {
			return err
		}
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
	if err := ssoClient.CreateUser(json, realm); err != nil {
		return errors.Wrap(err, "Error creating user in rhsso")
	}

	return nil
}

func (ph *phaseHandler) deleteUser(ssoClient keycloak.KeycloakInterface, realm, username string) error {
	user, err := ssoClient.FindUserByUsername(username, realm)
	if err != nil {
		return nil
	}
	if err := ssoClient.DeleteUser(user.ID, realm); err != nil {
		return errors.Wrap(err, "Error deleting user in rhsso")
	}

	return nil
}

func (ph *phaseHandler) createUserRealmRole(ssoClient keycloak.KeycloakInterface, realm, username, rolename string) error {
	user, err := ssoClient.FindUserByUsername(username, realm)
	if err != nil {
		return err
	}
	roleMappings, err := ssoClient.GetRealmRoleMappings(user.ID, realm)
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
		if err := ssoClient.CreateUserRealmRole(json, user.ID, realm); err != nil {
			return errors.Wrap(err, "Error creating userrole in rhsso")
		}
	}

	return nil
}

func (ph *phaseHandler) deleteUserRealmRole(ssoClient keycloak.KeycloakInterface, realm, username, rolename string) error {
	user, err := ssoClient.FindUserByUsername(username, realm)
	if err != nil {
		return err
	}
	roleMappings, err := ssoClient.GetRealmRoleMappings(user.ID, realm)
	if err != nil {
		return err
	}
	roleMap := make(map[string]*keycloak.KeycloakRole)
	for _, role := range roleMappings {
		roleMap[role.Name] = role
	}
	_, ok := roleMap[rolename]
	if ok {
		role, err := ssoClient.GetRole(rolename, realm)
		if err != nil {
			return err
		}
		userRoleParams := keycloak.UserRoleParameters{RoleName: rolename, RoleId: role.ID}
		userRoleHelper := keycloak.NewUserRoleHelper()
		json, err := userRoleHelper.LoadUserRoleTemplate(userRoleParams)
		if err := ssoClient.DeleteUserRealmRole(json, user.ID, realm); err != nil {
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

func isDefaultRole(role string) bool {
	for _, r := range defaultRoles {
		if role == r {
			return true
		}
	}
	return false
}

func isDefaultUser(username string) bool {
	for _, u := range defaultUsers {
		if username == u.Username {
			return true
		}
	}
	return false
}
