package hostgroup

import (
	"context"
	"reflect"
	"strconv"
	"strings"

	promonv1 "github.com/lz006/monitor-operator/pkg/apis/cache/v1"
	cachev1alpha1 "github.com/lz006/monitor-operator/pkg/apis/cache/v1alpha1"
	"github.com/lz006/monitor-operator/pkg/crdmgr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_hostgroup")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new HostGroup Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileHostGroup{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("hostgroup-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource HostGroup
	err = c.Watch(&source.Kind{Type: &cachev1alpha1.HostGroup{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to resource Endpoints
	err = c.Watch(&source.Kind{Type: &corev1.Endpoints{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &cachev1alpha1.HostGroup{},
	})
	if err != nil {
		return err
	}

	// Watch for changes to resource Service
	err = c.Watch(&source.Kind{Type: &corev1.Service{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &cachev1alpha1.HostGroup{},
	})
	if err != nil {
		return err
	}
	// Watch for changes to resource ServiceMonitor
	err = c.Watch(&source.Kind{Type: &promonv1.ServiceMonitor{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &cachev1alpha1.HostGroup{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileHostGroup{}

// ReconcileHostGroup reconciles a HostGroup object
type ReconcileHostGroup struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a HostGroup object and makes changes based on the state read
// and what is in the HostGroup.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileHostGroup) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling HostGroup")

	// Fetch the HostGroup instance
	instance := &cachev1alpha1.HostGroup{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Define a new Endpoints object
	endpoints := r.endpointsForHostGroup(instance)
	subsets := endpoints.Subsets[0].Ports[0]
	reqLogger.Info(subsets.Name)

	// Set HostGroup instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, endpoints, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this Endpoints already exists
	found := &corev1.Endpoints{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: endpoints.Name, Namespace: endpoints.Namespace}, found)
	// Create Endpoints objects if does not exists yet
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new Endpoints", "Endpoints.Namespace", endpoints.Namespace, "Endpoints.Name", endpoints.Name)
		err = r.client.Create(context.TODO(), endpoints)
		if err != nil {

			reqLogger.Error(err, "Failed to create Endpoints: "+instance.Name)
			return reconcile.Result{}, err
		}

		// Endpoints created successfully - don't requeue
		return reconcile.Result{}, nil
	} else if err != nil {

		reqLogger.Error(err, "Failed to get Endpoints: "+instance.Name)
		return reconcile.Result{}, err
	}

	// Check if Endpoints object differs from current HostGroup configuration
	if !isEndpointsEqualTo(endpoints, found) && found.GetLabels()["operator-managed"] == "true" {
		found.Subsets = endpoints.Subsets
		err = r.client.Update(context.TODO(), found)
		if err != nil {

			reqLogger.Error(err, "Failed to update Endpoints: "+instance.Name)
			return reconcile.Result{}, err
		}

		reqLogger.Info("Endpoints updates successfully: " + instance.Name)
		return reconcile.Result{}, nil
	}

	return reconcile.Result{}, nil
}

func isEndpointsEqualTo(new *corev1.Endpoints, old *corev1.Endpoints) bool {
	result := true

	if len(new.Subsets[0].Addresses) == len(old.Subsets[0].Addresses) &&
		len(new.Subsets[0].Ports) == len(old.Subsets[0].Ports) {

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
			newPortMap[strconv.Itoa(int(ep.Port))] = ep
		}

		oldPortMap := make(map[string]corev1.EndpointPort)
		for _, ep := range old.Subsets[0].Ports {
			oldPortMap[strconv.Itoa(int(ep.Port))] = ep
		}

		// Begin comparsion
		for _, ae := range newAddrMap {
			if !reflect.DeepEqual(newAddrMap[ae.IP].IP, oldAddrMap[ae.IP].IP) ||
				!reflect.DeepEqual(newAddrMap[ae.IP].Hostname, oldAddrMap[ae.IP].Hostname) {
				result = false
			}
		}

		for key, _ := range newPortMap {
			if !reflect.DeepEqual(newPortMap[key], oldPortMap[key]) {
				result = false
			}
		}

	} else {
		result = false
	}

	return result
}

func (r *ReconcileHostGroup) endpointsForHostGroup(hgr *cachev1alpha1.HostGroup) *corev1.Endpoints {
	ls := crdmgr.LabelsForHostGroup(hgr.Name)

	subsets := endpointSubsetForHostGroup(hgr)

	eps := &corev1.Endpoints{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Endpoints",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      hgr.Name,
			Namespace: hgr.Namespace,
			Labels:    ls,
		},
		Subsets: subsets,
	}
	// Set HostGroup instance as the owner and controller
	controllerutil.SetControllerReference(hgr, eps, r.scheme)
	return eps
}

func endpointSubsetForHostGroup(hgr *cachev1alpha1.HostGroup) []corev1.EndpointSubset {
	endpointAdresses := endpointAddressesForHostGroup(hgr)
	endpointPorts := endpointPortsForHostGroup(hgr)

	return []corev1.EndpointSubset{corev1.EndpointSubset{
		Addresses: endpointAdresses,
		Ports:     endpointPorts,
	}}
}

func endpointAddressesForHostGroup(hgr *cachev1alpha1.HostGroup) []corev1.EndpointAddress {
	var result []corev1.EndpointAddress

	for _, ip := range hgr.Spec.Endpoints {
		result = append(result, corev1.EndpointAddress{
			IP: ip,
		})
	}

	return result
}

func endpointPortsForHostGroup(hgr *cachev1alpha1.HostGroup) []corev1.EndpointPort {
	var result []corev1.EndpointPort

	for _, endpoint := range hgr.Spec.HostGroup.Vars.Endpoints {

		var protocol corev1.Protocol
		switch tmpProto := strings.ToUpper(endpoint.Protocol); tmpProto {
		case "TCP":
			protocol = corev1.ProtocolTCP
		case "STCP":
			protocol = corev1.ProtocolSCTP
		case "UDO":
			protocol = corev1.ProtocolUDP
		default:

		}

		result = append(result, corev1.EndpointPort{
			Name:     endpoint.PortName,
			Port:     endpoint.Port,
			Protocol: protocol,
		})
	}

	return result
}
