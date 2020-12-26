package controller

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/josh9191/mini-mnist-serving/clients"
	"k8s.io/client-go/util/homedir"
)

func TestRootController(t *testing.T) {
	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Error(err)
	}

	// initliaze kubernetes configuration - you should use config file in hoe directory ($HOME/.kube/config)
	home := homedir.HomeDir()
	kubeconfig := filepath.Join(home, ".kube", "config")
	clients.InitKubernetesClient(kubeconfig)

	w := httptest.NewRecorder()
	handler := http.HandlerFunc(RootController)
	handler.ServeHTTP(w, r)

	resp := w.Result()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Error - Status Code: %d", resp.StatusCode)
	}
}
