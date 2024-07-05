package configs

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"log"
)

var Client *kubernetes.Clientset

func CreateClient() {
	//var kubeConfig *string
	//home := homedir.HomeDir()
	//kubeConfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "")
	//flag.Parse()
	//access from outside the cluster
	//config, err := clientcmd.BuildConfigFromFlags("", *kubeConfig)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//access from inside the cluster
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatal(err)
	}
	Client, err = kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}
}