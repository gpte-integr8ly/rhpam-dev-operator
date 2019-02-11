package common

import (
	"context"
	"errors"
	"fmt"
	"os"

	"k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ReadSSOSecret(client client.Client) (*v1.Secret, error) {

	//get sso username, password, url
	ssoNamespace := os.Getenv("SSO_NAMESPACE")
	if ssoNamespace == "" {
		return nil, errors.New("Environment variable SSO_NAMESPACE is not set.")
	}
	ssoAdminCredentialsSecret := os.Getenv("SSO_ADMIN_CREDENTIALS_SECRET")
	if ssoAdminCredentialsSecret == "" {
		return nil, errors.New("Environment variable SSO_ADMIN_CREDENTIALS_SECRET is not set.")
	}

	//read secret in sso namespace, extract sso username, sso password, sso admin url, store in keycloakclient
	secret := &v1.Secret{}
	selector := types.NamespacedName{
		Namespace: ssoNamespace,
		Name:      ssoAdminCredentialsSecret,
	}

	err := client.Get(context.TODO(), selector, secret)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, fmt.Errorf("Secret %s in namespace %s not found: %s", ssoAdminCredentialsSecret, ssoNamespace, err)
		} else {
			return nil, fmt.Errorf("Error retrieving secret %s in namespace %s: %s", ssoAdminCredentialsSecret, ssoNamespace, err)
		}
	}

	return secret, nil
}

func AddFinalizer(obj runtime.Object, value string) error {
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return err
	}
	finalizers := sets.NewString(accessor.GetFinalizers()...)
	finalizers.Insert(value)
	accessor.SetFinalizers(finalizers.List())
	return nil
}
