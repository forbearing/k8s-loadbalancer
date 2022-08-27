package controller

import (
	"fmt"
	"time"

	"github.com/forbearing/k8s-loadbalancer/pkg/nginx"
	"github.com/forbearing/k8s/service"
	"github.com/forbearing/k8s/util/annotations"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
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

type Controller struct {
	serviceHandler *service.Handler
	serviceLister  corelisters.ServiceLister
	serviceSynced  cache.InformerSynced
	workqueue      workqueue.RateLimitingInterface
	recorder       record.EventRecorder
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
		AddFunc: controller.enqueueService,
		UpdateFunc: func(old, new interface{}) {
			newSvc := new.(*corev1.Service)
			oldSvc := old.(*corev1.Service)
			if newSvc.ResourceVersion == oldSvc.ResourceVersion {
				return
			}
			controller.enqueueService(new)
		},
		DeleteFunc: controller.enqueueService,
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
	for i := 0; i < workers; i++ {
		go wait.Until(c.runWorkers, time.Second, stopCh)
	}

	logrus.Info("Started workers")
	<-stopCh
	logrus.Info("Shutting down workers")

	return nil
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
		if err := c.processNginx(key); err != nil {
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
func (c *Controller) processNginx(key string) error {
	// convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	n := &nginx.Nginx{}
	var svcObj = &corev1.Service{}
	// if k8s service object exist, add nginx config.
	if svcObj, err = c.serviceHandler.WithNamespace(namespace).Get(name); err == nil {
		n.Do(nginx.ActionAdd, nginx.ProtocolTCP, namespace, name, svcObj.Spec.Ports)
		return n.Err()
	}
	// if k8s service object not exist, delete nginx config.
	if k8serrors.IsNotFound(err) {
		for n.Do(nginx.ActionDel, nginx.ProtocolTCP, namespace, name, svcObj.Spec.Ports) {
		}
		return n.Err()
	}

	utilruntime.HandleError(fmt.Errorf("service handler get service error: %s", err.Error()))
	return err
}

// enqueueService
func (c *Controller) enqueueService(obj interface{}) {
	namespacedName, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		utilruntime.HandleError(err)
		return
	}

	serviceType, err := c.serviceHandler.GetType(obj)
	if err != nil {
		logrus.Errorf("service handler get service type error: %s", err.Error())
		return
	}

	// if k8s service type is not LoadBalancer, skip it.
	if serviceType != string(corev1.ServiceTypeLoadBalancer) {
		logrus.Debugf(`service "%s" type is not LoadBalancer(is "%s"), skip enqueue`, namespacedName, serviceType)
		return
	}
	// if the k8s object not contains the annotation, skip it.
	if !annotations.Has(obj.(runtime.Object), LoadBalancerAnnotation) {
		logrus.Debugf(`service "%s" don't have annotation: "%s", skip enqueue`, namespacedName, LoadBalancerAnnotation)
		return
	}

	logrus.Debugf("enqueue service: '%s'", namespacedName)
	c.workqueue.Add(namespacedName)
}
