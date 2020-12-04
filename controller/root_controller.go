package controller

import (
	"context"
	"html/template"
	"log"
	"net/http"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/josh9191/mini-mnist-serving/clients"
	"github.com/josh9191/mini-mnist-serving/constants"
)

type TemplateVar struct {
	ProdModelReady   bool
	CanaryModelReady bool
	CurrentStrategy  constants.Strategy
}

// RootController renders root page
func RootController(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("templates/index.html"))

	prodModelReady := false
	canaryModelReady := false
	curStrategy := constants.None

	kubeClientSet := clients.GetKubernetesClientSet()
	// check the model is deployed - deployment
	deploymentsClient := kubeClientSet.AppsV1().Deployments(getNamespace(false))
	result, err := deploymentsClient.Get(context.TODO(), constants.DeploymentName, metav1.GetOptions{})
	if err != nil {
		// to confirm
		prodModelReady = false
	}
	// deployment ready
	if result.Status.AvailableReplicas > 0 {
		prodModelReady = true
		log.Println("Current model is ready.")
	}

	deploymentsClient = kubeClientSet.AppsV1().Deployments(getNamespace(true))
	result, err = deploymentsClient.Get(context.TODO(), constants.DeploymentName, metav1.GetOptions{})
	if err != nil {
		// to confirm
		canaryModelReady = false
	}
	// deployment ready
	if result.Status.AvailableReplicas > 0 {
		canaryModelReady = true
		log.Println("New model is ready.")
	}

	// check canary metadata
	ingressClient := kubeClientSet.ExtensionsV1beta1().Ingresses(getNamespace(true))
	ingressResult, err := ingressClient.Get(context.TODO(), constants.IngressName, metav1.GetOptions{})
	if err != nil {
		log.Println("Error getting ingress. Maybe it is not created.")
	} else {
		_, hasCanaryHeaderKey := ingressResult.ObjectMeta.Annotations["nginx.ingress.kubernetes.io/canary-by-header"]
		if hasCanaryHeaderKey {
			_, hasCanaryWeightKey := ingressResult.ObjectMeta.Annotations["nginx.ingress.kubernetes.io/canary-weight"]
			if hasCanaryWeightKey {
				curStrategy = constants.Canary
			} else {
				curStrategy = constants.NewModelOnly
			}
		} else {
			curStrategy = constants.CurrentModelOnly
		}
	}

	tmpl.Execute(w, TemplateVar{
		ProdModelReady:   prodModelReady,
		CanaryModelReady: canaryModelReady,
		CurrentStrategy:  curStrategy,
	})
}
