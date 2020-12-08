# mini-mnist-serving
Mini MNIST Serving

## Architecture
![architecture](https://user-images.githubusercontent.com/17065620/101497209-125b5400-39ae-11eb-97de-b071ee61d74a.png)

## Prerequisites
This project is tested on Go 1.14.6 and any other environments should be tested later.

Also Kubernetes cluster should be installed and Nginx ingress controller should exist.

## Run application
To run the application, you need to set some arguments.

- -google-app-creds
  - Google application credentials json file path
  - Because the application accesses the Google Cloud Storage, we need to set credentials to retrieve data.
  - Please follow the [Link](https://cloud.google.com/docs/authentication/getting-started) to get json file.
  - ex) /home/josh9191/key.json
  
- -ingress-host
  - Kubernetes Nginx ingress host
  - Domain name of server where Nginx ingress controller runs
  - ex) my.example.com

- -kubeconfig (optional - but the file should exist)
  - Kubernetes client configuration file path
  - Copy your .kube/config file from your cluster to your local machine where server is running on.
  - By default, it is set to "$HOME/.kube/config".
  - ex) /etc/kube/config
 
You can run server as follows.
```
go run cmd\main.go \
  -google-app-creds /home/josh9191/gcp/key.json \
  -ingress-host my-serving.duckdns.org \
  -kubeconfig /etc/kube/config
```

After the server started, you can access the web page via http://localhost:8080.

![Web Page](https://user-images.githubusercontent.com/17065620/101507464-e940c080-39b9-11eb-9bed-ce6cab9a098f.png)

## Deploy current / new models
You can see "Deploy" buttons in both "Current Model" and "New Model" sections.

Please set the Tensorflow model directory in your Google Cloud Storage (currently storage services other than GCS are not supported) and number of replicas (number of Pods) in the form.
In the example below, the Tensorflow saved model should be located in "gs://my-bucket/mnist/model/1" directory.

![Deploy model](https://user-images.githubusercontent.com/17065620/101508201-b814c000-39ba-11eb-976b-19b5c12520bf.png)
