/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	_ "runtime/debug"
	_ "time"

	"github.com/go-logr/logr"
	// . "github.com/onsi/ginkgo"
	// . "github.com/onsi/gomega"
	// appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	_ "sigs.k8s.io/controller-runtime/pkg/runtime/inject"

	batchv1alpha1 "sdewan.akraino.org/sdewan/api/v1alpha1"
	"sdewan.akraino.org/sdewan/openwrt"
	"sigs.k8s.io/controller-runtime/pkg/source"
	// "sdewan.akraino.org/sdewan/cnfprovider"
)

type NfnInterfaces struct {
	NetName   string      `json:"name,omitempty"`
	Name      string      `json:"interface,omitempty"`
	GatWay    string      `json:"gateway,omitempty"`
	IpAddress interface{} `json:"ipAddress,omitempty"`
}

type NfnNetwork struct {
	Type      string          `json:"type,omitempty"`
	Interface []NfnInterfaces `json:"interface,omitempty"`
}

func Interface2IpAddressArray(addriface interface{}) []string {
	var iparray []string
	switch addriface.(type) {
	case []interface{}:
		for _, aif := range addriface.([]interface{}) {
			iparray = append(iparray, aif.(string))
		}
	case string:
		iparray = append(iparray, addriface.(string))
	default:
		iparray = []string{}
	}
	return iparray
}

// CniInterfaceReconciler reconciles a CniInterface object
type CniInterfaceReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

//Function to create CniInterface
// func createCniInterface(podname, namespace, purpose string, ni NfnInterfaces, labels map[string]string, r CniInterfaceReconciler) error {
func createCniInterfaces(pod corev1.Pod, options ...interface{}) []*batchv1alpha1.CniInterface {
	// cniinterfaceClient := clientset.CoreV1().CniInterfaces(namespace)
	podname := pod.Name
	namespace := pod.Namespace
	labels := pod.Labels
	labels["ownedByPod"] = podname //NOTE (Shaohe), not sure it can change pod lables
	purpose := pod.Labels["sdewanPurpose"]
	nfnnetwork := pod.Annotations["k8s.plugin.opnfv.org/nfn-network"]
	var cniinterfaces []*batchv1alpha1.CniInterface

	get_cni_iface := func(nic string, ips []string) *batchv1alpha1.CniInterface {
		return &batchv1alpha1.CniInterface{

			ObjectMeta: metav1.ObjectMeta{
				Name:      podname + "-cniinterface-" + nic,
				Namespace: namespace,
				Labels:    labels,
			},
			Spec: batchv1alpha1.CniInterfaceSpec{
				OwnerPod:  podname,
				Name:      nic,
				IpAddress: ips,
				// CnfSelector: map[string]string{"sdewanPurpose": purpose},
				CnfSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{"sdewanPurpose": purpose, "ownedByPod": podname},
				},
			},
		}

	}
	// NOTE(Shaohe): if pass the interface info from openwrt, create CNI Interfaces from openwrt
	if len(options) > 0 {
		ifaces, ok := options[0].(*openwrt.AvailableInterfaces)
		// fmt.Printf("fcs %T\n", openwrt.AvailableInterfaces)
		fmt.Println("type openwrt.AvailableInterfaces :", ok, "value: ", ifaces)
		if ok {
			for _, ifc := range ifaces.Interfaces {
				cniinterfaces = append(cniinterfaces, get_cni_iface(ifc.Name, ifc.IpAddress))
			}
			if len(cniinterfaces) > 0 {
				fmt.Println("Generate interfaces object from openwrt Interfaces", cniinterfaces)
				return cniinterfaces
			}
		}
	}

	// NOTE(Shaohe):  create CNI Interfaces from Pod info
	// create nics from Annotations
	var nfn NfnNetwork
	var nis []NfnInterfaces
	if err := json.Unmarshal([]byte(nfnnetwork), &nfn); err != nil {
		// FIXME(shaohe) empty nfn and log for the err
		nis = []NfnInterfaces{}
		fmt.Println(err)
	} else {
		nis = nfn.Interface
	}
	for _, ni := range nis {
		cniinterfaces = append(cniinterfaces, get_cni_iface(ni.Name, Interface2IpAddressArray(ni.IpAddress)))
	}

	// create buildin eth0 nics
	nic := "eth0"
	iparray := []string{}
	for _, ip := range pod.Status.PodIPs {
		iparray = append(iparray, ip.IP)
	}

	cniinterfaces = append(cniinterfaces, get_cni_iface(nic, iparray))

	return cniinterfaces
}

func InterfaceHandler(ievt interface{}, r client.Client) []reconcile.Request {
	var enqueueRequest []reconcile.Request
	ctx := context.Background()
	switch evt := ievt.(type) {
	case event.CreateEvent:
		meta := evt.Meta
		pod, _ := evt.Object.DeepCopyObject().(*corev1.Pod)
		newifaces := createCniInterfaces(*pod)
		fmt.Println("event.CreateEvent========================================", newifaces)
		interfaceList := &batchv1alpha1.CniInterfaceList{}
		err := r.List(ctx, interfaceList, client.MatchingLabels{"ownedByPod": meta.GetName()})
		if err != nil {
			fmt.Println(err, "Failed to get icn interfaceList")
		}
		set := make(map[string]*batchv1alpha1.CniInterface)
		for _, v := range interfaceList.Items {
			set[v.Name] = &v
		}
		for _, v := range newifaces {
			exist := true
			_, ok := set[(*v).Name]
			if !ok {
				if err := r.Create(ctx, v); err != nil {
					fmt.Println(err, "unable to create CniInterface.")
				} else {
					exist = false
				}
			}
			if exist {
				req := reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      v.Name,
						Namespace: meta.GetNamespace(),
					}}
				enqueueRequest = append(enqueueRequest, req)
			}
		}
	case event.UpdateEvent:
		meta := evt.MetaNew
		pod, _ := evt.ObjectNew.DeepCopyObject().(*corev1.Pod)
		metaold := evt.MetaOld
		podold, _ := evt.ObjectOld.DeepCopyObject().(*corev1.Pod)
		fmt.Println("I'm an event.UpdateEvent", meta, pod, metaold, podold)
	case event.DeleteEvent:
		meta := evt.Meta
		pod, _ := evt.Object.DeepCopyObject().(*corev1.Pod)
		fmt.Println("I'm an event.DeleteEvent", meta, pod)
		interfaceList := &batchv1alpha1.CniInterfaceList{}
		err := r.List(ctx, interfaceList, client.MatchingLabels{"ownedByPod": meta.GetName()})
		if err != nil {
			fmt.Println(err, "Failed to get icn interfaceList")
		}
		for _, v := range interfaceList.Items {
			if err := r.Delete(ctx, &v); (err) != nil {
				fmt.Println(err, "unable to delete old icn interface: ", v.Name)
			} else {
				fmt.Println(err, "successful to delete icn interface: ", v.Name)
			}
		}
	case event.GenericEvent:
		meta := evt.Meta
		pod, _ := evt.Object.DeepCopyObject().(*corev1.Pod)
		fmt.Println("I'm an event.GenericFunc", meta, pod)
	default:
		fmt.Printf("Don't know type %T\n", evt)
	}
	return enqueueRequest
}

type EnqueueRequestsHandler struct {
	// Mapper transforms the argument into a slice of keys to be reconciled
	C       client.Client
	Object  runtime.Object
	Handler func(ev interface{}, r client.Client) []reconcile.Request
}

// Create implements EventHandler
func (e *EnqueueRequestsHandler) Create(evt event.CreateEvent, q workqueue.RateLimitingInterface) {
	e.Handler(evt, e.C)
	// TODO should add the reqs to workqueue??
	// e.mapAndEnqueue(q, reqs)
}

// Update implements EventHandler
func (e *EnqueueRequestsHandler) Update(evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	// e.mapAndEnqueue(q, MapObject{Meta: evt.MetaOld, Object: evt.ObjectOld})
	// e.mapAndEnqueue(q, MapObject{Meta: evt.MetaNew, Object: evt.ObjectNew})
	// TODO should add the reqs to workqueue??
	// e.mapAndEnqueue(q, reqs)
}

// Delete implements EventHandler
func (e *EnqueueRequestsHandler) Delete(evt event.DeleteEvent, q workqueue.RateLimitingInterface) {
	e.Handler(evt, e.C)
	// TODO should add the reqs to workqueue??
	// e.mapAndEnqueue(q, reqs)
}

// Generic implements EventHandler
func (e *EnqueueRequestsHandler) Generic(evt event.GenericEvent, q workqueue.RateLimitingInterface) {
	// TODO should add the reqs to workqueue??
	// e.mapAndEnqueue(q, MapObject{Meta: evt.Meta, Object: evt.Object})
}

func (e *EnqueueRequestsHandler) mapAndEnqueue(q workqueue.RateLimitingInterface, reqs []reconcile.Request) {
	for _, req := range reqs {
		q.Add(req)
	}
}

func updateCniInterfaceStatus(crdiface *batchv1alpha1.CniInterface, iface openwrt.Interface) {
	crdiface.Status.Status = iface.Status
	crdiface.Status.ReceivedPackets = iface.ReceivedPackets
	crdiface.Status.SendPackets = iface.SendPackets
	crdiface.Status.MacAddress = iface.MacAddress
	crdiface.Status.IpAddress = iface.IpAddress
}

func (r *CniInterfaceReconciler) GetPodByInterface(ifc batchv1alpha1.CniInterface) (*corev1.Pod, error) {
	key := client.ObjectKey{Namespace: ifc.Namespace, Name: ifc.Labels["ownedByPod"]}
	log := r.Log.WithValues("cniinterface", key)
	ctx := context.Background()
	pod := &corev1.Pod{}
	err := r.Get(ctx, key, pod)
	if err != nil {
		log.Error(err, "unable to fetch pod", key)
	}
	return pod, err
}

func (r *CniInterfaceReconciler) GetInterfacesByPod(pod corev1.Pod, req ctrl.Request) (*batchv1alpha1.CniInterfaceList, error) {
	log := r.Log.WithValues("cniinterface", req.NamespacedName)
	ctx := context.Background()
	interfaceList := &batchv1alpha1.CniInterfaceList{}
	err := r.Client.List(ctx, interfaceList, client.MatchingLabels{"ownedByPod": pod.Name})
	if err != nil {
		log.Error(err, "Failed to get pod list")
	}
	return interfaceList, err
}

// +kubebuilder:rbac:groups=batch.sdewan.akraino.org,resources=cniinterfaces,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch.sdewan.akraino.org,resources=cniinterfaces/status,verbs=get;update;patch

func (r *CniInterfaceReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("cniinterface", req.NamespacedName)
	// debug.PrintStack()

	// your logic here
	var cniInterface batchv1alpha1.CniInterface
	if err := r.Get(ctx, req.NamespacedName, &cniInterface); err != nil {
		// log.Info(fmt.Sprintf("ctx %v", ctx))
		// log.Info(fmt.Sprintf("cniinterface %v", cniInterface))
		log.Error(err, "unable to fetch CniInterface")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	podList := &corev1.PodList{}
	label := cniInterface.Spec.CnfSelector.MatchLabels["sdewanPurpose"]
	// TODO It is better to use Get of List
	// TODO IF not find pod, means the pod is deleted, we should delete ICN interface CR
	err := r.Client.List(ctx, podList, client.MatchingLabels{"sdewanPurpose": label})
	if err != nil {
		log.Error(err, "Failed to get pod list")
		return ctrl.Result{}, err
	}

	fcs := &openwrt.AvailableInterfaces{}
	for _, pod := range podList.Items {
		log.Info(fmt.Sprintf("Annotations nfn-network: %v\n", pod.Annotations["k8s.plugin.opnfv.org/nfn-network"]))
		log.Info(fmt.Sprintf("Annotations networks-status: %v\n", pod.Annotations["k8s.v1.cni.cncf.io/networks-status"]))
		log.Info(fmt.Sprintf("Annotations ovnInterfaces: %v\n", pod.Annotations["k8s.plugin.opnfv.org/ovnInterfaces"]))
		log.Info(fmt.Sprintf("Pod interfaces: %v\n", (*createCniInterfaces(pod)[0]).Spec))
		// for ifc := range pod.Annotations["k8s.plugin.opnfv.org/nfn-network"]["interface"] {
		// 	log.Info(fmt.Sprintf("Annotations interface: %v, ipAddress: %v\n", ifc["interface"], ifc["ipAddress"]))
		// }
		openwrtClient := openwrt.NewOpenwrtClient(pod.Status.PodIP, "root", "")
		ifc := openwrt.InterfaceClient{OpenwrtClient: openwrtClient}
		fcs, err = ifc.GetAvailableInterfaces() // TODO, support get one interface?
		if err != nil {
			log.Error(err, "Failed to get interface list from CNF openwert.")
			return ctrl.Result{}, err
		}
		fmt.Printf("fcs %T\n", fcs)
		// Debug info for the last interface
		log.Info(fmt.Sprintf("Pod interfaces: %v\n", (*createCniInterfaces(pod, fcs)[2]).Spec))
	}

	for _, fc := range fcs.Interfaces {
		if cniInterface.Spec.Name == fc.Name {
			updateCniInterfaceStatus(&cniInterface, fc)
			break
		}
	}
	// NOTE: if we Update ICN interface CR, the next reconcile will trigger automatically.
	if err := r.Update(ctx, &cniInterface); err != nil {
		log.Error(err, "unable to update CniInterface status")
		return ctrl.Result{}, err
	}
	// time.Sleep(5 * time.Second)

	return ctrl.Result{}, nil
}

// List the needed CR to specific events and return the reconcile Requests
// This fucntion is used for EnqueueRequestsFromMapFunc
// Usage: https://github.com/akraino-edge-stack/icn-sdwan/blob/3896bf7882cfac12960a893a7593b072211cb3b9/platform/crd-ctrlr/src/controllers/mwan3policy_controller.go
// .Watches(
// 	&source.Kind{Type: &appsv1.Deployment{}},
// 	&handler.EnqueueRequestsFromMapFunc{
// 		ToRequests: handler.ToRequestsFunc(GetToRequestsFunc(r, &batchv1alpha1.Mwan3PolicyList{})),
// 	},
// 	Filter).
func GetToRequestsFunc(r client.Client, crliststruct runtime.Object) func(h handler.MapObject) []reconcile.Request {

	return func(h handler.MapObject) []reconcile.Request {
		var enqueueRequest []reconcile.Request
		cnfName := h.Meta.GetLabels()["sdewanPurpose"]
		ctx := context.Background()
		obj := h.Object.DeepCopyObject()
		pod, _ := obj.(*corev1.Pod)
		newifaces := createCniInterfaces(*pod)
		fmt.Println("========================================", newifaces)
		interfaceList := &batchv1alpha1.CniInterfaceList{}
		err := r.List(ctx, interfaceList, client.MatchingLabels{"ownedByPod": h.Meta.GetName()})
		if err != nil {
			fmt.Println(err, "Failed to get icn interfaceList")
		}
		fmt.Println("========================================", h.Meta.GetOwnerReferences())
		// v := reflect.ValueOf(h).Elem()
		// t := v.Type()
		// for i := 0; i < t.NumField(); i++ {
		//     fmt.Println(reflect.TypeOf(h).Elem().Field(i).Name)
		// }
		// ------- Used for debug
		v := reflect.ValueOf(h.Meta).Elem()
		t := v.Type()
		for i := 0; i < t.NumField(); i++ {
			fmt.Println(reflect.TypeOf(h.Meta).Elem().Field(i).Name)
		}
		v = reflect.ValueOf(h.Object).Elem()
		t = v.Type()
		for i := 0; i < t.NumField(); i++ {
			fmt.Println(reflect.TypeOf(h.Object).Elem().Field(i).Name)
		}
		// ------ End for debug
		r.List(ctx, crliststruct, client.MatchingLabels{"sdewanPurpose": cnfName})
		value := reflect.ValueOf(crliststruct)
		items := reflect.Indirect(value).FieldByName("Items")
		for i := 0; i < items.Len(); i++ {
			meta := items.Index(i).Field(1).Interface().(metav1.ObjectMeta)
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      meta.GetName(),
					Namespace: meta.GetNamespace(),
				}}
			enqueueRequest = append(enqueueRequest, req)

		}
		return enqueueRequest
	}
}

var Filter = builder.WithPredicates(predicate.Funcs{
	CreateFunc: func(e event.CreateEvent) bool {
		fmt.Println("------------------------------- call back CreateFunc:")
		if _, ok := e.Meta.GetLabels()["sdewanPurpose"]; !ok {
			return false
		}
		return true
	},
	UpdateFunc: func(e event.UpdateEvent) bool {
		fmt.Println("------------------------------- call back UpdateFunc:")
		if _, ok := e.MetaOld.GetLabels()["sdewanPurpose"]; !ok {
			return false
		}
		// pre_status := reflect.ValueOf(e.ObjectOld).Interface().(*corev1.Pod).Status
		// fmt.Println("------------------------------- call back UpdateFunc:", pre_status)
		// post_status := reflect.ValueOf(e.ObjectNew).Interface().(*corev1.Pod).Status
		// fmt.Println("------------------------------- call back UpdateFunc:", post_status)
		return true
	},
	DeleteFunc: func(e event.DeleteEvent) bool {
		fmt.Println("------------------------------- call back DeleteFunc:")
		if _, ok := e.Meta.GetLabels()["sdewanPurpose"]; !ok {
			return false
		}
		return true
	},
	GenericFunc: func(e event.GenericEvent) bool {
		fmt.Println("------------------------------- call back GenericFunc:")
		if _, ok := e.Meta.GetLabels()["sdewanPurpose"]; !ok {
			return false
		}
		return true
	},
})

// NOTE, If add "sdewanPurpose" lable after Pod cnf created, will not create ICN interface CR automatically.
// We ignore this corner case at present.
func (r *CniInterfaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	ps := builder.WithPredicates(predicate.GenerationChangedPredicate{})
	return ctrl.NewControllerManagedBy(mgr).
		For(&batchv1alpha1.CniInterface{}, ps).
		Watches(
			&source.Kind{Type: &corev1.Pod{}},
			&EnqueueRequestsHandler{C: r, Object: &batchv1alpha1.CniInterfaceList{}, Handler: InterfaceHandler},
			Filter).
		Complete(r)
}
