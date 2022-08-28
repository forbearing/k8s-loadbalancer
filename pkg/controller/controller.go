package controller

import (
	"fmt"
	"reflect"
	"time"

	"github.com/forbearing/k8s-loadbalancer/pkg/nginx"
	"github.com/forbearing/k8s/service"
	"github.com/forbearing/k8s/util/annotations"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
)

const (
	LoadBalancerAnnotation = "k8s-loadbalancer=true"
)

type QueueType string

const (
	QueueTypeAdd    QueueType = "Add"
	QueueTypeDelete QueueType = "Delete"
)

type Controller struct {
	serviceHandler *service.Handler
	serviceLister  corelisters.ServiceLister
	serviceSynced  cache.InformerSynced

	addQueue    workqueue.RateLimitingInterface
	deleteQueue workqueue.RateLimitingInterface

	workqueue workqueue.RateLimitingInterface
	recorder  record.EventRecorder
}

func NewController(serviceHandler *service.Handler) *Controller {
	controller := &Controller{
		serviceHandler: serviceHandler,
		serviceLister:  serviceHandler.Lister(),
		serviceSynced:  serviceHandler.Informer().HasSynced,
		addQueue:       workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "loadbalancer"),
		deleteQueue:    workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "loadbalancer"),
	}

	logrus.Info("Setting up event handlers")
	serviceHandler.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    controller.addService,
		UpdateFunc: controller.updateService,
		DeleteFunc: controller.deleteService,
	})

	return controller
}

// Run
func (c *Controller) Run(workers int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()
	defer c.addQueue.ShutDown()
	defer c.deleteQueue.ShutDown()

	logrus.Info("Starting loadbalancer controller")

	logrus.Info("Waiting for informe cache to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.serviceSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	logrus.Info("Starting workers")
	for i := 0; i < workers/2; i++ {
		go wait.Until(c.processAddQueue, time.Second, stopCh)
	}
	for i := 0; i < workers/2; i++ {
		go wait.Until(c.processDeleteQueue, time.Second, stopCh)
	}

	logrus.Info("Started workers")
	<-stopCh
	logrus.Info("Shutting down workers")

	return nil
}

// processAddQueue()
func (c *Controller) processAddQueue() {
	for c.processNextItem(QueueTypeAdd) {
	}
}

// processDeleteQueue
func (c *Controller) processDeleteQueue() {
	for c.processNextItem(QueueTypeDelete) {
	}
}

// processNextItem
func (c *Controller) processNextItem(queuetype QueueType) bool {
	switch queuetype {
	case QueueTypeAdd:
		obj, shutdown := c.addQueue.Get()
		if shutdown {
			return false
		}

		err := func(obj interface{}) error {
			defer c.addQueue.Done(obj)
			if err := c.processNginx(obj.(string), QueueTypeAdd); err != nil {
				return fmt.Errorf(`error syncing "%s": %s, requeuing`, obj, err.Error())
			}
			c.addQueue.Forget(obj)
			logrus.Infof(`Successfully synced "%s"`, obj)
			return nil
		}(obj)

		if err != nil {
			utilruntime.HandleError(err)
			return true
		}
		return true
	case QueueTypeDelete:
		obj, shutdown := c.deleteQueue.Get()
		if shutdown {
			return false
		}

		err := func(obj interface{}) error {
			defer c.deleteQueue.Done(obj)
			if err := c.processNginx(obj.(string), QueueTypeDelete); err != nil {
				return fmt.Errorf(`error syncing "%s": %s, requeuing`, obj, err.Error())
			}
			c.deleteQueue.Forget(obj)
			logrus.Infof(`Successfully synced "%s"`, obj)
			return nil
		}(obj)

		if err != nil {
			utilruntime.HandleError(err)
			return true
		}
		return true
	default:
		return false
	}
}

// runWorkers
func (c *Controller) runWorkers() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem
func (c *Controller) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()
	if shutdown {
		return false
	}

	err := func(obj interface{}) error {
		defer c.workqueue.Done(obj)
		var key string
		var ok bool
		if key, ok = obj.(string); !ok {
			c.workqueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
		}
		if err := c.processNginx(key, ""); err != nil {
			c.workqueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}
		c.workqueue.Forget(obj)
		logrus.Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

// processNginx
func (c *Controller) processNginx(key string, queuetype QueueType) error {
	n := &nginx.Nginx{}
	switch QueueType {
	case QueueTypeAdd:
		//for n.Do()
	}
	return nil
}

// addService
func (c *Controller) addService(obj interface{}) {
	// determine whether the service object is LoadBalancer type and have specified annotation.
	// if not meet the condition, skip enqueue.
	if !c.filterService(obj) {
		return
	}

	// enqueue ServiceInfo object containing the nginx configuration filename we shoud create
	serviceInfo := c.constructServiceInfo(obj)
	serviceInfo.Action = nginx.ActionTypeAdd
	c.addQueue.Add(serviceInfo)
}

// updateService
func (c *Controller) updateService(oldObj, newObj interface{}) {
	// two different version of the same service object always have different ResourceVersion.
	// if ResourceVersion is the same, skip enqueue.
	oldSvc := oldObj.(*corev1.Service)
	newSvc := newObj.(*corev1.Service)
	if oldSvc.ResourceVersion == newSvc.ResourceVersion {
		return
	}

	oldServiceInfo := c.constructServiceInfo(oldObj)
	newServiceInfo := c.constructServiceInfo(newObj)
	// if old ServiceInfo deep equal to new serviceInfo, skip enqueue
	if reflect.DeepEqual(oldServiceInfo, newServiceInfo) {
		return
	}
	// enqueue ServiceInfo object containing the nginx configuration filename we should remove
	oldServiceInfo.Action = nginx.ActionTypeDelete
	c.deleteQueue.Add(oldServiceInfo)

	// determine whether the new service object is LoadBalancer type and have specified annotation.
	// if not meet the condition, skip enqueue.
	if !c.filterService(newObj) {
		return
	}
	// enqueue ServiceInfo object containing the nginx configuration filename we should create
	newServiceInfo.Action = nginx.ActionTypeAdd
	c.addQueue.Add(newServiceInfo)
}

// deleteService
func (c *Controller) deleteService(obj interface{}) {
	// enqueue items containing the nginx configuration filename we should remove
	serviceInfo := c.constructServiceInfo(obj)
	serviceInfo.Action = nginx.ActionTypeDelete
	c.deleteQueue.Add(serviceInfo)
}

// filterService will check the k8s service object meet the condition to enqueue.
func (c *Controller) filterService(obj interface{}) bool {
	// if the k8s object don't have metadata, return false
	namespacedName, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		logrus.Errorf("the k8s object has no metadata: %s", err.Error())
		return false
	}

	serviceType, err := c.serviceHandler.GetType(obj)
	if err != nil {
		logrus.Errorf("service handler get service type error: %s", err.Error())
		return false
	}
	// if the k8s service type is not LoadBalancer, return false.
	if serviceType != string(corev1.ServiceTypeLoadBalancer) {
		logrus.Debugf(`service "%s" type is "%s", skip enqueue`, namespacedName, serviceType)
		return false
	}
	// if the k8s service don't contains the annotation, return false
	if !annotations.Has(obj.(runtime.Object), LoadBalancerAnnotation) {
		logrus.Debugf(`service "%s" don't have annotation: "%s", skip enqueue`, namespacedName, LoadBalancerAnnotation)
		return false
	}
	return true
}

// constructServiceInfo
func (c *Controller) constructServiceInfo(obj interface{}) *nginx.ServiceInfo {
	namespace := obj.(metav1.Object).GetNamespace()
	name := obj.(metav1.Object).GetName()

	//var svcObj = &corev1.Service{}
	svcObj, err := c.serviceLister.Services(namespace).Get(name)
	if err != nil {
		// the k8s service resourcemy no longer exist, in which case we stop processing
		if k8serrors.IsNotFound(err) {
			logrus.Errorf(`service "%s/%s" in listers not longer exists`, namespace, name)
			return nil
		}
		logrus.Errorf("get service resource from listers error: %s", err.Error())
		return nil
	}

	var serviceInfo = &nginx.ServiceInfo{
		Namespace: svcObj.Namespace,
		Name:      svcObj.Name,
	}

	var portsInfo []nginx.PortInfo
	for _, port := range svcObj.Spec.Ports {
		portInfo := nginx.PortInfo{
			Protocol: string(port.Protocol),
			Name:     port.Name,
		}
		portsInfo = append(portsInfo, portInfo)
	}
	serviceInfo.PortsInfo = portsInfo
	return serviceInfo
}
