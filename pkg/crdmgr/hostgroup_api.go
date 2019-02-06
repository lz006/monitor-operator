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
	"k8s.io/apimachinery/pkg/types"
)

var manager mgr.Manager
var client cli.Client

func init() {

}

func setupK8sAccess(mgr mgr.Manager) {
	manager = mgr
	client = mgr.GetClient()
}

func getHostGroups() *cachev1alpha1.HostGroupList {

	hostGroupList := &cachev1alpha1.HostGroupList{}
	labelSelector := labels.SelectorFromSet(map[string]string{"operator-managed": "true"})
	listOps := &cli.ListOptions{Namespace: "openshift-monitoring", LabelSelector: labelSelector}
	err := client.List(ctx.TODO(), listOps, hostGroupList)
	if err != nil {
		log.Info("Could not get HostGroupList from API Server.")
	}

	return hostGroupList

}

func getHostGroup(hgr HostGroup, found *cachev1alpha1.HostGroup) error {

	group := hgr.Group()
	selector := types.NamespacedName{Name: group.Name(), Namespace: "openshift-monitoring"}
	err := client.Get(ctx.TODO(), selector, found)

	return err

}

func updateHostGroup(update *cachev1alpha1.HostGroup) error {

	err := client.Update(ctx.TODO(), update)

	return err

}

func createHostGroup(hostGroup HostGroup) {

	ipAddrList := ipList(hostGroup.Hosts())
	k8sHostGroup := hostGroupForGroupAndHosts(hostGroup.Group(), ipAddrList)

	err := client.Create(ctx.TODO(), k8sHostGroup)
	if err != nil {
		log.Error("Could not create CRD-HostGroupInstance for AWX-Group: \"" + hostGroup.Group().Name() + "\"")
	}
}

// serviceForMemcached returns a memcached Service object
func hostGroupForGroupAndHosts(group *eawx.Group, hosts []string) *cachev1alpha1.HostGroup {

	ls := LabelsForHostGroup((*group).Name())

	var tmpEndpoints []cachev1alpha1.Endpoint

	for _, endpoint := range group.Vars().Endpoints() {
		tmpEndpoints = append(tmpEndpoints, cachev1alpha1.Endpoint{
			Endpoint:        endpoint.Endpoint(),
			BearerTokenFile: endpoint.BearerTokenFile(),
			Port:            endpoint.Port(),
			PortName:        endpoint.PortName(),
			Protocol:        endpoint.Protocol(),
			Scheme:          endpoint.Scheme(),
			TargetPort:      endpoint.TargetPort(),
			HonorLabels:     endpoint.HonorLabels(),
			Interval:        endpoint.Interval(),
			ScrapeTimeout:   endpoint.ScrapeTimeout(),
			TLSConf: cachev1alpha1.TLSConfig{
				CAFile:             endpoint.TLSConfig().CAFile(),
				Hostname:           endpoint.TLSConfig().Hostname(),
				InsecureSkipVerify: endpoint.TLSConfig().InsecureSkipVerify(),
			},
		})
	}

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
					Type:      group.Vars().MType(),
					Endpoints: tmpEndpoints,
				},
			},
			Endpoints: hosts,
		},
	}
	return hg
}

func deleteHostGroup(toDelete *cachev1alpha1.HostGroup) error {

	err := client.Delete(ctx.TODO(), toDelete)

	return err

}

func LabelsForHostGroup(name string) map[string]string {
	return map[string]string{"k8s-app": name, "operator-managed": "true"}
}

func ipList(hosts []*eawx.Host) []string {
	var list []string
	for _, host := range hosts {

		ipAddr := determineIPAddress(host)

		if ipAddr != "" {
			list = append(list, ipAddr)
		}
	}
	return list
}

func determineIPAddress(host *eawx.Host) string {
	if host.IP() != "" {
		return host.IP()
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

	// Get the first ipv6 address if no ipv4 address could be found
	for _, a := range addrs {
		if strings.Count(a, ":") >= 2 {
			return a
		}
	}

	log.Error("Looking up ip adress of host \"" + host.Name() + "\" failed! Please check system logs")
	return ""
}
