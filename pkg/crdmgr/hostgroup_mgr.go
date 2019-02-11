package crdmgr

import (
	"reflect"

	"github.com/spf13/viper"

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
					case "absent":
						deleteByAWXHostGroup(hostGroup)
					case "ignore":
						// Ignoring hosts/groups from awx.
						log.Info("Group \"" + group.Name() + "\" is configured te be ingored")
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

			purgeHostGroups(awxHostGroups)

		}
	}

}

func purgeHostGroups(awxHostGroups *map[string]HostGroup) {
	// Purging:
	// Delete instances of "CRD: HostGroup" if they do not exist in awx anymore
	// First load all existing instances
	existingHostGroups := getHostGroups()

	// Loop trough exisiting instances and check if their corresnponding group in awx inventory still exist
	for _, k8sInstance := range existingHostGroups.Items {
		if _, ok := (*awxHostGroups)[k8sInstance.Name]; !ok {
			// AWX HostGroup does not exist anymore so delete it in k8s/openshift
			deleteHostGroup(&k8sInstance)
			log.Info("HostGroup: \"" + k8sInstance.GetName() + "\" deleted successfully - Reason: No AWX pendant anymore")
		}
	}
}

func deleteByAWXHostGroup(hostGroup HostGroup) {
	existingHostGroups := getHostGroups()

	// Loop trough exisiting instances and look for corresnponding group
	for _, k8sInstance := range existingHostGroups.Items {
		if k8sInstance.Name == hostGroup.Group().Name() {
			// AWX HostGroup still exists but is marked for deletion so delete it in k8s/openshift too
			deleteHostGroup(&k8sInstance)
			log.Info("HostGroup: \"" + k8sInstance.GetName() + "\" deleted successfully - Marked as \"absent\" on AWX")
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

	// Check if operator-managed label is still "yes"
	labelKey := viper.GetString("k8s_label_operator_indicator")
	managedLabel := found.GetObjectMeta().GetLabels()[labelKey]
	if managedLabel != "yes" {
		log.Info("Skipping HostGroup \"" + candidate.Group().Name() + "\" due to label \"" + viper.GetString("k8s_namespace") + "\" is not set to \"yes\"")
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
	log.Info("Services mangaged by k8s/openshit will be ignored by now. If there is a demand such a feature can be implemented very quickly. Skipped HostGroup: \"" + hostGroup.Group().Name() + "\"")
}

func isHostGroupEqualTo(new *cachev1alpha1.HostGroup, old *cachev1alpha1.HostGroup) bool {

	result := true

	if len(new.Spec.HostGroup.Vars.Endpoints) != len(old.Spec.HostGroup.Vars.Endpoints) ||
		len(new.Spec.Endpoints) != len(old.Spec.Endpoints) ||
		new.Spec.HostGroup.Id != old.Spec.HostGroup.Id ||
		new.Spec.HostGroup.Name != old.Spec.HostGroup.Name {

		result = false

		return result
	}

	// Building maps is necessary due not predictible order of elements in arrays
	// Compare cachev1alpha1.Endpoint objects
	newHGR := make(map[string]*cachev1alpha1.Endpoint)
	for _, ep := range new.Spec.HostGroup.Vars.Endpoints {
		newHGR[ep.PortName] = &ep
	}

	oldHGR := make(map[string]*cachev1alpha1.Endpoint)
	for _, ep := range old.Spec.HostGroup.Vars.Endpoints {
		oldHGR[ep.PortName] = &ep
	}

	if !reflect.DeepEqual(newHGR, oldHGR) {
		result = false
	}

	// Compare strings
	newEP := make(map[string]string)
	for _, val := range new.Spec.Endpoints {
		newEP[val] = val
	}

	oldEP := make(map[string]string)
	for _, val := range old.Spec.Endpoints {
		oldEP[val] = val
	}

	if !reflect.DeepEqual(newEP, oldEP) {
		result = false
	}

	return result
}
