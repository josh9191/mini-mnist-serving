package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/josh9191/mini-mnist-serving/clients"
	"github.com/josh9191/mini-mnist-serving/controller"
	"k8s.io/client-go/util/homedir"
)

func main() {
	// parse command-line arguments
	var kubeconfig *string
	var googleAppCreds *string
	var ingressHost *string

	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	googleAppCreds = flag.String("google-app-creds", "", "absolute path to the google application credentials json file")
	ingressHost = flag.String("ingress-host", "", "Kubernetes Nginx ingress host (should be a domain name)")

	flag.Parse()

	if _, err := os.Stat(*kubeconfig); os.IsNotExist(err) {
		flag.PrintDefaults()
		log.Fatalf("Kubernetes config file doesn't exist: %v", err)
	}

	if _, err := os.Stat(*googleAppCreds); os.IsNotExist(err) {
		flag.PrintDefaults()
		log.Fatalf("Google Application Credentials file doesn't exist: %v", err)
	}

	if *ingressHost == "" {
		flag.PrintDefaults()
		log.Fatalf("Kubernetes Ingress Host flag (-ingress-host) is missing.")
	}

	// Initialize clients to connect to external services
	clients.InitKubernetesClient(*kubeconfig)

	r := mux.NewRouter()
	// Root page
	r.HandleFunc("/", controller.RootController).Methods(http.MethodGet)

	s := http.StripPrefix("/static/", http.FileServer(http.Dir("./static/")))
	r.PathPrefix("/static/").Handler(s)

	// Model controllers
	r.HandleFunc("/model:deploy", controller.DeployControllerWrapper(*googleAppCreds, *ingressHost)).Methods(http.MethodPost)
	r.HandleFunc("/model/strategy", controller.ModelStrategyController).Methods(http.MethodPut)
	r.HandleFunc("/model:predict", controller.ModelPredictControllerWrapper(*ingressHost)).Methods(http.MethodPost)

	http.ListenAndServe(":8080", r)
}
