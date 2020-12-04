package controller

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/josh9191/mini-mnist-serving/clients"
	"github.com/josh9191/mini-mnist-serving/constants"

	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	exv1beta1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// DeployRequest stores deploy request JSON data
type DeployRequest struct {
	ModelBaseDir string `json:"model-base-dir"`
	ModelName    string `json:"model-name"`
	IsNewModel   bool   `json:"is-new-model"`
	NumReplicas  int32  `json:"num-replicas"`
}

// SetStrategyRequest stores strategy set request JSON data
type SetStrategyRequest struct {
	Strategy constants.Strategy `json:"strategy"`
	Weight   *int               `json:"weight,omitempty"`
}

// PredictResponse stores prediction data
type PredictResponse struct {
	Predictions [][]float32 `json:"predictions"`
}

// DeployControllerWrapper deploys model
func DeployControllerWrapper(googleCredsFilePath string, ingressHost string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		googleCredsB64Encoded, err := readFileToBase64String(googleCredsFilePath)

		decoder := json.NewDecoder(r.Body)

		var deployRequest DeployRequest
		err = decoder.Decode(&deployRequest)

		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		// [FIXME] model name should be fixed to "model"
		if deployRequest.ModelName != "model" {
			http.Error(w, "The model name should be set to \"model\".", 500)
			return
		}

		kubeClientSet := clients.GetKubernetesClientSet()

		// First of all, we create namespaces for production / canary deployment
		// named mnist-prod, mnist-canary respectively.
		namespacesClient := kubeClientSet.CoreV1().Namespaces()
		prodNamespace := &apiv1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: constants.ProdNamespace,
			},
		}
		canaryNamespace := &apiv1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: constants.CanaryNamespace,
			},
		}

		namespaceResult, err := namespacesClient.Create(context.TODO(), prodNamespace, metav1.CreateOptions{})
		if err != nil {
			if errors.IsAlreadyExists(err) {
				log.Printf("The namespace %v already exists.", namespaceResult.GetObjectMeta().GetName())
			} else {
				http.Error(w, err.Error(), 500)
				return
			}
		}
		namespaceResult, err = namespacesClient.Create(context.TODO(), canaryNamespace, metav1.CreateOptions{})
		if err != nil {
			if errors.IsAlreadyExists(err) {
				log.Printf("The namespace %v already exists.", namespaceResult.GetObjectMeta().GetName())
			} else {
				http.Error(w, err.Error(), 500)
				return
			}
		}

		// Create secret if it doesn't exist
		secretsClient := kubeClientSet.CoreV1().Secrets(getNamespace(deployRequest.IsNewModel))
		secret := &apiv1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: constants.ModelSecretName,
			},
			StringData: map[string]string{
				"sa_json": googleCredsB64Encoded,
			},
		}
		secretResult, err := secretsClient.Create(context.TODO(), secret, metav1.CreateOptions{})
		if err != nil {
			if errors.IsAlreadyExists(err) {
				log.Printf("The secret %v already exists.", secretResult.GetObjectMeta().GetName())
			} else {
				http.Error(w, err.Error(), 500)
				return
			}
		}

		// Create or update deployment
		deploymentsClient := kubeClientSet.AppsV1().Deployments(getNamespace(deployRequest.IsNewModel))
		// container env
		envVar := []apiv1.EnvVar{
			{
				Name:  "MODEL_BASE_PATH",
				Value: deployRequest.ModelBaseDir,
			},
			{
				Name:  "MODEL_NAME",
				Value: deployRequest.ModelName,
			},
			{
				Name:  "GOOGLE_APPLICATION_CREDENTIALS",
				Value: "/etc/gcp/sa_credentials.json",
			},
		}
		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name: constants.DeploymentName,
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: int32Ptr(deployRequest.NumReplicas),
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": constants.LabelAppSelector,
					},
				},
				Template: apiv1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app": constants.LabelAppSelector,
						},
					},
					Spec: apiv1.PodSpec{
						Containers: []apiv1.Container{
							{
								Name:  "tensorflow-serving",
								Image: "tensorflow/serving:latest",
								Ports: []apiv1.ContainerPort{
									{
										Name:          "http",
										Protocol:      apiv1.ProtocolTCP,
										ContainerPort: 8501,
									},
								},
								VolumeMounts: []apiv1.VolumeMount{
									{
										Name:      "google-app-creds-vol",
										MountPath: "/etc/gcp",
										ReadOnly:  true,
									},
								},
								Env: envVar,
								ReadinessProbe: &apiv1.Probe{
									Handler: apiv1.Handler{
										TCPSocket: &apiv1.TCPSocketAction{
											Port: intstr.FromInt(8500),
										},
									},
									InitialDelaySeconds: 5,
									PeriodSeconds:       10,
								},
							},
						},
						Volumes: []apiv1.Volume{
							{
								Name: "google-app-creds-vol",
								VolumeSource: apiv1.VolumeSource{
									Secret: &apiv1.SecretVolumeSource{
										SecretName: constants.ModelSecretName,
										Items: []apiv1.KeyToPath{
											{
												Key:  "sa_json",
												Path: "sa_credentials.json",
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		deploymentResult, err := deploymentsClient.Create(context.TODO(), deployment, metav1.CreateOptions{})
		if err != nil {
			if errors.IsAlreadyExists(err) {
				log.Printf("The deployment %v already exists. Force rolling update...", deploymentResult.GetObjectMeta().GetName())

				result, err := deploymentsClient.Get(context.TODO(), constants.DeploymentName, metav1.GetOptions{})
				if err != nil {
					http.Error(w, err.Error(), 500)
					return
				}

				// update model base directory / name
				result.Spec.Template.Spec.Containers[0].Env = envVar
				result.Spec.Replicas = int32Ptr(deployRequest.NumReplicas)
				// Force rolling update using date label
				if result.Spec.Template.ObjectMeta.Annotations == nil {
					result.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
				}
				result.Spec.Template.ObjectMeta.Annotations["date"] = strconv.FormatInt(time.Now().Unix(), 10)
				_, err = deploymentsClient.Update(context.TODO(), result, metav1.UpdateOptions{})
				if err != nil {
					http.Error(w, err.Error(), 500)
					return
				}

			} else {
				http.Error(w, err.Error(), 500)
				return
			}
		}

		// Create Service object
		servicesClient := kubeClientSet.CoreV1().Services(getNamespace(deployRequest.IsNewModel))
		service := &apiv1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name: constants.ServiceName,
			},
			Spec: apiv1.ServiceSpec{
				Type: apiv1.ServiceTypeClusterIP,
				Selector: map[string]string{
					"app": constants.LabelAppSelector,
				},
				Ports: []apiv1.ServicePort{
					{
						Protocol:   apiv1.ProtocolTCP,
						Port:       8501,
						TargetPort: intstr.FromInt(8501),
					},
				},
			},
		}
		serviceResult, err := servicesClient.Create(context.TODO(), service, metav1.CreateOptions{})
		if err != nil {
			if errors.IsAlreadyExists(err) {
				log.Printf("The service %v already exists.", serviceResult.GetObjectMeta().GetName())
			} else {
				http.Error(w, err.Error(), 500)
				return
			}
		}

		// Ingress - 80 port
		ingressClient := kubeClientSet.ExtensionsV1beta1().Ingresses(getNamespace(deployRequest.IsNewModel))
		var nginxAnnotations = make(map[string]string)
		nginxAnnotations["kubernetes.io/ingress.class"] = "nginx"
		if deployRequest.IsNewModel {
			nginxAnnotations["nginx.ingress.kubernetes.io/canary"] = "true"
		} else {
			// If canary option is specified to true, other annotations are ignored
			// We set rewrite option only to prod model
			nginxAnnotations["nginx.ingress.kubernetes.io/rewrite-target"] = fmt.Sprintf("/v1/models/%s:predict", deployRequest.ModelName)
		}

		ingress := &exv1beta1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Name:        constants.IngressName,
				Annotations: nginxAnnotations,
			},
			Spec: exv1beta1.IngressSpec{
				Rules: []exv1beta1.IngressRule{
					{
						Host: ingressHost,
						IngressRuleValue: exv1beta1.IngressRuleValue{
							HTTP: &exv1beta1.HTTPIngressRuleValue{
								Paths: []exv1beta1.HTTPIngressPath{
									{
										Backend: exv1beta1.IngressBackend{
											ServiceName: constants.ServiceName,
											ServicePort: intstr.FromInt(8501),
										},
										Path: "/predict",
									},
								},
							},
						},
					},
				},
			},
		}

		ingressResult, err := ingressClient.Create(context.TODO(), ingress, metav1.CreateOptions{})
		if err != nil {
			if errors.IsAlreadyExists(err) {
				log.Printf("The ingress %v already exists.", ingressResult.GetObjectMeta().GetName())
				result, err := ingressClient.Get(context.TODO(), constants.IngressName, metav1.GetOptions{})
				if err != nil {
					http.Error(w, err.Error(), 500)
					return
				}

				// Force rolling update using date label
				result.ObjectMeta.Annotations = nginxAnnotations
				_, err = ingressClient.Update(context.TODO(), result, metav1.UpdateOptions{})
				if err != nil {
					http.Error(w, err.Error(), 500)
					return
				}
			} else {
				http.Error(w, err.Error(), 500)
				return
			}
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Deployed: %v\n", deployRequest.ModelName)
	}

}

// ModelStrategyController sets routing strategy
func ModelStrategyController(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)

	var setStrategyRequest SetStrategyRequest
	err := decoder.Decode(&setStrategyRequest)

	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	kubeClientSet := clients.GetKubernetesClientSet()
	// canary namespace
	ingressClient := kubeClientSet.ExtensionsV1beta1().Ingresses(getNamespace(true))
	result, err := ingressClient.Get(context.TODO(), constants.IngressName, metav1.GetOptions{})

	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	strategyStr := ""
	if setStrategyRequest.Strategy == constants.CurrentModelOnly {
		result.ObjectMeta.Annotations["nginx.ingress.kubernetes.io/canary"] = "false"
		deleteMapKeyIfExists(result.ObjectMeta.Annotations, "nginx.ingress.kubernetes.io/canary-by-header")
		deleteMapKeyIfExists(result.ObjectMeta.Annotations, "nginx.ingress.kubernetes.io/canary-weight")
		strategyStr = "Current Model Only"
	} else if setStrategyRequest.Strategy == constants.NewModelOnly {
		result.ObjectMeta.Annotations["nginx.ingress.kubernetes.io/canary"] = "true"
		result.ObjectMeta.Annotations["nginx.ingress.kubernetes.io/canary-by-header"] = constants.CanaryHeader
		deleteMapKeyIfExists(result.ObjectMeta.Annotations, "nginx.ingress.kubernetes.io/canary-weight")
		strategyStr = "New Model Only"
	} else { // else if setStrategyRequest.Strategy == constants.Canary
		if setStrategyRequest.Weight == nil {
			http.Error(w, "Weight missing.", 500)
			return
		}
		result.ObjectMeta.Annotations["nginx.ingress.kubernetes.io/canary"] = "true"
		deleteMapKeyIfExists(result.ObjectMeta.Annotations, "nginx.ingress.kubernetes.io/canary-by-header")
		result.ObjectMeta.Annotations["nginx.ingress.kubernetes.io/canary-weight"] = strconv.Itoa(*setStrategyRequest.Weight)
		strategyStr = "Canary"
	}

	_, err = ingressClient.Update(context.TODO(), result, metav1.UpdateOptions{})
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	fmt.Fprintf(w, "Changed to strategy: %v\n", strategyStr)
}

// ModelPredictControllerWrapper handles prediction
func ModelPredictControllerWrapper(ingressHost string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		decoder := json.NewDecoder(r.Body)
		var pixels []float32
		err := decoder.Decode(&pixels)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		if len(pixels) != 784 {
			http.Error(w, "Pixel should have 768 elements.", 500)
			return
		}

		// [FIXME] better way to reshape array
		var reshapedPixels [][][][]float32 = make([][][][]float32, 1)
		for i := 0; i < 1; i++ {
			reshapedPixels[i] = make([][][]float32, 28)
			for j := 0; j < 28; j++ {
				reshapedPixels[i][j] = make([][]float32, 28)
				for k := 0; k < 28; k++ {
					reshapedPixels[i][j][k] = make([]float32, 1)
					for l := 0; l < 1; l++ {
						reshapedPixels[i][j][k][l] = pixels[j*28+k]
					}
				}
			}
		}

		pixelJson, err := json.Marshal(reshapedPixels)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		requestJson := []byte(fmt.Sprintf(`{"signature_name": "serving_default", "instances": %v}`, string(pixelJson)))
		predictUrl, err := url.Parse(ingressHost)

		// [FIXME] scheme as flag
		predictUrl.Scheme = "http"
		predictUrl.Path = path.Join(predictUrl.Path, "predict")

		req, err := http.NewRequest("POST", predictUrl.String(), bytes.NewBuffer(requestJson))
		// the header will be ignored when non-canary model prediction
		req.Header.Set(constants.CanaryHeader, "always")
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		defer resp.Body.Close()
		body, _ := ioutil.ReadAll(resp.Body)
		var predResp PredictResponse
		json.Unmarshal(body, &predResp)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(predResp.Predictions[0])
	}
}

func deleteMapKeyIfExists(m map[string]string, key string) {
	_, ok := m[key]
	if ok {
		delete(m, key)
	}
}

func readFileToBase64String(googleCredsFilePath string) (string, error) {
	googleCredsFile, err := os.Open(googleCredsFilePath)
	if err != nil {
		return "", err
	}
	reader := bufio.NewReader(googleCredsFile)
	content, _ := ioutil.ReadAll(reader)

	googleCredsB64Encoded := base64.StdEncoding.EncodeToString(content)

	googleCredsFile.Close()
	return googleCredsB64Encoded, nil
}

// getNamespace returns namespace
func getNamespace(isNewModel bool) string {
	if isNewModel {
		return constants.CanaryNamespace
	} else {
		return constants.ProdNamespace
	}
}

func int32Ptr(i int32) *int32 { return &i }
