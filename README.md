# Kubectl-Safe-Drain
A kubectl plug-in, according to the update strategy, safe drain node and dispatch pod to other available nodes.

# Summary / [中文介绍](https://github.com/maxzhang1985/kubectl-safe-drain/blob/master/README.md "中文介绍")

# Why the Needed
You can use kubectl drain to safely evict all of your pods from a node before you perform maintenance on the node (e.g. kernel upgrade, hardware maintenance, etc.). Safe evictions allow the pod’s containers to gracefully terminate and will respect the PodDisruptionBudgets you have specified.

But kubectl drain has a problem , that  it forces the pod to be evicted or delete, instead of the update strategy defined by the resource such as replicas strategy .

# An example
There is a deployment resource :
```yml
 replicas: 2
 strategy:
     rollingUpdate:
     maxSurge: 1
     maxUnavailable: 0
 type: RollingUpdate
```

The number of resource copies is 3, and rolling update is adopted. One pod is started before deleting the old pod (the maximum unavailable is 0, and the minimum available is 2)

At present, there are two worker nodes in the cluster, one of which is scheduled with two pod, the other is scheduled with one pod.

Suppose node1 is running POD1 and POD3, and the node2 is running POD2.

At this time,kubectl drain NODE1 will show that there is only one pod available for deployment.

# Worse, it's all on the nodes that need to be maintained
It will be a disaster to execute drain at this time. Before the new pod starts, the deployment cannot provide external services. The recovery time depends on the startup speed of the new pod.

# What's the difference with Pod Disruption Budget?
Pod Disruption Budget will only protect pod from being evicted, and will not help it rebuild on other available nodes.

It prevents service unavailability, but it still requires manual migration of pod

# The principle of kubectl safe drain
* Mark nodes as non schedulable (kubectl cordon).
* Locate the deployment and statefullset resources on this node.
* Modify Pod Template and save that rebuild to other nodes.

# Support safe migration of resources
1. Deployment
2. StatefulSet

# Demo
Deployment:
```yml
spec:
    replicas: 2
strategy:
    type: RollingUpdate
    rollingUpdate:
        maxSurge: 1
        maxUnavailable: 0
```







## Has two pods
![](https://cdn.jsdelivr.net/gh/majian159/blogs@master/images/2020_04_29_19_42_iaR3Cs%20.jpg)
![](https://cdn.jsdelivr.net/gh/majian159/blogs@master/images/2020_04_29_19_42_Nb8NZA%20.png)

 ## Execute safe-drain

![](https://cdn.jsdelivr.net/gh/majian159/blogs@master/images/2020_04_29_19_43_xc2Jhz%20.png)

## View deployment changes
![](https://cdn.jsdelivr.net/gh/majian159/blogs@master/images/2020_04_29_19_43_lmDtYv%20.png)

## View pod changes
![](https://cdn.jsdelivr.net/gh/majian159/blogs@master/images/2020_04_29_19_43_Nd6lPE%20.png)


# Install
## Linux Binary
```shell script
curl -sLo sdrain.tgz https://github.com/majian159/kubectl-safe-drain/releases/download/v0.0.1-preview1/kubectl-safe-drain_0.0.1-preview1_linux_amd64.tar.gz \
&& tar xf sdrain.tgz \
&& rm -f sdrain.tgz \
&& mv kubectl-safe-drain /usr/local/bin/kubectl-safe_drain
```
## MacOS Binary
```shell script
curl -sLo sdrain.tgz https://github.com/majian159/kubectl-safe-drain/releases/download/v0.0.1-preview1/kubectl-safe-drain_0.0.1-preview1_darwin_amd64.tar.gz \
&& tar xf sdrain.tgz \
&& rm -f sdrain.tgz \
&& mv kubectl-safe-drain /usr/local/bin/kubectl-safe_drain
```
## Windows Binary
https://github.com/majian159/kubectl-safe-drain/releases/download/v0.0.1-preview1/kubectl-safe-drain_0.0.1-preview1_windows_amd64.tar.gz

## Krew Install
```shell script
curl -O https://raw.githubusercontent.com/majian159/kubectl-safe-drain/master/krew.yaml \
&& kubectl krew install --manifest=krew.yaml \
&& rm -f krew.yaml
```

# How to use
```sh
kubectl safe-drain NODE
```

# TODO

* Node selector , such as annotationsֵ or labels of Kubernetes-Pod 
* Output more friendly prompt information









