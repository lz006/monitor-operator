package crdmgr

import (
	"reflect"

	cachev1alpha1 "github.com/lz006/monitor-operator/pkg/apis/cache/v1alpha1"
	"github.com/prometheus/common/log"
	"k8s.io/apimachinery/pkg/api/errors"
	mgr "sigs.k8s.io/controller-runtime/pkg/manager"
)

func init() {

}

func Start(channel chan *map[string]HostGroup, mgr mgr.Manager) {

	setupK8sAccess(mgr)

	var awxHostGroups *map[string]HostGroup

	for ok := true; ok; awxHostGroups, ok = <-channel {

		if awxHostGroups != nil {
			// Create & update instances of CRD: HostGroup
			for _, hostGroup := range *awxHostGroups {
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

			// Delete instances of "CRD: HostGroup" if they do not exist in awx anymore
			// First load all existing instances
			existingHostGroups := getHostGroups()

			// Loop trough exisiting instances and check if their corresnponding group in awx inventory still exist
			for _, k8sInstance := range existingHostGroups.Items {
				if _, ok := (*awxHostGroups)[k8sInstance.Name]; !ok {
					// AWX HostGroup does not exist anymore so delete it in k8s/openshift
					deleteHostGroup(&k8sInstance)
					k8sInstance.GetName()
					log.Info("HostGroup: \"" + k8sInstance.GetName() + "\" deleted successfully - No AWX pendant anymore")
				}
			}

		}
	}

}

func manageOutsideHostGroups(candidate HostGroup) {

	// Look for pre-existing HostGroup-Instances
	found := &cachev1alpha1.HostGroup{}

	err := getHostGroup(candidate, found)
	if err != nil && errors.IsNotFound(err) {
		// Related CRD-HostGroupInstance could not be found
		// Create a new instance
		createHostGroup(candidate)
		log.Info("CRD-HostGroup instance created for AWX inventory group: \"" + candidate.Group().Name() + "\"")
		return
	} else if err != nil {
		log.Error("Could not get HostGroup from API Server. Looked up for AWX-HostGroup: \"" + candidate.Group().Name() + "\"")
		return
	}
	// Clean err
	err = nil

	// Check if operator-managed label is still "true"
	managedLabel := found.GetObjectMeta().GetLabels()["operator-managed"]
	if managedLabel != "true" {
		log.Info("Skipping HostGroup \"" + candidate.Group().Name() + "\" due to label \"operator-managed\" is not set to \"true\"")
		return
	}

	// Create HostGroup-Definition in order to be able to compare it with existing instance
	ipAddrList := ipList(candidate.Hosts())
	newHostGroup := hostGroupForGroupAndHosts(candidate.Group(), ipAddrList)

	if isHostGroupEqualTo(newHostGroup, found) {
		return
	} else {
		found.Spec.HostGroup = newHostGroup.Spec.HostGroup
		found.Spec.Endpoints = newHostGroup.Spec.Endpoints
		err = updateHostGroup(found)
		if err == nil {
			log.Info("CRD-HostGroup instance updated sucessfully for AWX-Group: \"" + candidate.Group().Name() + "\"")
		} else {
			log.Error("Could not update HostGroup via API Server. Tried for AWX-HostGroup: \"" + candidate.Group().Name() + "\"")
		}
	}

}

func manageInsideHostGroups(hostGroup HostGroup) {
	getHostGroups()
	log.Info("inside")
}

func isHostGroupEqualTo(new *cachev1alpha1.HostGroup, old *cachev1alpha1.HostGroup) bool {

	if !reflect.DeepEqual(new.Spec.HostGroup, old.Spec.HostGroup) {
		return false
	} else if !reflect.DeepEqual(new.Spec.Endpoints, old.Spec.Endpoints) {
		return false
	} else {
		return true
	}
}
