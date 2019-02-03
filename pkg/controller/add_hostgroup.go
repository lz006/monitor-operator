package controller

import (
	"github.com/lz006/monitor-operator/pkg/controller/hostgroup"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, hostgroup.Add)
}
