package hostgroup

import (
	"reflect"
	"strconv"

	promonv1 "github.com/lz006/monitor-operator/pkg/apis/cache/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func isEndpointsEqualTo(new *corev1.Endpoints, old *corev1.Endpoints) bool {
	result := true

	if len(new.Subsets[0].Addresses) == len(old.Subsets[0].Addresses) &&
		len(new.Subsets[0].Ports) == len(old.Subsets[0].Ports) {

		// Building maps is necessary due not predictible order of elements in arrays
		newAddrMap := make(map[string]corev1.EndpointAddress)
		for _, ae := range new.Subsets[0].Addresses {
			newAddrMap[ae.IP] = ae
		}
		oldAddrMap := make(map[string]corev1.EndpointAddress)
		for _, ae := range old.Subsets[0].Addresses {
			oldAddrMap[ae.IP] = ae
		}

		newPortMap := make(map[string]corev1.EndpointPort)
		for _, ep := range new.Subsets[0].Ports {
			newPortMap[strconv.Itoa(int(ep.Port))+string(ep.Protocol)] = ep
		}

		oldPortMap := make(map[string]corev1.EndpointPort)
		for _, ep := range old.Subsets[0].Ports {
			oldPortMap[strconv.Itoa(int(ep.Port))+string(ep.Protocol)] = ep
		}

		// Begin comparsion
		for key, _ := range newAddrMap {
			if !reflect.DeepEqual(newAddrMap[key].IP, oldAddrMap[key].IP) ||
				!reflect.DeepEqual(newAddrMap[key].Hostname, oldAddrMap[key].Hostname) {
				result = false
			}
		}

		if !reflect.DeepEqual(newPortMap, oldPortMap) {
			result = false
		}

	} else {
		result = false
	}

	return result
}

func isServiceEqualTo(new *corev1.Service, old *corev1.Service) bool {
	result := true

	if len(new.Spec.Ports) == len(old.Spec.Ports) {

		// Building maps is necessary due not predictible order of elements in arrays
		newPortMap := make(map[string]corev1.ServicePort)
		for _, p := range new.Spec.Ports {
			newPortMap[p.Name] = p
		}
		oldPortMap := make(map[string]corev1.ServicePort)
		for _, p := range old.Spec.Ports {
			oldPortMap[p.Name] = p
		}

		// Begin comparsion
		if !reflect.DeepEqual(newPortMap, oldPortMap) {
			result = false
		}

	} else {
		result = false
	}

	return result
}

func isServiceMonitorEqualTo(new *promonv1.ServiceMonitor, old *promonv1.ServiceMonitor) bool {
	result := true

	if len(new.Spec.Endpoints) == len(old.Spec.Endpoints) {

		// Building maps is necessary due not predictible order of elements in arrays
		newEndpointMap := make(map[string]*promonv1.Endpoint)
		for _, ep := range new.Spec.Endpoints {
			newEndpointMap[ep.Port] = &ep
		}
		oldEndpointMap := make(map[string]*promonv1.Endpoint)
		for _, ep := range old.Spec.Endpoints {
			oldEndpointMap[ep.Port] = &ep
		}

		// Begin comparsion
		for key, _ := range newEndpointMap {

			// Keep TLSConfig
			newOriginTLS := newEndpointMap[key].TLSConfig
			oldOriginTLS := oldEndpointMap[key].TLSConfig
			newEndpointMap[key].TLSConfig = &promonv1.TLSConfig{}
			oldEndpointMap[key].TLSConfig = newEndpointMap[key].TLSConfig

			// Keep TargetPort
			newOriginTargetPort := newEndpointMap[key].TargetPort
			oldOriginTargetPort := oldEndpointMap[key].TargetPort
			targetPort := intstr.FromInt(0)
			newEndpointMap[key].TargetPort = &targetPort
			oldEndpointMap[key].TargetPort = newEndpointMap[key].TargetPort

			// Keep BasicAuth
			newOriginBasicAuth := newEndpointMap[key].BasicAuth
			oldOriginBasicAuth := oldEndpointMap[key].BasicAuth
			newEndpointMap[key].BasicAuth = &promonv1.BasicAuth{}
			oldEndpointMap[key].BasicAuth = newEndpointMap[key].BasicAuth

			if !reflect.DeepEqual(newEndpointMap[key], oldEndpointMap[key]) {
				result = false
			}

			// Check TLSConfig
			if newOriginTLS == nil && oldOriginTLS == nil {

			} else {
				if !reflect.DeepEqual(newOriginTLS, oldOriginTLS) {
					result = false
				}
			}
			// Begin restoring values
			// Restore TLSConfig
			newEndpointMap[key].TLSConfig = newOriginTLS
			oldEndpointMap[key].TLSConfig = oldOriginTLS

			// Restore TargetPort
			newEndpointMap[key].TargetPort = newOriginTargetPort
			oldEndpointMap[key].TargetPort = oldOriginTargetPort

			// Restore BasicAuth
			newEndpointMap[key].BasicAuth = newOriginBasicAuth
			oldEndpointMap[key].BasicAuth = oldOriginBasicAuth

		}
	} else {
		result = false
	}

	return result
}
