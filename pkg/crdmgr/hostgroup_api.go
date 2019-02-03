package crdmgr

import (
	ctx "context"
	"net"
	"strings"

	"github.com/lz006/extended-awx-client-go/eawx"

	"github.com/prometheus/common/log"
	cli "sigs.k8s.io/controller-runtime/pkg/client"

	mgr "sigs.k8s.io/controller-runtime/pkg/manager"

	cachev1alpha1 "github.com/lz006/monitor-operator/pkg/apis/cache/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

var manager mgr.Manager
var client cli.Client

func init() {

}

func setupK8sAccess(mgr mgr.Manager) {
	manager = mgr
	client = mgr.GetClient()
}

func getHostGroups() {
	hostGroupList := &cachev1alpha1.HostGroupList{}
	labelSelector := labels.SelectorFromSet(map[string]string{"operator-managed": "true"})
	listOps := &cli.ListOptions{Namespace: "openshift-monitoring", LabelSelector: labelSelector}
	err := client.List(ctx.TODO(), listOps, hostGroupList)
	if err != nil {
		log.Info("error k8s communication")
	}
	groupList := hostGroupList.Items
	item := hostGroupList.Items[0]

	log.Info(len(groupList))
	log.Info(item.APIVersion)

}

func createHostGroup(hostGroup HostGroup) {

	ipv4List := ipv4List(hostGroup.Hosts())
	k8sHostGroup := hostGroupForGroupAndHosts(hostGroup.Group(), ipv4List)

	err := client.Create(ctx.TODO(), k8sHostGroup)
	if err != nil {
		log.Info("error k8s communication")
	}
	log.Info("hat getan")
}

// serviceForMemcached returns a memcached Service object
func hostGroupForGroupAndHosts(group *eawx.Group, hosts []string) *cachev1alpha1.HostGroup {

	ls := labelsForHostGroup((*group).Name())

	hg := &cachev1alpha1.HostGroup{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "cache.sulzer.de/v1alpha1",
			Kind:       "HostGroup",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      group.Name(),
			Namespace: "openshift-monitoring",
			Labels:    ls,
		},
		Spec: cachev1alpha1.HostGroupSpec{
			HostGroup: cachev1alpha1.Group{
				Id:   group.Id(),
				Name: group.Name(),
				Vars: cachev1alpha1.Variables{
					Type:            group.Vars().MType(),
					Endpoint:        group.Vars().Endpoint(),
					BearerTokenFile: group.Vars().BearerTokenFile(),
					Port:            group.Vars().Port(),
					Scheme:          group.Vars().Scheme(),
					TargetPort:      group.Vars().TargetPort(),
					TLSConf: cachev1alpha1.TLSConfig{
						CAFile:             group.Vars().TLSConfig().CAFile(),
						Hostname:           group.Vars().TLSConfig().Hostname(),
						InsecureSkipVerify: group.Vars().TLSConfig().InsecureSkipVerify(),
					},
				},
			},
			Endpoints: hosts,
		},
	}
	return hg
}

func labelsForHostGroup(name string) map[string]string {
	return map[string]string{"k8s-app": name, "operator-managed": "true"}
}

func ipv4List(hosts []*eawx.Host) []string {
	var list []string
	for _, host := range hosts {

		ipv4Addr := determineIPV4Address(host)

		list = append(list, ipv4Addr)
	}
	return list
}

func determineIPV4Address(host *eawx.Host) string {
	if host.IPV4() != "" {
		return host.IPV4()
	}

	// Get Ip Adress from hostname or ip address itself
	addrs, err := net.LookupHost(host.Name())
	if err != nil {
		log.Error("Looking up ip adress of host \"" + host.Name() + "\" failed! Please check connectivity")
		return ""
	}

	// Get the first ipv4 address if there are multiple results
	for _, a := range addrs {
		if strings.Count(a, ":") < 2 {
			return a
		}
	}

	log.Error("Looking up ip adress of host \"" + host.Name() + "\" failed! Please check system logs")
	return ""
}
