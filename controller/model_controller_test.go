package controller

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/josh9191/mini-mnist-serving/clients"
	"k8s.io/client-go/util/homedir"
)

func TestDeployControllerWrapper(t *testing.T) {
	const modelBaseDir string = "gs://nice-soldev-tf-models/mnist-new/model/1"
	const modelName string = "model"
	const isNewModel bool = true
	const numReplicas int = 2

	deployReqBody := map[string]interface{}{
		"model-base-dir": modelBaseDir,
		"model-name":     modelName,
		"is-new-model":   isNewModel,
		"num-replicas":   numReplicas,
	}
	body, _ := json.Marshal(deployReqBody)

	r, err := http.NewRequest("POST", "/model:deploy", bytes.NewReader(body))
	if err != nil {
		t.Error(err)
	}

	// initliaze kubernetes configuration - you should use config file in hoe directory ($HOME/.kube/config)
	home := homedir.HomeDir()
	kubeconfig := filepath.Join(home, ".kube", "config")
	clients.InitKubernetesClient(kubeconfig)

	// [FIXME] change the paths below
	googleCredsFilePath := "/home/josh9191/gcp/key.json"
	ingressHost := "mini-serving.duckdns.org"

	w := httptest.NewRecorder()
	handlerFunc := DeployControllerWrapper(googleCredsFilePath, ingressHost)
	handler := http.HandlerFunc(handlerFunc)

	handler.ServeHTTP(w, r)

	resp := w.Result()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Error - Status Code: %d", resp.StatusCode)
	}
}

func TestModelPredictControllerWrapper(t *testing.T) {
	deployReqBody := make([]float32, 784)
	body, _ := json.Marshal(deployReqBody)

	r, err := http.NewRequest("POST", "/model:predict", bytes.NewReader(body))
	if err != nil {
		t.Error(err)
	}

	// initliaze kubernetes configuration - you should use config file in hoe directory ($HOME/.kube/config)
	home := homedir.HomeDir()
	kubeconfig := filepath.Join(home, ".kube", "config")
	clients.InitKubernetesClient(kubeconfig)

	// [FIXME] change the path below
	ingressHost := "mini-serving.duckdns.org"

	w := httptest.NewRecorder()
	handlerFunc := ModelPredictControllerWrapper(ingressHost)
	handler := http.HandlerFunc(handlerFunc)

	handler.ServeHTTP(w, r)

	resp := w.Result()
	decoder := json.NewDecoder(resp.Body)
	var predictions []float32
	err = decoder.Decode(&predictions)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Error - Status Code: %d", resp.StatusCode)
	}

	var max float32 = -1.0
	argmax := 0
	for i := 0; i < len(predictions); i++ {
		if max < predictions[i] {
			max = predictions[i]
			argmax = i
		}
	}

	if argmax != 5 {
		t.Errorf("Unit test failed - Wrong prediction value")
		t.Errorf("Input: %v", deployReqBody)
	}
}
