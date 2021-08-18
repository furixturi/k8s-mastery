# What is this
A review and learn session of Microservices, Docker, k8s.

Started with this wonderful material:
https://www.freecodecamp.org/news/learn-kubernetes-in-under-3-hours-a-detailed-guide-to-orchestrating-containers-114ff420e882/

With a rewrite of the Java web-app in Golang, experiments to multi-stage build FE image with ARGs, use ENVs for flexibility wherever needed, etc.

Below are notes I took on the go.
# Step 1: Locally get the three apps up and running and working together

## 1. sa-frontend
### Nginx on Mac
#### Install with Homebrew

```sh
$ brew install nginx
```

```sh
==> nginx
Docroot is: /usr/local/var/www

The default port has been set in /usr/local/etc/nginx/nginx.conf to 8080 so that
nginx can run without sudo.

nginx will load all files in /usr/local/etc/nginx/servers/.

To have launchd start nginx now and restart at login:
  brew services start nginx
Or, if you don't want/need a background service you can just run:
  nginx
```
**Note**: The doc root of Nginx Docker image is at
```
/usr/share/nginx/html
```

#### Start, Stop, Log
- start
```
$ nginx
```
Open in browser
http://localhost:8080/

- stop
```
$ nginx -s stop
```
- access log
```
$ tail -f /usr/local/var/log/nginx/access.log
```
- error log
```
$ tail -f /usr/local/var/log/nginx/error.log
```

Reference: https://utano.jp/entry/2018/08/macos-path-to-nginx-access-and-error-log-used-homebrew/

## 2.1 sa-webapp
### Java and Maven
#### Install JDK
1. Download the JDK .dmg file, jdk-15.interim.update.patch_osx-x64_bin.dmg from [Java SE Downloads](https://www.oracle.com/java/technologies/javase-downloads.html) page.
2. Execute the downloaded .dmg is installed to `/Library/Java/JavaVirtualMachines`
3. Set the `JAVA_HOME` environment variable

```
export JAVA_HOME=/Library/Java/JavaVirtualMachines/jdk-16.0.2.jdk/Contents/Home
```
Verify:
```
$ echo $JAVA_HOME
/Library/Java/JavaVirtualMachines/jdk-16.0.2.jdk/Contents/Home
```

#### Install Maven

```
$ brew install maven
```
Verify
```
$ mvn -v
```

#### Didn't work after all
```
$ java -jar sentiment-analysis-web-0.0.1-SNAPSHOT.jar --sa.logic.api.url=http://localhost:5000
```
Probaboly Java version problem...anyway

## 2.2 sa-webapp-go
A rewrite of the Java sa-webapp in go
### init

Generate the go.mod file
```
$ go mod init github.com/furixturi/sa-webapp-go
```

Install dependency and generates go.sum file
```
$ go get -u github.com/gorilla/mux 
```

### go get module proxy problem

If this error comes up
```
$ go get -u github.com/gorilla/mux                                                   
go get: module github.com/gorilla/mux: Get "https://proxy.golang.org/github.com/gorilla/mux/@v/list": dial tcp: i/o timeout
```
Don't use go proxy
```
export GOPROXY=direct
```
### run
Have to add both
```
$ go run main.go app.go
```

### References for coding
#### CORS
https://stackoverflow.com/questions/40985920/making-golang-gorilla-cors-handler-work
https://github.com/rs/cors

#### make request
https://zetcode.com/golang/getpostrequest/

#### mux CRUD
https://semaphoreci.com/community/tutorials/building-and-testing-a-rest-api-in-go-with-gorilla-mux-and-postgresql


# Step 2: dockerize everything
- Run an image and access its bash
```
docker run -it <image> /bin/bash
```
- To build an image:
```
$ docker build -f Dockerfile -t $DOCKER_USERNAME/sentiment-analysis-frontend .
```
- To push it to docker hub
```
$ docker login -u=$DOCKER_HUB_USER -p=$DOCKER_HUB_PW
$ docker push alabebop/sentiment-analysis-frontend
```

- To run the image just built:
```
$ docker run --name sa-frontend -p 8081:80 -d alabebop/sentiment-analysis-frontend:latest
```
## sa-frontend
### Option 1 - build locally and only use nginx docker

The Nginx Docker image has the following default which is different from the local version:
- web doc root is `/usr/share/nginx/html`
- default port listening at is `80`

### Option 2 - multi-stage 
https://medium.com/geekculture/dockerizing-a-react-application-with-multi-stage-docker-build-4a5c6ca68166

### Use ARG to build with dynamic backend url

- Add ARG and ENV in Dockerfile
```
ARG SA_WEBAPP_URL
ARG SA_WEBAPP_PORT

ENV REACT_APP_BE_SERVICE_URL=$SA_WEBAPP_URL
ENV REACT_APP_BE_SERVICE_PORT=$SA_WEBAPP_PORT
```
- Use `--build-arg` when build image

```
$ docker build -f Dockerfile -t $DOCKER_USERNAME/sentiment-analysis-frontend-multistage --build-arg SA_WEBAPP_URL=http://localhost --build-arg SA_WEBAPP_PORT=8080 .
```

## sa-webapp-go
### build golang image
- The `golang:alpine` image doesn't include git so `go mod download` won't work out of the box. https://github.com/docker-library/golang/issues/209 
Two options:
  1. use `RUN apk update && apk upgrade && \
    apk add --no-cache bash git openssh` to add git & co., or
  2. use multi stage to build a much smaller image
     1. can't name the runtime stage
     2. at build stage, we need to build a static go executable without CGO (https://stackoverflow.com/questions/62632340/golang-linux-docker-error-standard-init-linux-go211-no-such-file-or-dir)
```
CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' .
```

## sa-logic
### Get ENV and convert to int
```
# sentiment_analysis.py
SA_LOGIC_PORT = int(os.getenv('SA_LOGIC_PORT'))
```
# Step 3: k8s with minikube
## minikube 
- Install minikube
https://minikube.sigs.k8s.io/docs/start/

  ```sh
  $ brew install minikube
  ```
- start minikube
  ```sh
  $ minikube start
  ```
- interact with the cluster
  ```sh
  $ kubectl get po -A
  ```
- enable the dashboard
  ```sh
  $ minikube dashboard
  ```
- get nodes
  ```
  $ kubctl get nodes
  NAME       STATUS   ROLES                  AGE   VERSION
  minikube   Ready    control-plane,master   9h    v1.21.2
  ```
- get pods and show labels
  ```
  $ kubctl get pods --show-labels
  NAME          READY   STATUS    RESTARTS   AGE   LABELS
  sa-frontend   1/1     Running   0          80m   app=sa-frontend
  ```

## create `pod` and access the application running on the pods
### sa-frontend
- update the image
- create the pod using `kubectl`
  ```sh
  $ kubectl create -f sa-frontend-pod.yaml
  ```
- use port-forward to access at `localhost` (note: use a port bigger than 1024)
  ```sh
  $ kubectl port-forward sa-frontend 8888:80
  ```
  note: this is not how we expose the frontend properly, deploy and use a load balancer service.
## create `service` - load balancer
### frontend load balancer
- use labels as selectors in the manifest yaml
  ```yaml
  # sa-frontend-pod.yaml
    metadata:
      ...
      labels:
        app: sa-frontend     
  ```
  ```yaml
  # service-sa-frontend-lb.yaml
    selector:
      app: sa-frontend
  ```
- create the service
  ```sh
  $ kubectl create -f service-sa-frontend-lb.yaml
  ```
- Verify:
  ```sh
  $ kubectl get svc
  NAME             TYPE           CLUSTER-IP     EXTERNAL-IP   PORT(S)        AGE
  kubernetes       ClusterIP      10.96.0.1      <none>        443/TCP        10h
  sa-frontend-lb   LoadBalancer   10.98.205.78   <pending>     80:31820/TCP   18s
  ```
### start the service with minikube
minikube will open it up in the default browser
```sh
$ minikube service sa-frontend-lb
```
### Important notes for frontend
- cannot use ENV since the code will be running in the user browser and there's no way to inject runtime ENV there.
- if there's some dynamic content to inject, use `ARG` then `ENV=$ARG` in Dockerfile to inject at build time
- **cannot** use k8s `cluster internal IP` or `cluster internal host name` since the frontend code is running in the user's browser, not in the cluster
  - which means the webapp URL for the frontend needs to be accessible outside of the cluster, e.g., by deploying a public `load balancer` service
## create `deployment`
About the `apiVersion: apps/v1` in `sa-frontend-deployment.yaml`: https://matthewpalmer.net/kubernetes-app-developer/articles/kubernetes-apiversion-definition-guide.html

### create a rolling deployment

- To provide ENVs in the deployment manifest yaml:
  ```
  # sa-web-app-deployment.yaml
    ...
    template:
      ...
      spec:
        containers:
        - image: alabebop/sentiment-analysis-webapp-multistage
          imagePullPolicy: Always
          name: sa-web-app
          env:
            - name: SA_LOGIC_URL
              value: "http://sa-logic"
              # value: "http://10.108.232.130"
            - name: SA_LOGIC_PORT
              value: "80"
            - name: SA_WEBAPP_PORT
              value: "8080"
          ...
  ```
- To create and update the deployment manifest yaml:
  ```sh
  $ kubectl apply -f sa-frontend-deployment.yaml
  ```
  Repeat this when we e.g. change an ENV value.

- To verify the deployed pods: 

  ```sh
  $ kubectl get pods --show-labels
  NAME                           READY   STATUS    RESTARTS   AGE     LABELS
  sa-frontend # this was manual  1/1     Running   0          128m    app=sa-frontend
  sa-frontend-69465f4877-55zx6   1/1     Running   0          5m48s   app=sa-frontend,pod-template-hash=69465f4877
  sa-frontend-69465f4877-pwxl5   1/1     Running   0          5m48s   app=sa-frontend,pod-template-hash=69465f4877
  ```

- To delete the sa-frontend, which was created earlier separately.
  ```sh
  $ kubectl delete pod sa-frontend
  ```
  If we rebuilt and pushed a new image, do this will deploy new containers with the updated image.
### roll out a new version with the possibility to restore using the `--record` flag
- rollout
  ```
  $ kubectl apply -f sa-frontend-deployment-green.yaml --record
  deployment "sa-frontend" configured
  ```
- monitor the rollout status
  ```
  $ kubectl rollout status deployment sa-frontend
  Waiting for deployment "sa-frontend" rollout to finish: 1 old replicas are pending termination...
  Waiting for deployment "sa-frontend" rollout to finish: 1 old replicas are pending termination...
  Waiting for deployment "sa-frontend" rollout to finish: 1 old replicas are pending termination...
  Waiting for deployment "sa-frontend" rollout to finish: 1 old replicas are pending termination...
  Waiting for deployment "sa-frontend" rollout to finish: 1 old replicas are pending termination...
  Waiting for deployment "sa-frontend" rollout to finish: 1 of 2 updated replicas are available...
  deployment "sa-frontend" successfully rolled out
  ```
- check rollout history
  ```sh
  $ kubectl rollout history deployment sa-frontend
  deployment.apps/sa-frontend
  REVISION  CHANGE-CAUSE
  1         <none>
  2         kubectl apply --filename=sa-frontend-deployment-green.yaml --record=true
  ```
- revert to the previous version
  ```sh
  $ kubectl rollout undo deployment sa-frontend --to-revision=1
  deployment.apps/sa-frontend rolled back
  ```

## the sa-logic
### create the deployment
```sh
$ kubectl apply -f sa-logic-deployment.yaml
```

### create the service
```sh
$ kubectl apply -f service-sa-logic.yaml
```
Verify
```sh
$ kubectl get svc
NAME             TYPE           CLUSTER-IP       EXTERNAL-IP   PORT(S)        AGE
kubernetes       ClusterIP      10.96.0.1        <none>        443/TCP        16h
sa-frontend-lb   LoadBalancer   10.98.205.78     <pending>     80:31820/TCP   5h37m
sa-logic         ClusterIP      10.108.232.130   <none>        80/TCP         147m
sa-web-app-lb    LoadBalancer   10.104.249.181   <pending>     80:32161/TCP   88m
```
Other containers in the cluster can access this service by its name defined in the manifest yaml.
```yaml
# service-sa-logic.yaml
apiVersion: v1
kind: Service
metadata:
  name: sa-logic
spec:
  ports:
    - port: 80
      protocol: TCP
      targetPort: 5000
...
```
```yaml
# sa-web-app-deployment.yaml
...
    spec:
      containers:
      - image: alabebop/sentiment-analysis-webapp-multistage
        imagePullPolicy: Always
        name: sa-web-app
        env:
          - name: SA_LOGIC_URL
            value: "http://sa-logic"
            # or use the internal IP
            # value: "http://10.108.232.130"
          - name: SA_LOGIC_PORT
            value: "80"
```
Pay attention to use the exposed port of the internal service, not its target port.
## work with k8s containers with `kubectl`
### follow log of containers under a particular label
```
$ kubectl logs -f -l app=sa-web-app
```
### get the shell of a container

```
$ kubectl exec -it sa-web-app-699fd8cfcd-m8km2 -- /bin/bash
```
### Linux Alpine
- install curl
  ```
  bash-5.1# apk --no-cache add curl
  ```
- print all env
  ```
  bash-5.1# printenv
  ```


# Step 4: EKS

## Prepare to use EKS
### install eksctl
https://docs.aws.amazon.com/eks/latest/userguide/eksctl.html

1. If no Homebrew, install Homebrew
  ```
  $ /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/master/install.sh)"
  ```

2. Install Weaveworks Homebrew tap
  ```
  $ brew tap weaveworks/tap
  ```

3. Install `eksctl`
  ```
  $ brew install weaveworks/tap/eksctl
  ```
  verify
  ```
  $ eksctl version
  0.60.0
  ```

## Deploy an EKS fargate cluster
### Use `eksctl`
https://docs.aws.amazon.com/eks/latest/userguide/getting-started-eksctl.html
```
$ eksctl create cluster \                           
--name sentiment-analysis \
--region ap-northeast-1 \
--fargate
```
It took 32 min to create 
```
2021-08-08 21:42:37 [ℹ]  eksctl version 0.60.0
2021-08-08 21:42:37 [ℹ]  using region ap-northeast-1
2021-08-08 21:42:38 [ℹ]  setting availability zones to [ap-northeast-1d ap-northeast-1c ap-northeast-1a]
2021-08-08 21:42:38 [ℹ]  subnets for ap-northeast-1d - public:192.168.0.0/19 private:192.168.96.0/19
2021-08-08 21:42:38 [ℹ]  subnets for ap-northeast-1c - public:192.168.32.0/19 private:192.168.128.0/19
2021-08-08 21:42:38 [ℹ]  subnets for ap-northeast-1a - public:192.168.64.0/19 private:192.168.160.0/19
2021-08-08 21:42:38 [ℹ]  nodegroup "ng-ca0f54b4" will use "" [AmazonLinux2/1.20]
2021-08-08 21:42:38 [ℹ]  using Kubernetes version 1.20
2021-08-08 21:42:38 [ℹ]  creating EKS cluster "sentiment-analysis" in "ap-northeast-1" region with Fargate profile and managed nodes
2021-08-08 21:42:38 [ℹ]  will create 2 separate CloudFormation stacks for cluster itself and the initial managed nodegroup
2021-08-08 21:42:38 [ℹ]  if you encounter any issues, check CloudFormation console or try 'eksctl utils describe-stacks --region=ap-northeast-1 --cluster=sentiment-analysis'
2021-08-08 21:42:38 [ℹ]  CloudWatch logging will not be enabled for cluster "sentiment-analysis" in "ap-northeast-1"
2021-08-08 21:42:38 [ℹ]  you can enable it with 'eksctl utils update-cluster-logging --enable-types={SPECIFY-YOUR-LOG-TYPES-HERE (e.g. all)} --region=ap-northeast-1 --cluster=sentiment-analysis'
2021-08-08 21:42:38 [ℹ]  Kubernetes API endpoint access will use default of {publicAccess=true, privateAccess=false} for cluster "sentiment-analysis" in "ap-northeast-1"
2021-08-08 21:42:38 [ℹ]  2 sequential tasks: { create cluster control plane "sentiment-analysis", 3 sequential sub-tasks: { 2 sequential sub-tasks: { wait for control plane to become ready, create fargate profiles }, 1 task: { create addons }, create managed nodegroup "ng-ca0f54b4" } }
2021-08-08 21:42:38 [ℹ]  building cluster stack "eksctl-sentiment-analysis-cluster"
2021-08-08 21:42:39 [ℹ]  deploying stack "eksctl-sentiment-analysis-cluster"
2021-08-08 21:43:09 [ℹ]  waiting for CloudFormation stack "eksctl-sentiment-analysis-cluster"
2021-08-08 21:43:41 [ℹ]  waiting for CloudFormation stack "eksctl-sentiment-analysis-cluster"
2021-08-08 21:44:42 [ℹ]  waiting for CloudFormation stack "eksctl-sentiment-analysis-cluster"
2021-08-08 21:45:43 [ℹ]  waiting for CloudFormation stack "eksctl-sentiment-analysis-cluster"
2021-08-08 21:46:44 [ℹ]  waiting for CloudFormation stack "eksctl-sentiment-analysis-cluster"
2021-08-08 21:47:45 [ℹ]  waiting for CloudFormation stack "eksctl-sentiment-analysis-cluster"
2021-08-08 21:48:46 [ℹ]  waiting for CloudFormation stack "eksctl-sentiment-analysis-cluster"
2021-08-08 21:49:48 [ℹ]  waiting for CloudFormation stack "eksctl-sentiment-analysis-cluster"
2021-08-08 21:50:49 [ℹ]  waiting for CloudFormation stack "eksctl-sentiment-analysis-cluster"
2021-08-08 21:51:50 [ℹ]  waiting for CloudFormation stack "eksctl-sentiment-analysis-cluster"
2021-08-08 21:52:51 [ℹ]  waiting for CloudFormation stack "eksctl-sentiment-analysis-cluster"
2021-08-08 21:53:52 [ℹ]  waiting for CloudFormation stack "eksctl-sentiment-analysis-cluster"
2021-08-08 21:54:53 [ℹ]  waiting for CloudFormation stack "eksctl-sentiment-analysis-cluster"
2021-08-08 21:55:54 [ℹ]  waiting for CloudFormation stack "eksctl-sentiment-analysis-cluster"
2021-08-08 21:58:00 [ℹ]  creating Fargate profile "fp-default" on EKS cluster "sentiment-analysis"
2021-08-08 22:02:19 [ℹ]  created Fargate profile "fp-default" on EKS cluster "sentiment-analysis"
2021-08-08 22:04:51 [ℹ]  "coredns" is now schedulable onto Fargate
2021-08-08 22:06:58 [ℹ]  "coredns" is now scheduled onto Fargate
2021-08-08 22:06:58 [ℹ]  "coredns" pods are now scheduled onto Fargate
2021-08-08 22:09:02 [ℹ]  building managed nodegroup stack "eksctl-sentiment-analysis-nodegroup-ng-ca0f54b4"
2021-08-08 22:09:03 [ℹ]  deploying stack "eksctl-sentiment-analysis-nodegroup-ng-ca0f54b4"
2021-08-08 22:09:03 [ℹ]  waiting for CloudFormation stack "eksctl-sentiment-analysis-nodegroup-ng-ca0f54b4"
2021-08-08 22:09:19 [ℹ]  waiting for CloudFormation stack "eksctl-sentiment-analysis-nodegroup-ng-ca0f54b4"
2021-08-08 22:09:37 [ℹ]  waiting for CloudFormation stack "eksctl-sentiment-analysis-nodegroup-ng-ca0f54b4"
2021-08-08 22:09:58 [ℹ]  waiting for CloudFormation stack "eksctl-sentiment-analysis-nodegroup-ng-ca0f54b4"
2021-08-08 22:10:16 [ℹ]  waiting for CloudFormation stack "eksctl-sentiment-analysis-nodegroup-ng-ca0f54b4"
2021-08-08 22:10:37 [ℹ]  waiting for CloudFormation stack "eksctl-sentiment-analysis-nodegroup-ng-ca0f54b4"
2021-08-08 22:10:57 [ℹ]  waiting for CloudFormation stack "eksctl-sentiment-analysis-nodegroup-ng-ca0f54b4"
2021-08-08 22:11:17 [ℹ]  waiting for CloudFormation stack "eksctl-sentiment-analysis-nodegroup-ng-ca0f54b4"
2021-08-08 22:11:34 [ℹ]  waiting for CloudFormation stack "eksctl-sentiment-analysis-nodegroup-ng-ca0f54b4"
2021-08-08 22:11:53 [ℹ]  waiting for CloudFormation stack "eksctl-sentiment-analysis-nodegroup-ng-ca0f54b4"
2021-08-08 22:12:11 [ℹ]  waiting for CloudFormation stack "eksctl-sentiment-analysis-nodegroup-ng-ca0f54b4"
2021-08-08 22:12:28 [ℹ]  waiting for CloudFormation stack "eksctl-sentiment-analysis-nodegroup-ng-ca0f54b4"
2021-08-08 22:12:47 [ℹ]  waiting for CloudFormation stack "eksctl-sentiment-analysis-nodegroup-ng-ca0f54b4"
2021-08-08 22:12:48 [ℹ]  waiting for the control plane availability...
2021-08-08 22:12:48 [✔]  saved kubeconfig as "/Users/xiaolshe/.kube/config"
2021-08-08 22:12:48 [ℹ]  no tasks
2021-08-08 22:12:48 [✔]  all EKS cluster resources for "sentiment-analysis" have been created
2021-08-08 22:12:50 [ℹ]  nodegroup "ng-ca0f54b4" has 2 node(s)
2021-08-08 22:12:50 [ℹ]  node "ip-192-168-11-48.ap-northeast-1.compute.internal" is ready
2021-08-08 22:12:50 [ℹ]  node "ip-192-168-38-155.ap-northeast-1.compute.internal" is ready
2021-08-08 22:12:50 [ℹ]  waiting for at least 2 node(s) to become ready in "ng-ca0f54b4"
2021-08-08 22:12:50 [ℹ]  nodegroup "ng-ca0f54b4" has 2 node(s)
2021-08-08 22:12:50 [ℹ]  node "ip-192-168-11-48.ap-northeast-1.compute.internal" is ready
2021-08-08 22:12:50 [ℹ]  node "ip-192-168-38-155.ap-northeast-1.compute.internal" is ready
2021-08-08 22:14:56 [ℹ]  kubectl command should work with "/Users/xiaolshe/.kube/config", try 'kubectl get nodes'
2021-08-08 22:14:56 [✔]  EKS cluster "sentiment-analysis" in "ap-northeast-1" region is ready
```

This creates:
- A VPC with <u>3 public subnets</u> and <u>3 private subnets</u> in your default region
- A NAT GW in one of the public subnets
- An EKS Fargate cluster with a node group of 2 EC2
- 3 SGs: cluster, control plane, node
- An IAM role for the EKS cluster
- A config in `~/.kube` that switches your `kubectl` context to the EKS just created (https://kubernetes.io/docs/tasks/access-application-cluster/configure-access-multiple-clusters/)
  - To switch it to your local minikube EKS:
    ```sh
    $ kubectl config use-context minikube
    ```
  - To see which EKS context is the current
    ```sh
    $ kubectl config view --minify
    ```
  - The context name of the EKS cluster would be something like `{IAM-user-name}@{cluster-name}.{AWS-region}.eksctl.io`

To check the resources
- view nodes
  ```sh
  $ kubectl get nodes -o wide
  ```
  ```sh
  NAME                                                         STATUS   ROLES    AGE    VERSION              INTERNAL-IP       EXTERNAL-IP      OS-IMAGE         KERNEL-VERSION                  CONTAINER-RUNTIME
  fargate-ip-192-168-160-249.ap-northeast-1.compute.internal   Ready    <none>   15h    v1.20.4-eks-6b7464   192.168.160.249   <none>           Amazon Linux 2   4.14.238-182.422.amzn2.x86_64   containerd://1.4.1
  fargate-ip-192-168-164-117.ap-northeast-1.compute.internal   Ready    <none>   15h    v1.20.4-eks-6b7464   192.168.164.117   <none>           Amazon Linux 2   4.14.238-182.422.amzn2.x86_64   containerd://1.4.1
  ip-192-168-28-150.ap-northeast-1.compute.internal            Ready    <none>   138m   v1.20.4-eks-6b7464   192.168.28.150    13.230.73.74     Amazon Linux 2   5.4.129-63.229.amzn2.x86_64     docker://19.3.13
  ip-192-168-33-253.ap-northeast-1.compute.internal            Ready    <none>   138m   v1.20.4-eks-6b7464   192.168.33.253    175.41.226.192   Amazon Linux 2   5.4.129-63.229.amzn2.x86_64     docker://19.3.13
  ```

- view pods
  ```sh
  $ kubectl get pods --all-namespaces -o wide
  ```
  ```sh
  NAMESPACE     NAME                       READY   STATUS    RESTARTS   AGE    IP                NODE                                                         NOMINATED NODE   READINESS GATES
  kube-system   aws-node-b8fsx             1/1     Running   0          140m   192.168.33.253    ip-192-168-33-253.ap-northeast-1.compute.internal            <none>           <none>
  kube-system   aws-node-zjflw             1/1     Running   0          140m   192.168.28.150    ip-192-168-28-150.ap-northeast-1.compute.internal            <none>           <none>
  kube-system   coredns-6b9c5cf745-4xsgc   1/1     Running   0          15h    192.168.160.249   fargate-ip-192-168-160-249.ap-northeast-1.compute.internal   <none>           <none>
  kube-system   coredns-6b9c5cf745-c45mz   1/1     Running   0          15h    192.168.164.117   fargate-ip-192-168-164-117.ap-northeast-1.compute.internal   <none>           <none>
  kube-system   kube-proxy-7j45z           1/1     Running   0          140m   192.168.33.253    ip-192-168-33-253.ap-northeast-1.compute.internal            <none>           <none>
  kube-system   kube-proxy-mmndz           1/1     Running   0          140m   192.168.28.150    ip-192-168-28-150.ap-northeast-1.compute.internal            <none>           <none>
  ```

## Deploy the sentiment analysis on EKS

### Deploy sa-logic
- sa-logic deployment
  ```sh
  $ kubectl apply -f sa-logic-deployment.yaml
  ```
- sa-logic cluster ip service
  ```
  $ kubectl apply -f service-sa-logic.yaml
  ```
### Deploy sa-web-app
- sa-web-app deployment
  ```sh
  $ kubectl apply -f sa-web-app-deployment.yaml
  ```
- sa-web-app load balancer service
  ```sh
  $ kubectl apply -f service-sa-web-app-lb.yaml
  ```
### verify and get the external URL of sa-web-app-lb
  ```sh
  $ kubectl get svc
  NAME             TYPE           CLUSTER-IP       EXTERNAL-IP                    PORT(S)        AGE
  kubernetes       ClusterIP      10.100.0.1       <none>                         443/TCP        16h
  sa-logic         ClusterIP      10.100.90.84     <none>                         80/TCP         5m15s
  sa-web-app-lb    LoadBalancer   10.100.120.236   {sa-web-app-lb external url}   80:30779/TCP   15s
  ```
- Call the web app load balancer external IP
  - `/healthcheck` endpoint
    ```
    $ curl http://{sa-web-app-lb external URL}/healthcheck
    ```
    ```
    {"result":"ok"}%
    ```
  - `/sentiment` endpoint
    ```
    $ curl --header "Content-Type: application/json" \
      --request POST \
      --data '{"sentence":"I am very happy"}' \
      http://{sa-web-app-lb external URL}/sentiment
    ```
    ```
    {"polarity":1,"sentence":"I am very happy"}%
    ```
Note down the sa-web-app-lb service external URL.

### Deploy sa-frontend
#### create the sa-frontend deployment
- rebuild the image giving the sa-web-app-lb external url as build arg
  - replace {sa-web-app-lb external URL} with the URL noted from the calling `kubectl get svc`
  ```sh
  # In the folder where the sa-frontend Dockerfile is
  $ docker build -f Dockerfile -t $DOCKER_USERNAME/sentiment-analysis-frontend-multistage --build-arg SA_WEBAPP_URL=http://{sa-web-app-lb external URL} --build-arg SA_WEBAPP_PORT=80 .
  ```
- push the image
  ```sh
  $ docker push alabebop/sentiment-analysis-frontend-multistage
  ```
- create the sa-frontend deployment
  ```sh
  # In the folder where the manifest for sa-frontend-deployment yaml file is
  $ kubectl apply -f sa-frontend-deployment.yaml
  ```
  - to check the deployment status
  ```sh
  $ kubectl rollout status deployment sa-frontend
  Waiting for deployment "sa-frontend" rollout to finish: 0 of 2 updated replicas are available...
  Waiting for deployment "sa-frontend" rollout to finish: 0 of 2 updated replicas are available...
  Waiting for deployment "sa-frontend" rollout to finish: 1 of 2 updated replicas are available...
  Waiting for deployment "sa-frontend" rollout to finish: 1 of 2 updated replicas are available...
  deployment "sa-frontend" successfully rolled out
  ```
  - verify the pods are deployed
  ```sh
  $ kubectl get pods
  NAME                           READY   STATUS    RESTARTS   AGE
  sa-frontend-69465f4877-2z889   1/1     Running   0          22m
  sa-frontend-69465f4877-45842   1/1     Running   0          22m
  sa-logic-78c77d4ff6-4tmlg      1/1     Running   0          73m
  sa-logic-78c77d4ff6-m6jv4      1/1     Running   0          73m
  sa-web-app-6c6dfbbc75-gpz9h    1/1     Running   0          72m
  sa-web-app-6c6dfbbc75-gqjgm    1/1     Running   0          72m
  ```
#### create the frontend sa-frontend load balancer
```sh
$ kubectl apply -f service-sa-frontend-lb.yaml
```
verify
```sh
$ kubectl get svc
NAME             TYPE           CLUSTER-IP       EXTERNAL-IP            PORT(S)        AGE
kubernetes       ClusterIP      10.100.0.1       <none>                 443/TCP        16h
sa-frontend-lb   LoadBalancer   10.100.50.143    {sa frontend lb url}   80:30688/TCP   62m
sa-logic         ClusterIP      10.100.90.84     <none>                 80/TCP         61m
sa-web-app-lb    LoadBalancer   10.100.120.236   {sa web app lb url}    80:30779/TCP   56m
```
Now you can open the frontend app in browser and see it working on EKS.

# Connect to an External Database 
### Info: How to access a service running on Docker's host computer from a Docker container (Docker for Mac OSX)

Docker for Mac internal URL: `http://host.docker.internal`

To verify:
- Run a minimal Alpine Linux container
  ```
  $ docker run --rm -it alpine sh
  ```
- Add `cURL` to it
  ```
  # apk add curl
  ```
- Run a local python http server on port 8000
  ```
  $ python3 -m http.server 8000
  ```
- Access it from the container
  ```
  # curl http://host.docker.internal:8000
  ``` 

## Locally
### Run a mysql on localhost
- Start mysql
  ```sh
  $ mysql.server start
  ```
- Connect to it
  ```sh
  $ mysql -u root
  ```
- Create a fully priviliged user for local as well as remote
  ```sh
  # create for localhost access
  mysql> create user 'alabebop'@'localhost' identified with mysql_native_password by 'password';
  mysql> grant all on *.* to 'alabebop'@'localhost' with grant option;
  # create for remote access
  mysql> create user 'alabebop'@'%' identified with mysql_native_password by 'password';
  mysql> grant all on *.* to 'alabebop'@'%' with grant option;
  # quit to switch user
  mysql>\q
  ```
- Connect as the new user and create a DB
  ```sh
  $ mysql -u alabebop -ppassword
  # verify user
  mysql> select user();
  +--------------------+
  | user()             |
  +--------------------+
  | alabebop@localhost |
  +--------------------+
  ```
- Create a test database
  ```sql
  -- create a test db
  mysql> create database test_db;
  -- switch to the new db
  mysql> use test_db;
  -- create a test table
  mysql> create table test_table ( id smallint unsigned not null auto_increment, name varchar(20) not null, constraint pk_example primary key (id) );
  -- insert one record
  mysql> insert into test_table (id, name) values (null, 'Sample data');
  -- verify
  mysql> select * from test_table;
  +----+-------------+
  | id | name        |
  +----+-------------+
  |  1 | Sample data |
  +----+-------------+
  1 row in set (0.00 sec)
  ```
- verify database
  ```sql
  mysql> show databases;
  +--------------------+
  | Database           |
  +--------------------+
  | information_schema |
  | mysql              |
  | performance_schema |
  | sys                |
  | test_db            |
  +--------------------+
  ```
- To shut down the local mysql later
  ```sh
  $ mysql.server stop
  ```
### Run a Docker container and access the localhost mysql db from it
- Run a minimal Alpine Linux container and add `curl` and `mysql-client` to it
  ```
  $ docker run --rm -it alpine sh
  # apk add curl
  # apk add mysql-client
  ```
- Connect to the localhost mysql test db from the container
  ```
  # mysql -h host.docker.internal -u alabebop -ppassword test_db
  ```
- Run sql at the localhost mysql database from container
  ```sql
  MySQL [test_db]> select * from test_table;
  +----+-------------+
  | id | name        |
  +----+-------------+
  |  1 | Sample data |
  +----+-------------+
  
  MySQL [test_db]> insert into test_table (id, name) values (null, 'Sample data remote');
  Query OK, 1 row affected (0.002 sec)

  MySQL [test_db]> select * from test_table;
  +----+--------------------+
  | id | name               |
  +----+--------------------+
  |  1 | Sample data        |
  |  2 | Sample data remote |
  +----+--------------------+
  2 rows in set (0.001 sec)
  ```
### Run Docker MySQL locally
Run a mysql Docker image exposing the port 3306
https://hub.docker.com/_/mysql
```sh
# example: docker run --name some-mysql -e MYSQL_ROOT_PASSWORD=my-secret-pw -d mysql:tag
$ docker run -p 3306:3306 --name test-mysql -e MYSQL_ROOT_PASSWORD=password -d mysql:8
```
- To access it locally
  ```sh
  mysql --host 127.0.0.1 --port 3306 --user root --password
  ```
- To access it from another container
  - since we exposed the port 3306, everything works the same as running a local mysql
  ```
  / # mysql -h host.docker.internal -u alabebop -ppassword test_db
  ```

### Use k8s `externalName` to expose the local mysql to the k8s cluster
- create the manifest
  ```yaml
  # service-external-db.yaml
  apiVersion: v1
  kind: Service
  metadata:
    name: database-service
  spec:
    type: ExternalName
    externalName: host.docker.internal # this only works on Docker for Mac
    ports:
      - protocol: TCP
        port: 3306
        name: mysql
      - protocol: TCP
        port: 8000
        name: http
  ```
- bash into a web-app-go pod and test the connection
  ```sh
  $ kubectl exec -it sa-web-app-6c6dfbbc75-4kmts -- /bin/bash
  ```
  - add `dig` by installing the `bind-tool` if you want to use `dig`
    ```
    bash-5.1# apk add --no-cache bind-tools
    ```
  - add `curl` and `mysql-client` to test connection
    ```
    bash-5.1# apk add --no-cache curl mysql-client
    ```
  - test the connection to things running externally on localhost
    - suppose we are running `$ python3 -m http.server 8000`, we can access it from a k8s pod using the `externalName` service `database-service`
      ```
      bash-5.1# curl database-service:8000
      <!DOCTYPE HTML PUBLIC "-//W3C//DTD HTML 4.01//EN" "http://www.w3.org/TR/html4/strict.dtd">
      <html>
      <head>
      ...
      ```
    - same works for the mysql db running on a docker container on the host machine
      ```
      bash-5.1# mysql --host database-service --port 3306 --user alabebop -ppassword test_db
      ...
      MySQL [test_db]> select * from test_table;
      +----+---------------+
      | id | name          |
      +----+---------------+
      |  1 | sample record |
      +----+---------------+
      ```

## On EKS

We will use an EKS cluster on EC2 created earlier. Switch context to use it (your AWS CLI credential must be properly configured to access it).
```sh
# check the context name
$ cat ~/.kube/config
```
Look for something like
```yaml
- context:
    cluster: <your cluster name>.ap-northeast-1.eksctl.io
    user: <your IAM user name>@<your cluster name>.ap-northeast-1.eksctl.io
  name: <your IAM user name>@<your cluster name>.ap-northeast-1.eksctl.io
```
Take the value of the `name` as the context to switch.
```sh
$ kubectl config use-context <your IAM user name>@<your cluster name>.ap-northeast-1.eksctl.io
```
The following part will use [EKS workshop - Security Groups for Pods](https://www.eksworkshop.com/beginner/115_sg-per-pod/) as an reference.
### SGs
#### Prerequisite - a node group using EC2 instances that support pod level SG

Security groups for pods are supported by most Nitro-based Amazon EC2 instance families, including the m5, c5, r5, p3, m6g, c6g, and r6g instance families, but **not the t3 instance family**. 

Create a node group with m5.large instances

- add a manifest yaml file `nodegroup-sec-group.yaml`
  ```yaml
  # nodegroup-sec-group.yaml
  apiVersion: eksctl.io/v1alpha5
  kind: ClusterConfig
  metadata:
    name: eksworkshop-eksctl
    region: ap-northeast-1

  managedNodeGroups:
  - name: nodegroup-sec-group
    desiredCapacity: 1
    instanceType: m5.large
  ```
- use `eksctl` to create 
  ```sh
  $ eksctl create nodegroup -f nodegroup-sec-group.yaml
  ```
  This should take about 5 min, wait until the creation is complete and verify
  ```sh
  $ kubectl get nodes \
  --selector beta.kubernetes.io/instance-type=m5.large
  ```
#### Create SGs
- Export the VPC ID as an ENV
  ```sh
  $ export VPC_ID=$(aws eks describe-cluster \
    --name eksworkshop-eksctl \
    --query "cluster.resourcesVpcConfig.vpcId" \
    --output text)
  ```
- Create RDS security group
  ```sh
  $ aws ec2 create-security-group \
    --description 'RDS SG' \
    --group-name 'RDS_SG' \
    --vpc-id ${VPC_ID}

  {
    "GroupId": "sg-05d8a1aafcc0353a3"
  }

  # save the security group ID for future use
  $ export RDS_SG=$(aws ec2 describe-security-groups \
      --filters Name=group-name,Values=RDS_SG Name=vpc-id,Values=${VPC_ID} \
      --query "SecurityGroups[0].GroupId" --output text)
  ```
- Create Pod SG
  ```sh
  $ aws ec2 create-security-group \
    --description 'POD SG' \
    --group-name 'POD_SG' \
    --vpc-id ${VPC_ID}
  
  {
    "GroupId": "sg-05ef575e7b2519455"
  }

  # save the security group ID for future use
  $ export POD_SG=$(aws ec2 describe-security-groups \
      --filters Name=group-name,Values=POD_SG Name=vpc-id,Values=${VPC_ID} \
      --query "SecurityGroups[0].GroupId" --output text)
  ```

#### Update SGs
- update the node group SG to allow **TCP and UDP on port 53** from pod SG, since pods need this for DNS resolution
  ```sh
  $ export NODE_GROUP_SG=$(aws ec2 describe-security-groups \
      --filters Name=tag:Name,Values=eks-cluster-sg-eksworkshop-eksctl-* Name=vpc-id,Values=${VPC_ID} \
      --query "SecurityGroups[0].GroupId" \
      --output text)
  $ echo "Node Group security group ID: ${NODE_GROUP_SG}"

  # allow POD_SG to connect to NODE_GROUP_SG using TCP 53
  $ aws ec2 authorize-security-group-ingress \
      --group-id ${NODE_GROUP_SG} \
      --protocol tcp \
      --port 53 \
      --source-group ${POD_SG}

  # allow POD_SG to connect to NODE_GROUP_SG using UDP 53
  $ aws ec2 authorize-security-group-ingress \
      --group-id ${NODE_GROUP_SG} \
      --protocol udp \
      --port 53 \
      --source-group ${POD_SG}
  ```

- Launch a jumpbox EC2 instance to populate the RDS, note down its SG
  ```
  export JUMPBOX_SG=sg-0b3f975d4b0da3030
  ```
- Allow the pod SG and the jumpbox SG to access the RDS SG
  ```sh
  # allow Cloud9 to connect to RDS
  aws ec2 authorize-security-group-ingress \
      --group-id ${RDS_SG} \
      --protocol tcp \
      --port 3306 \
      --source-group ${JUMPBOX_SG}

  # Allow POD_SG to connect to the RDS
  aws ec2 authorize-security-group-ingress \
      --group-id ${RDS_SG} \
      --protocol tcp \
      --port 3306 \
      --source-group ${POD_SG}
  ```

### RDS MySQL DB
#### Create DB 
- Create DB subnet group
  - Use the same subnets that the EKS cluster is in
  ```sh
  $ export PUBLIC_SUBNETS_ID=$(aws ec2 describe-subnets \
      --filters "Name=vpc-id,Values=$VPC_ID" "Name=tag:Name,Values=eksctl-eksworkshop-eksctl-cluster/SubnetPublic*" \
      --query 'Subnets[*].SubnetId' \
      --output json | jq -c .)

  # create a db subnet group
  $ aws rds create-db-subnet-group \
      --db-subnet-group-name rds-eksworkshop \
      --db-subnet-group-description rds-eksworkshop \
      --subnet-ids ${PUBLIC_SUBNETS_ID}
  ```
- Create DB
  ```sh
  # get RDS SG ID if not yet
  $ export RDS_SG=$(aws ec2 describe-security-groups \
      --filters Name=group-name,Values=RDS_SG Name=vpc-id,Values=${VPC_ID} \
      --query "SecurityGroups[0].GroupId" --output text)

  # specify a password for RDS
  $ export RDS_PASSWORD=password

  # create RDS MySQL instance
  $ aws rds create-db-instance \
      --db-instance-identifier rds-eksworkshop \
      --db-name test_db \
      --db-instance-class db.t3.micro \
      --engine mysql \
      --db-subnet-group-name rds-eksworkshop \
      --vpc-security-group-ids $RDS_SG \
      --master-username admin \
      --publicly-accessible \
      --master-user-password ${RDS_PASSWORD} \
      --backup-retention-period 0 \
      --allocated-storage 20
  ```
  - It takes some time before the DB instance becomes available
    ```sh
    # Check status and see if it becomes "available"
    $ aws rds describe-db-instances \
    --db-instance-identifier rds-eksworkshop \
    --query "DBInstances[].DBInstanceStatus" \
    --output text
    ```
- Get DB endpoint
  ```sh
  $ export RDS_ENDPOINT=$(aws rds describe-db-instances \
    --db-instance-identifier rds-eksworkshop \
    --query 'DBInstances[0].Endpoint.Address' \
    --output text)

  $ echo "RDS endpoint: ${RDS_ENDPOINT}"
  RDS endpoint: rds-eksworkshop.<xxxxxxxxx>.ap-northeast-1.rds.amazonaws.com
  ```
- Get into the EC2 jumpbox and create some content in the DB
  ```sh
  $ mysql -h rds-eksworkshop.<xxxxxxxxx>.ap-northeast-1.rds.amazonaws.com --port 3306 -u admin -ppassword test_db
  ```
  ```sql
  MySQL [test_db]> show databases;
  +--------------------+
  | Database           |
  +--------------------+
  | information_schema |
  | mysql              |
  | performance_schema |
  | sys                |
  | test_db            |
  +--------------------+

  MySQL [test_db]> create table test_table ( id smallint unsigned not null auto_increment, name varchar(20) not null, constraint pk_example primary key (id) );

  MySQL [test_db]> insert into test_table (id, name) values (null, 'Sample data');
  Query OK, 1 row affected (0.01 sec)

  MySQL [test_db]> select * from test_table;
  +----+-------------+
  | id | name        |
  +----+-------------+
  |  1 | Sample data |
  +----+-------------+
  1 row in set (0.00 sec)
  ```
### CNI (Container Network Interface)
The CNI plugin for k8s assigns an IP address (a secondary ENI in the node instance) from the VPC to each pod. It is deployed with each of EC2 node in a Daemonset with the name `aws-node`. https://docs.aws.amazon.com/eks/latest/userguide/pod-networking.html

Since SG is specified with network interfaces, we are now able to schedule pods requiring specific security groups.

- Add IAM policy to the node group role to allow the EC2 instances to manage network interfaces, their private IP addresses, and attachment and detachment of them.
  `AmazonEKS_CNI_Policy` seems to be already attached to the node group's IAM role
- enable CNI plugin by setting env `ENABLE_POD_ENI=true` in the aws-node `DaemonSet`
  ```sh
  $ kubectl -n kube-system set env daemonset aws-node ENABLE_POD_ENI=true
  # wait for the rolling update of the daemonset
  $ kubectl -n kube-system rollout status ds aws-node
  ```
- The plugin will then add a label `vpc.amazonaws.com/has-trunk-attached=true` to compatible instances
  ```sh
  $ kubectl get nodes \
    --selector alpha.eksctl.io/nodegroup-name=nodegroup-sec-group \
    --show-labels
  ```
### SG for pods
- verify that the cluster has the security group policies CRD (Custom Resource Definition)
  ```sh
  $ kubectl get crd
  NAME                                         CREATED AT
  eniconfigs.crd.k8s.amazonaws.com             2021-08-11T09:56:07Z
  securitygrouppolicies.vpcresources.k8s.aws   2021-08-11T09:56:11Z
  ```
- create pod sg policy
  This policy will attach the POD_SG to pods who have the label `app: green-pod`
  ```yaml
  # sg-policy.yaml
  apiVersion: vpcresources.k8s.aws/v1beta1
  kind: SecurityGroupPolicy
  metadata:
    name: allow-rds-access
  spec:
    podSelector:
      matchLabels:
        app: green-pod
    securityGroups:
      groupIds:
        - sg-05ef575e7b2519455 # $POD_SG
  ```
- create a namespace `sg-per-pod` and deploy the policy in that namespace
  ```sh
  # create a namespace
  $ kubectl create namespace sg-per-pod
  namespace/sg-per-pod created

  # deploy the policy in that namespace
  $ kubectl -n sg-per-pod apply -f sg-policy.yaml
  securitygrouppolicy.vpcresources.k8s.aws/allow-rds-access created

  # describe the policy
  $ kubectl -n sg-per-pod describe securitygrouppolicy
  ```
  This should be in the output
  ```
  Spec:
    Pod Selector:
      Match Labels:
        App:  green-pod
    Security Groups:
      Group Ids:
        sg-05ef575e7b2519455
  ```

### store RDS credentials in a k8s secret
```sh
$ export RDS_PASSWORD=password
$ export RDS_ENDPOINT=$(aws rds describe-db-instances \
    --db-instance-identifier rds-eksworkshop \
    --query 'DBInstances[0].Endpoint.Address' \
    --output text)

$ kubectl create secret generic rds \
    --namespace=sg-per-pod \
    --from-literal="password=${RDS_PASSWORD}" \
    --from-literal="host=${RDS_ENDPOINT}"
secret/rds created
```
Verify:
```sh
$ kubectl -n sg-per-pod describe secret rds
Name:         rds
Namespace:    sg-per-pod
Labels:       <none>
Annotations:  <none>

Type:  Opaque

Data
====
host:      61 bytes
password:  8 bytes
```
### deploy pods
#### green-pods
- Create the green-pod deployment yaml based on the `sa-web-app-deployment.yaml`
  - update relevant sections
    ```yaml
    # sa-web-app-green-pod-deployment.yaml
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: green-pod
      labels:
        app: green-pod
      namespace: sg-per-pod
    spec:
      selector:
        matchLabels:
          app: green-pod
      ...
      template:
        metadata:
          labels:
            app: green-pod
        spec:
          affinity:
            nodeAffinity:
              requiredDuringSchedulingIgnoredDuringExecution:
                nodeSelectorTerms:
                - matchExpressions:
                  - key: "vpc.amazonaws.com/has-trunk-attached"
                    operator: In
                    values:
                      - "true"
          containers:
            - image: alabebop/sentiment-analysis-webapp-multistage
              imagePullPolicy: Always
              name: green-pod
              env:
                ...
                # DB related
                - name: HOST
                  valueFrom:
                    secretKeyRef:
                      name: rds
                      key: host
                - name: DBNAME
                  value: test_db
                - name: USER
                  value: admin
                - name: PASSWORD
                  valueFrom:
                    secretKeyRef:
                      name: rds
                      key: password
      ```
- deploy it to the `sg-per-pod` namespace
  ```sh
  $ kubectl -n sg-per-pod apply -f sa-web-app-green-pod-deployment.yaml
  ```
  ```sh
  $ kubectl -n sg-per-pod get pods
  NAME                         READY   STATUS    RESTARTS   AGE
  green-pod-7895c5dfc6-qqbrf   1/1     Running   0          7m59s
  ```

- get a shell of the pod and verify RDS access
  ```sh
  $ kubectl -n sg-per-pod exec -it green-pod-7895c5dfc6-qqbrf -- /bin/bash
  ```
  - verify ENVs set in the deployment
    ```
    bash-5.1# echo $USER
    admin
    bash-5.1# echo $DBNAME
    test_db
    ```
  - verify RDS read and write access
    ```
    bash-5.1# mysql -h ${HOST} --port 3306 -u ${USER} -p${PASSWORD} ${DB_NAME}
    
    MySQL [test_db]> insert into test_table (id, name) values (null, 'from EKS');

    MySQL [test_db]> select * from test_table;
    +----+-------------+
    | id | name        |
    +----+-------------+
    |  1 | Sample data |
    |  2 | from EKS    |
    +----+-------------+
    ```
#### red-pods
- Do the same as green-pod but replace `green-pod` with `red-pod` and deploy a red pod in the same namespace.

  ```sh
  $ kubectl -n sg-per-pod apply -f sa-web-app-red-pod-deployment.yaml
  ```
- get the red pod
  ```sh
  $ kubectl -n sg-per-pod get pods
  NAME                         READY   STATUS    RESTARTS   AGE
  green-pod-7895c5dfc6-qqbrf   1/1     Running   0          25m
  red-pod-975c49d4d-f6k6n      1/1     Running   0          36s
  ```
- get a shell in the red pod and verify it has the same config but doesn't have access to RDS
  ```sh
  $ kubectl -n sg-per-pod exec -it red-pod-975c49d4d-f6k6n -- /bin/bash
  ```
  ```
  bash-5.1# echo $HOST
  <this works>
  bash-5.1# mysql -h ${HOST} --port 3306 -u ${USER} -p${PASSWORD} ${DB_NAME}
  ERROR 2002 (HY000): Can't connect to MySQL server on 'rds-eksworkshop.<xxxxxxxxxx>.ap-northeast-1.rds.amazonaws.com' (115)
  <this does not work, timeout>
  ```
