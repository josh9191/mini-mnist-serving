package clients

import (
	"sync"

	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
)

// KubernetesClient stores kubernetes.Clientset
type KubernetesClient struct {
	clientSet *kubernetes.Clientset
}

var client *KubernetesClient
var once sync.Once

// GetKubernetesClientSet returns singleton instance of KubernetesClient
func GetKubernetesClientSet() *kubernetes.Clientset {
	// Because the client must have been initialized, use panic
	if client == nil {
		panic("The Kubernetes client has not been initialized.")
	}

	return client.clientSet
}

// InitKubernetesClient initialize singleton instance of KubernetesClient
func InitKubernetesClient(kubeconfig string) {
	once.Do(func() {
		config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			panic(err.Error())
		}
		clientSet, err := kubernetes.NewForConfig(config)
		if err != nil {
			panic(err.Error())
		}
		client = &KubernetesClient{clientSet}
	})
}
