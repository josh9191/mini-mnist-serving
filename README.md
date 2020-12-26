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

![Web Page](https://user-images.githubusercontent.com/17065620/101513456-8e11cc80-39bf-11eb-8d96-4232de5fd34f.png)

## Deploy current / new models
You can see "Deploy" buttons in both "Current Model" and "New Model" sections.

Please set the Tensorflow model directory in your Google Cloud Storage (currently storage services other than GCS are not supported) and number of replicas (number of Pods) in the form.
In the example below, the Tensorflow saved model should be located in "gs://my-bucket/mnist/model/1" directory.

![Deploy model](https://user-images.githubusercontent.com/17065620/101513549-a681e700-39bf-11eb-8be1-e6f37363c757.png)

After the models are deployed, you can re-deploy or set strategy (Current model only / New model only / Canary) and predict your hand-written image.

## Setting strategy
You can set strategy using "Set Strategy" button. When you select "Canary", the portion of requests to be sent to your model can be adjusted by range bar.

Depending on your strategy, the input data will be sent to the current model or new model.

![Set strategy](https://user-images.githubusercontent.com/17065620/101514675-e09fb880-39c0-11eb-9b7e-6d155bff9c8c.png)

## Run prediction
After the strategy has been set, you can run prediction using your own hand-written image by clicking "Predict" button.

![Prediction](https://user-images.githubusercontent.com/17065620/101514964-2d838f00-39c1-11eb-8612-73b398edf12a.png)

You can check the result (probability) in a chart on the right side. Enjoy!

## Caveats
- TODO

## License
[MIT](https://choosealicense.com/licenses/mit/)
