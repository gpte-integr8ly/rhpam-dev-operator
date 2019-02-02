package keycloak

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("keycloak_client")

const (
	authUrl = "auth/realms/master/protocol/openid-connect/token"
)

type Requester interface {
	Do(req *http.Request) (*http.Response, error)
}

type TokenResponse struct {
	AccessToken      string `json:"access_token"`
	ExpiresIn        int    `json:"expires_in"`
	RefreshExpiresIn int    `json:"refresh_expires_in"`
	RefreshToken     string `json:"refresh_token"`
	TokenType        string `json:"token_type"`
	NotBeforePolicy  int    `json:"not-before-policy"`
	SessionState     string `json:"session_state"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

type Client struct {
	requester Requester
	URL       string
	token     string
}

type KeycloakFactory struct {
	AdminUser     string
	AdminPassword string
	AdminUrl      string
}

func NewKeycloakFactory() *KeycloakFactory {
	return &KeycloakFactory{}
}

// T is a generic type for keycloak spec resources
type T interface{}

// Generic create function for creating new Keycloak resources
func (c *Client) create(obj []byte, resourcePath, resourceName string, statuscode int) error {
	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/auth/admin/%s", c.URL, resourcePath),
		bytes.NewBuffer(obj),
	)
	if err != nil {
		log.Error(err, "error creating POST request", "resource", resourceName)
		return errors.Wrapf(err, "error creating POST %s request", resourceName)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.token))
	res, err := c.requester.Do(req)

	if err != nil {
		log.Error(err, "error on request.")
		return errors.Wrapf(err, "error performing POST %s request", resourceName)
	}
	defer res.Body.Close()

	if res.StatusCode != statuscode {
		return fmt.Errorf("failed to create %s: (%d) %s", resourceName, res.StatusCode, res.Status)
	}

	return nil
}

func (c *Client) CreateRealm(json []byte) error {
	return c.create(json, "realms", "realm", 201)
}

func (c *Client) CreateClient(json []byte, realm string) error {
	return c.create(json, fmt.Sprintf("realms/%s/clients", realm), "client", 201)
}

func (c *Client) CreateRole(json []byte, realm string) error {
	return c.create(json, fmt.Sprintf("realms/%s/roles", realm), "role", 201)
}

func (c *Client) CreateUser(json []byte, realm string) error {
	return c.create(json, fmt.Sprintf("realms/%s/users", realm), "user", 201)
}

func (c *Client) CreateUserRole(json []byte, user, realm string) error {
	return c.create(json, fmt.Sprintf("realms/%s/users/%s/role-mappings/realm", realm, user), "user-role", 204)
}

func (c *Client) FindUserByUsername(name, realm string) (*KeycloakApiUser, error) {
	result, err := c.get(fmt.Sprintf("realms/%s/users?username=%s", realm, name), "user", func(body []byte) (T, error) {
		var users []*KeycloakApiUser
		if err := json.Unmarshal(body, &users); err != nil {
			return nil, err
		}
		if len(users) == 0 {
			return &KeycloakApiUser{}, nil
		}
		return users[0], nil
	})
	if err != nil {
		return nil, err
	}
	user := result.(*KeycloakApiUser)
	if user.ID == "" {
		return nil, errors.New("Not Found")
	}
	return user, nil
}

// Generic get function for returning a Keycloak resource
func (c *Client) get(resourcePath, resourceName string, unMarshalFunc func(body []byte) (T, error)) (T, error) {
	u := fmt.Sprintf("%s/auth/admin/%s", c.URL, resourcePath)
	req, err := http.NewRequest(
		"GET",
		u,
		nil,
	)
	if err != nil {
		log.Error(err, "error creating GET request", "resource", resourceName)
		return nil, errors.Wrapf(err, "error creating GET %s request", resourceName)
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.token))
	res, err := c.requester.Do(req)
	if err != nil {
		log.Error(err, "error on request")
		return nil, errors.Wrapf(err, "error performing GET %s request", resourceName)
	}

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("failed to GET %s: (%d) %s", resourceName, res.StatusCode, res.Status)
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Error(err, "error reading response %+v")
		return nil, errors.Wrapf(err, "error reading %s GET response", resourceName)
	}

	obj, err := unMarshalFunc(body)
	if err != nil {
		log.Error(err, "Error unmarshalling response")
		return nil, err
	}

	return obj, nil
}

func (c *Client) GetClientSecret(clientId, realmName string) (string, error) {
	result, err := c.get(fmt.Sprintf("realms/%s/clients/%s/client-secret", realmName, clientId), "client-secret", func(body []byte) (T, error) {
		res := map[string]string{}
		if err := json.Unmarshal(body, &res); err != nil {
			return nil, err
		}
		return res["value"], nil
	})
	if err != nil {
		return "", errors.Wrap(err, "failed to get: "+fmt.Sprintf("realms/%s/clients/%s/client-secret", realmName, clientId))
	}
	return result.(string), nil
}

func (c *Client) GetUser(userID, realmName string) (*KeycloakUser, error) {
	result, err := c.get(fmt.Sprintf("realms/%s/users/%s", realmName, userID), "user", func(body []byte) (T, error) {
		user := &KeycloakApiUser{}
		err := json.Unmarshal(body, user)
		return user, err
	})
	ret := &KeycloakUser{
		KeycloakApiUser: result.(*KeycloakApiUser),
	}
	return ret, err
}

func (c *Client) GetRole(roleName, realmName string) (*KeycloakRole, error) {
	result, err := c.get(fmt.Sprintf("realms/%s/roles/%s", realmName, roleName), "role", func(body []byte) (T, error) {
		role := &KeycloakRole{}
		err := json.Unmarshal(body, role)
		return role, err
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get: "+fmt.Sprintf("realms/%s/roles/%s", realmName, roleName))
	}
	return result.(*KeycloakRole), nil
}

func (c *Client) GetRoleMappings(user, realm string) ([]*KeycloakRole, error) {
	result, err := c.get(fmt.Sprintf("realms/%s/users/%s/role-mappings/realm", realm, user), "role-mapping", func(body []byte) (T, error) {
		var roleMappings []*KeycloakRole
		err := json.Unmarshal(body, &roleMappings)
		return roleMappings, err
	})
	if err != nil {
		return nil, err
	}
	res, ok := result.([]*KeycloakRole)

	if !ok {
		return nil, errors.New("error decoding list clients response")
	}

	return res, nil
}

// Generic list function for listing Keycloak resources
func (c *Client) list(resourcePath, resourceName string, unMarshalListFunc func(body []byte) (T, error)) (T, error) {
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/auth/admin/%s", c.URL, resourcePath),
		nil,
	)
	if err != nil {
		log.Error(err, "error creating LIST request", "resource", resourceName)
		return nil, errors.Wrapf(err, "error creating LIST %s request", resourceName)
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.token))
	res, err := c.requester.Do(req)
	if err != nil {
		log.Error(err, "error on request")
		return nil, errors.Wrapf(err, "error performing LIST %s request", resourceName)
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode > 299 {
		return nil, fmt.Errorf("failed to LIST %s: (%d) %s", resourceName, res.StatusCode, res.Status)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Error(err, "error reading response")
		return nil, errors.Wrapf(err, "error reading %s LIST response", resourceName)
	}

	objs, err := unMarshalListFunc(body)
	if err != nil {
		log.Error(err, "Error unmarshalling response")
	}
	return objs, nil
}

func (c *Client) ListClients(realmName string) ([]*KeycloakClient, error) {
	result, err := c.list(fmt.Sprintf("realms/%s/clients", realmName), "clients", func(body []byte) (T, error) {
		var clients []*KeycloakClient
		err := json.Unmarshal(body, &clients)
		return clients, err
	})

	if err != nil {
		return nil, err
	}

	res, ok := result.([]*KeycloakClient)

	if !ok {
		return nil, errors.New("error decoding list clients response")
	}

	return res, nil

}

// login requests a new auth token from Keycloak
func (c *Client) login(user, pass string) error {
	form := url.Values{}
	form.Add("username", user)
	form.Add("password", pass)
	form.Add("client_id", "admin-cli")
	form.Add("grant_type", "password")

	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/%s", c.URL, authUrl),
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return errors.Wrap(err, "error creating login request")
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	res, err := c.requester.Do(req)
	if err != nil {
		log.Error(err, "error on request")
		return errors.Wrap(err, "error performing token request")
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Error(err, "error reading response")
		return errors.Wrap(err, "error reading token response")
	}

	tokenRes := &TokenResponse{}
	err = json.Unmarshal(body, tokenRes)
	if err != nil {
		return errors.Wrap(err, "error parsing token response")
	}

	if tokenRes.Error != "" {
		log.Info("error with request", "ErrorDescription", tokenRes.ErrorDescription)
		return errors.New(tokenRes.ErrorDescription)
	}

	c.token = tokenRes.AccessToken

	return nil
}

// defaultRequester returns a default client for requesting http endpoints
func defaultRequester() Requester {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	c := &http.Client{Transport: transport, Timeout: time.Second * 10}
	return c
}

type KeycloakInterface interface {
	CreateRealm(json []byte) error

	CreateRole(json []byte, realm string) error

	CreateClient(json []byte, realmName string) error
	GetClientSecret(clientId, realmName string) (string, error)
	ListClients(realmName string) ([]*KeycloakClient, error)

	CreateUser(json []byte, realmName string) error
	FindUserByUsername(name, realm string) (*KeycloakApiUser, error)
	GetUser(userID, realmName string) (*KeycloakUser, error)

	GetRole(roleName string, realmName string) (*KeycloakRole, error)
	GetRoleMappings(user string, realm string) ([]*KeycloakRole, error)
	CreateUserRole(json []byte, userId, realm string) error
}

type KeycloakClientFactory interface {
	AuthenticatedClient() (KeycloakInterface, error)
}

func (kf *KeycloakFactory) AuthenticatedClient() (KeycloakInterface, error) {
	//Refactor to read sso secret if kf is not initialized
	client := &Client{
		URL:       kf.AdminUrl,
		requester: defaultRequester(),
	}
	if err := client.login(kf.AdminUser, kf.AdminPassword); err != nil {
		return nil, err
	}
	return client, nil
}
