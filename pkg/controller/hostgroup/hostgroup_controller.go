package hostgroup

import (
	"context"
	"strconv"
	"strings"

	"github.com/spf13/viper"

	promonv1 "github.com/lz006/monitor-operator/pkg/apis/cache/v1"
	cachev1alpha1 "github.com/lz006/monitor-operator/pkg/apis/cache/v1alpha1"
	"github.com/lz006/monitor-operator/pkg/crdmgr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
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
	// Define a new Service object
	service := r.serviceForHostGroup(instance)
	// Define a new ServiceMonitor object
	serviceMonitor := r.serviceMonitorForHostGroup(instance)

	// Set HostGroup instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, endpoints, r.scheme); err != nil {
		return reconcile.Result{}, err
	}
	if err := controllerutil.SetControllerReference(instance, service, r.scheme); err != nil {
		return reconcile.Result{}, err
	}
	if err := controllerutil.SetControllerReference(instance, serviceMonitor, r.scheme); err != nil {
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

		// Endpoints created successfully
		return reconcile.Result{Requeue: true}, nil
	} else if err != nil {

		reqLogger.Error(err, "Failed to get Endpoints: "+instance.Name)
		return reconcile.Result{}, err
	}
	// Check if this Service already exists
	foundService := &corev1.Service{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: service.Name, Namespace: service.Namespace}, foundService)
	// Create Service object if does not exists yet
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new Service", "Service.Namespace", service.Namespace, "Service.Name", service.Name)
		err = r.client.Create(context.TODO(), service)
		if err != nil {

			reqLogger.Error(err, "Failed to create Service: "+instance.Name)
			return reconcile.Result{}, err
		}

		// Service created successfully - don't requeue
		return reconcile.Result{Requeue: true}, nil
	} else if err != nil {

		reqLogger.Error(err, "Failed to get Service: "+instance.Name)
		return reconcile.Result{}, err
	}
	// Check if this ServiceMonitor already exists
	foundServiceMonitor := &promonv1.ServiceMonitor{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: serviceMonitor.Name, Namespace: serviceMonitor.Namespace}, foundServiceMonitor)
	// Create ServiceMonitor object if does not exists yet
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new ServiceMonitor", "ServiceMonitor.Namespace", serviceMonitor.Namespace, "ServiceMonitor.Name", serviceMonitor.Name)
		err = r.client.Create(context.TODO(), serviceMonitor)
		if err != nil {

			reqLogger.Error(err, "Failed to create ServiceMonitor: "+instance.Name)
			return reconcile.Result{}, err
		}

		// ServiceMonitor created successfully - don't requeue
		return reconcile.Result{Requeue: true}, nil
	} else if err != nil {

		reqLogger.Error(err, "Failed to get ServiceMonitor: "+instance.Name)
		return reconcile.Result{}, err
	}

	// Check if Endpoints object differs from current HostGroup configuration
	if !isEndpointsEqualTo(endpoints, found) && found.GetLabels()[viper.GetString("k8s_label_operator_indicator")] == "yes" {
		found.Subsets = endpoints.Subsets
		err = r.client.Update(context.TODO(), found)
		if err != nil {

			reqLogger.Error(err, "Failed to update Endpoints: "+instance.Name)
			return reconcile.Result{}, err
		}

		reqLogger.Info("Endpoints updates successfully: " + instance.Name)
		return reconcile.Result{Requeue: true}, nil
	}

	// Check if Service object differs from current HostGroup configuration
	if !isServiceEqualTo(service, foundService) && found.GetLabels()[viper.GetString("k8s_label_operator_indicator")] == "yes" {
		foundService.Spec.Ports = service.Spec.Ports
		err = r.client.Update(context.TODO(), foundService)
		if err != nil {

			reqLogger.Error(err, "Failed to update Service: "+instance.Name)
			return reconcile.Result{}, err
		}

		reqLogger.Info("Service updates successfully: " + instance.Name)
		return reconcile.Result{Requeue: true}, nil
	}

	// Check if ServiceMonitor object differs from current HostGroup configuration
	if !isServiceMonitorEqualTo(serviceMonitor, foundServiceMonitor) && found.GetLabels()[viper.GetString("k8s_label_operator_indicator")] == "yes" {
		foundServiceMonitor.Spec.Endpoints = serviceMonitor.Spec.Endpoints
		err = r.client.Update(context.TODO(), foundServiceMonitor)
		if err != nil {

			reqLogger.Error(err, "Failed to update ServiceMonitor: "+instance.Name)
			return reconcile.Result{}, err
		}

		reqLogger.Info("ServiceMonitor updates successfully: " + instance.Name)
		return reconcile.Result{Requeue: true}, nil
	}

	return reconcile.Result{}, nil
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
		case "UDP":
			protocol = corev1.ProtocolUDP
		default:
			protocol = corev1.ProtocolTCP
		}

		result = append(result, corev1.EndpointPort{
			Name:     endpoint.PortName,
			Port:     endpoint.Port,
			Protocol: protocol,
		})
	}

	return result
}

func (r *ReconcileHostGroup) serviceForHostGroup(hgr *cachev1alpha1.HostGroup) *corev1.Service {
	ls := crdmgr.LabelsForHostGroup(hgr.Name)
	servicePorts := servicePortsForHostGroup(hgr)

	svc := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      hgr.Name,
			Namespace: hgr.Namespace,
			Labels:    ls,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{},
			Ports:    servicePorts,
		},
	}
	// Set Memcached instance as the owner and controller
	controllerutil.SetControllerReference(hgr, svc, r.scheme)
	return svc
}

func servicePortsForHostGroup(hgr *cachev1alpha1.HostGroup) []corev1.ServicePort {
	var result []corev1.ServicePort

	// Ensure unique combination of Port & Protocol which might could have duplicates with different endpoints (paths)
	uniquePorts := make(map[string]cachev1alpha1.Endpoint)
	for _, ep := range hgr.Spec.HostGroup.Vars.Endpoints {
		uniquePorts[strconv.Itoa(int(ep.Port))+ep.Protocol] = ep
	}

	for _, endpoint := range uniquePorts {

		var protocol corev1.Protocol
		switch tmpProto := strings.ToUpper(endpoint.Protocol); tmpProto {
		case "TCP":
			protocol = corev1.ProtocolTCP
		case "STCP":
			protocol = corev1.ProtocolSCTP
		case "UDP":
			protocol = corev1.ProtocolUDP
		default:
			protocol = corev1.ProtocolTCP
		}

		result = append(result, corev1.ServicePort{
			Name:       endpoint.PortName,
			Port:       endpoint.Port,
			Protocol:   protocol,
			TargetPort: intstr.FromInt(int(endpoint.Port)),
		})
	}

	return result
}

func (r *ReconcileHostGroup) serviceMonitorForHostGroup(hgr *cachev1alpha1.HostGroup) *promonv1.ServiceMonitor {
	ls := crdmgr.LabelsForHostGroup(hgr.Name)
	endpoints := promEndpointForHostGroup(hgr)

	sm := &promonv1.ServiceMonitor{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "monitoring.coreos.com/v1",
			Kind:       "ServiceMonitor",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      hgr.Name,
			Namespace: hgr.Namespace,
			Labels:    ls,
		},
		Spec: promonv1.ServiceMonitorSpec{
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"k8s-app": hgr.Name,
				},
			},
			Endpoints: endpoints,
		},
	}
	// Set HostGroup instance as the owner and controller
	controllerutil.SetControllerReference(hgr, sm, r.scheme)
	return sm
}

func promEndpointForHostGroup(hgr *cachev1alpha1.HostGroup) []promonv1.Endpoint {
	var result []promonv1.Endpoint

	for _, ep := range hgr.Spec.HostGroup.Vars.Endpoints {

		targetPort := intstr.FromInt(ep.TargetPort)

		result = append(result, promonv1.Endpoint{
			Port:          ep.PortName,
			TargetPort:    &targetPort,
			Path:          ep.Endpoint,
			Scheme:        ep.Scheme,
			Interval:      ep.Interval,
			ScrapeTimeout: ep.ScrapeTimeout,
			TLSConfig: &promonv1.TLSConfig{
				CAFile:             ep.TLSConf.CAFile,
				ServerName:         ep.TLSConf.Hostname,
				InsecureSkipVerify: ep.TLSConf.InsecureSkipVerify,
			},
			BearerTokenFile: ep.BearerTokenFile,
			HonorLabels:     ep.HonorLabels,
		})
	}

	return result
}
