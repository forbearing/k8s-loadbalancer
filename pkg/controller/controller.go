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
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
)

const (
	LoadBalancerAnnotation = "loadbalancer=enabled"
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

	workqueue workqueue.RateLimitingInterface

	recorder record.EventRecorder
}

func NewController(serviceHandler *service.Handler) *Controller {
	controller := &Controller{
		serviceHandler: serviceHandler,
		serviceLister:  serviceHandler.Lister(),
		serviceSynced:  serviceHandler.Informer().HasSynced,
		workqueue:      workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "loadbalancer"),
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

	logrus.Info("Starting loadbalancer controller")

	logrus.Info("Waiting for informe cache to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.serviceSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	logrus.Info("Starting workers")
	go wait.Until(c.runWorkers, time.Second, stopCh)

	logrus.Info("Started workers")
	<-stopCh
	logrus.Info("Shutting down workers")

	return nil
}

// processAddQueue()
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
	nginxService := obj.(*nginx.Service)
	l := logrus.WithFields(logrus.Fields{
		"action":    nginxService.Action,
		"namespace": nginxService.Namespace,
		"name":      nginxService.Name,
	})

	err := func(obj interface{}) error {
		defer c.workqueue.Done(obj)
		if err := c.processNginx(obj); err != nil {
			//return fmt.Errorf(`error syncing "%+v": %s, requeuing`, obj, err.Error())
			l.Errorf("Failed to processed nginx config: %s", err.Error())
			return err
		}
		c.workqueue.Forget(obj)
		l.Info("Successfully processed nginx config")
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}
	return true
}

//// runWorkers
//func (c *Controller) runWorkers() {
//    for c.processNextWorkItem() {
//    }
//}

//// processNextWorkItem
//func (c *Controller) processNextWorkItem() bool {
//    obj, shutdown := c.workqueue.Get()
//    if shutdown {
//        return false
//    }

//    err := func(obj interface{}) error {
//        defer c.workqueue.Done(obj)
//        var key string
//        var ok bool
//        if key, ok = obj.(string); !ok {
//            c.workqueue.Forget(obj)
//            utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
//        }
//        if err := c.processNginx(key, ""); err != nil {
//            c.workqueue.AddRateLimited(key)
//            return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
//        }
//        c.workqueue.Forget(obj)
//        logrus.Infof("Successfully synced '%s'", key)
//        return nil
//    }(obj)

//    if err != nil {
//        utilruntime.HandleError(err)
//        return true
//    }

//    return true
//}

// processNginx
func (c *Controller) processNginx(obj interface{}) error {
	nginxService, ok := obj.(*nginx.Service)
	if !ok {
		return fmt.Errorf("object type is not *nginx.Service")
	}
	n := &nginx.Nginx{}
	for n.Do(nginxService) {
	}
	return n.Err()
}

// addService
func (c *Controller) addService(obj interface{}) {
	logger := logrus.WithField("event", "add")
	nginxService := c.constructNginxService(obj)
	// determine whether the service object is LoadBalancer type and have specified annotation.
	// if not meet the condition, skip enqueue.
	if c.isMeetCondition(logger, obj) {
		// enqueue Service object containing the nginx configuration filename we shoud create
		nginxService.Action = nginx.ActionTypeAdd
		c.workqueue.Add(nginxService)
	}
}

// updateService
func (c *Controller) updateService(oldObj, newObj interface{}) {
	logger := logrus.WithField("event", "update")
	// two different version of the same service object always have different ResourceVersion.
	// if ResourceVersion is the same, skip enqueue.
	oldSvc := oldObj.(*corev1.Service)
	newSvc := newObj.(*corev1.Service)
	if oldSvc.ResourceVersion == newSvc.ResourceVersion {
		logger.Debugf("k8s service updated, but ResourceVersion is the same, skip enqueue.")
		return
	}

	oldNginxService := c.constructNginxService(oldObj)
	newNginxService := c.constructNginxService(newObj)

	// if the old nginx.Service deep equal to the new nginx.Service, it's no need to enqueue
	if reflect.DeepEqual(oldNginxService, newNginxService) {
		return
	}

	// you should always enqueue oldNginxService before newNginxService.
	//
	// determine whether the old service object is LoadBalancer type and have specified annotation.
	// if not meet the condition, skip enqueue.
	if c.isMeetCondition(logger.WithField("version", "old"), oldObj) {
		// enqueue Service object containing the nginx configuration filename we should remove
		oldNginxService.Action = nginx.ActionTypeDel
		c.workqueue.Add(oldNginxService)
	}
	// determine whether the new service object is LoadBalancer type and have specified annotation.
	// if not meet the condition, skip enqueue.
	if c.isMeetCondition(logger.WithField("version", "new"), newObj) {
		// enqueue Service object containing the nginx configuration filename we should create
		newNginxService.Action = nginx.ActionTypeAdd
		c.workqueue.Add(newNginxService)
	}

}

// deleteService
func (c *Controller) deleteService(obj interface{}) {
	logger := logrus.WithField("event", "delete")
	nginxService := c.constructNginxService(obj)
	// determine whether the service object is LoadBalancer type and have specified annotation.
	// if not meet the condition, skip enqueue.
	if c.isMeetCondition(logger, obj) {
		// enqueue items containing the nginx configuration filename we should remove
		nginxService.Action = nginx.ActionTypeDel
		c.workqueue.Add(nginxService)
	}
}

// isMeetCondition will check the k8s service object meet the condition to enqueue.
func (c *Controller) isMeetCondition(logger *logrus.Entry, obj interface{}) bool {
	// if the k8s object don't have metadata, return false
	accessor, err := meta.Accessor(obj)
	if err != nil {
		logrus.Errorf("the k8s object has no metadata: %s", err.Error())
		return false
	}

	l := logger.WithFields(logrus.Fields{
		"namespace": accessor.GetNamespace(),
		"name":      accessor.GetName(),
	})

	serviceType, err := c.serviceHandler.GetType(obj)
	if err != nil {
		l.Errorf("service handler get service type error: %s", err.Error())
		return false
	}
	// if the k8s service type is not LoadBalancer, return false.
	if serviceType != string(corev1.ServiceTypeLoadBalancer) {
		l.Debugf(`service type is "%s", skip enqueue`, serviceType)
		return false
	}
	// if the k8s service don't contains the annotation, return false
	if !annotations.Has(obj.(runtime.Object), LoadBalancerAnnotation) {
		l.Debugf(`service don't have annotation: "%s", skip enqueue`, LoadBalancerAnnotation)
		return false
	}

	l.Debugf(`service meet condition, start enqueue`)
	return true
}

// constructNginxService
func (c *Controller) constructNginxService(obj interface{}) *nginx.Service {
	svcObj, ok := obj.(*corev1.Service)
	if !ok {
		logrus.Errorf("the object is not *corev1.Service")
		return nil
	}
	namespace := svcObj.Namespace
	name := svcObj.Name

	var nginxService = &nginx.Service{
		Namespace: namespace,
		Name:      name,
	}

	serviceType, err := c.serviceHandler.GetType(obj)
	if err != nil {
		logrus.Errorf("service handler get service type error: %s", err.Error())
	}
	if serviceType == string(corev1.ServiceTypeLoadBalancer) {
		nginxService.MeetType = true
	}
	if annotations.Has(obj.(runtime.Object), LoadBalancerAnnotation) {
		nginxService.MeetAnnotations = true
	}

	var ports []nginx.ServicePort
	for _, p := range svcObj.Spec.Ports {
		portInfo := nginx.ServicePort{
			Name:     p.Name,
			Port:     p.Port,
			NodePort: p.NodePort,
			Protocol: string(p.Protocol),
		}
		ports = append(ports, portInfo)
	}
	nginxService.Ports = ports
	return nginxService
}
