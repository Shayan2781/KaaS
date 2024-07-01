package confgs

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"log"
)

var Client *kubernetes.Clientset

func CreateClient() {
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("Failed to create kubernetes client: %v", err)
	}
	Client = kubernetes.NewForConfigOrDie(config)
}
