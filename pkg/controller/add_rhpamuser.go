package controller

import (
	"github.com/gpte-integr8ly/rhpam-dev-operator/pkg/controller/rhpamuser"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, rhpamuser.Add)
}
