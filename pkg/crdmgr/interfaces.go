package crdmgr

import "github.com/lz006/extended-awx-client-go/eawx"

type HostGroup interface {
	Group() *eawx.Group

	Hosts() []*eawx.Host
}
