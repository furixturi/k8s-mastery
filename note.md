# What is this
https://www.freecodecamp.org/news/learn-kubernetes-in-under-3-hours-a-detailed-guide-to-orchestrating-containers-114ff420e882/

# Step 1: Locally get the three apps up and running and working together

## sa-frontend
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


# Step 2 dockerize
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
# Kubernetes
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

## create `deployment`
About the `apiVersion: apps/v1` in `sa-frontend-deployment.yaml`: https://matthewpalmer.net/kubernetes-app-developer/articles/kubernetes-apiversion-definition-guide.html

### create a rolling deployment
```sh
$ kubectl apply -f sa-frontend-deployment.yaml
```
Verify: 

```sh
$ kubectl get pods --show-labels
NAME                           READY   STATUS    RESTARTS   AGE     LABELS
sa-frontend                    1/1     Running   0          128m    app=sa-frontend
sa-frontend-69465f4877-55zx6   1/1     Running   0          5m48s   app=sa-frontend,pod-template-hash=69465f4877
sa-frontend-69465f4877-pwxl5   1/1     Running   0          5m48s   app=sa-frontend,pod-template-hash=69465f4877
```

Delete the sa-frontend, which was created earlier separately.
```
$ kubectl delete pod sa-frontend
```
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
```
$ kubectl apply -f sa-logic-deployment.yaml
```

### create the service
```
$ kubectl apply -f service-sa-logic.yaml
```
Verify
```
$ kubectl get svc
NAME             TYPE           CLUSTER-IP       EXTERNAL-IP   PORT(S)        AGE
kubernetes       ClusterIP      10.96.0.1        <none>        443/TCP        14h
sa-frontend-lb   LoadBalancer   10.98.205.78     <pending>     80:31820/TCP   3h11m
sa-logic         ClusterIP      10.108.232.130   <none>        80/TCP         118s
```
