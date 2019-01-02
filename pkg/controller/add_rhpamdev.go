package controller

import (
	"github.com/gpte-naps/rhpam-dev-operator/pkg/controller/rhpamdev"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, rhpamdev.Add)
}
