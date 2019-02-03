package awxmgr

import (
	"github.com/lz006/extended-awx-client-go/eawx"
)

type HostGroup struct {
	group *eawx.Group

	hosts []*eawx.Host
}

func (hg HostGroup) Group() *eawx.Group {
	return hg.group
}

func (hg HostGroup) Hosts() []*eawx.Host {
	return hg.hosts
}
