package controllers

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"

	// Blank import necessary for kubeConfig to work.
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/watch"
)

const kubeSystemNs = "kube-system"
const kubeProxyPodPrefix = "kube-proxy"

// K8sMetadataController listens to any metadata updates from the K8s API.
type K8sMetadataController struct {
	mdHandler *MetadataHandler
	clientset *kubernetes.Clientset
	quitCh    chan bool
}

// NewK8sMetadataController creates a new K8sMetadataController.
func NewK8sMetadataController(mdh *MetadataHandler) (*K8sMetadataController, error) {
	// There is a specific config for services running in the cluster.
	kubeConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	// Create k8s client.
	clientset, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, err
	}

	mc := &K8sMetadataController{mdHandler: mdh, clientset: clientset}

	// Clean up current metadata store state.
	namespaces, err := mc.listObject("namespaces")
	if err != nil {
		log.Info("Could not list all namespaces")
	}
	nRv := mdh.SyncNamespaceData(runtimeObjToNamespaceList(namespaces))

	pods, err := mc.listObject("pods")
	if err != nil {
		log.Info("Could not list all pods")
	}
	pRv := mdh.SyncPodData(runtimeObjToPodList(pods))

	eps, err := mc.listObject("endpoints")
	if err != nil {
		log.Info("Could not list all endpoints")
	}
	eRv := mdh.SyncEndpointsData(runtimeObjToEndpointsList(eps))

	services, err := mc.listObject("services")
	if err != nil {
		log.Info("Could not list all services")
	}
	sRv := mdh.SyncServiceData(runtimeObjToServiceList(services))

	nodes, err := mc.listObject("nodes")
	if err != nil {
		log.Info("Could not list all nodes")
	}
	nodeRv := mdh.SyncNodeData(runtimeObjToNodeList(nodes))

	// Start up Watchers.
	go mc.startWatcher("namespaces", nRv)
	go mc.startWatcher("pods", pRv)
	go mc.startWatcher("endpoints", eRv)
	go mc.startWatcher("services", sRv)
	go mc.startWatcher("nodes", nodeRv)

	return mc, nil
}

func (mc *K8sMetadataController) listObject(resource string) (runtime.Object, error) {
	watcher := cache.NewListWatchFromClient(mc.clientset.CoreV1().RESTClient(), resource, v1.NamespaceAll, fields.Everything())
	opts := metav1.ListOptions{}
	return watcher.List(opts)
}

func (mc *K8sMetadataController) startWatcher(resource string, resourceVersion int) {
	// Start up watcher for the given resource.
	for {
		watcher := cache.NewListWatchFromClient(mc.clientset.CoreV1().RESTClient(), resource, v1.NamespaceAll, fields.Everything())
		retryWatcher, err := watch.NewRetryWatcher(fmt.Sprintf("%d", resourceVersion), watcher)
		if err != nil {
			log.WithError(err).Fatal("Could not start watcher for k8s resource: " + resource)
		}

		resCh := retryWatcher.ResultChan()

		for {
			select {
			case <-mc.quitCh:
				return
			case c := <-resCh:
				s, ok := c.Object.(*metav1.Status)
				if ok && s.Status == metav1.StatusFailure {
					// Ignore and let the retry watcher retry.
					log.WithField("resource", resource).WithField("object", c.Object).Error("Failed to read from k8s watcher")
					continue
				}

				msg := &K8sMessage{
					Object:     c.Object,
					ObjectType: resource,
					EventType:  c.Type,
				}
				mc.mdHandler.GetChannel() <- msg
			}
		}

		log.WithField("resource", resource).Info("k8s watcher channel closed. retrying")

		select {
		case <-mc.quitCh:
			return
		case <-time.After(time.Second):
			continue
		}
	}
}

// Stop stops all K8s watchers.
func (mc *K8sMetadataController) Stop() {
	mc.quitCh <- true
}
