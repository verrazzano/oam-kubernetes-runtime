# Build Instructions

The base tag this release is branched from is `v0.3.0`


Create Environment Variables

```
export DOCKER_REPO=<Docker Repository>
export DOCKER_NAMESPACE=<Docker Namespace>
export DOCKER_TAG=v0.3.0-OL
```

Build and Push Images

```
# Build and push OAM Kubernetes Runtim

docker build -t ${DOCKER_REPO}/${DOCKER_NAMESPACE}/oam-kubernetes-runtime:${DOCKER_TAG} .
docker push ${DOCKER_REPO}/${DOCKER_NAMESPACE}/oam-kubernetes-runtime:${DOCKER_TAG}
```
