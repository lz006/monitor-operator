package crdmgr

import (
	"github.com/prometheus/common/log"
	mgr "sigs.k8s.io/controller-runtime/pkg/manager"
)

func init() {

}

func Start(channel chan *map[string]HostGroup, mgr mgr.Manager) {

	setupK8sAccess(mgr)

	var hostGroups *map[string]HostGroup

	for ok := true; ok; hostGroups, ok = <-channel {

		if hostGroups != nil {
			for _, hostGroup := range *hostGroups {
				group := hostGroup.Group()

				if group != nil && group.Vars() != nil {
					switch mType := group.Vars().MType(); mType {
					case "outside":
						manageOutsideHostGroups(hostGroup)
					case "inside":
						manageInsideHostGroups(hostGroup)
					default:
						log.Info("Group: \"" + group.Name() + "\" has unknown type \"" + mType + "\" configured -> Ignoring group")
					}
				} else {
					if group != nil {
						log.Info("Group: \"" + group.Name() + "\" has no group vars configured -> Ignoring group")
					} else {
						log.Error("Empty Group! Please check inventory on awx/ansible-tower")
					}

				}
			}
		}
	}

}

func manageOutsideHostGroups(hostGroup HostGroup) {
	getHostGroups()
	//createHostGroup(hostGroup)
	log.Info("outside")
}

func manageInsideHostGroups(hostGroup HostGroup) {
	getHostGroups()
	log.Info("inside")
}
