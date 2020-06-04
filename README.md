# redis-operator

## Overview

Redis operator build a **Highly Available Redis cluster with Sentinel** atop Kubernetes.
Using this operator you can create a Redis deployment that resists without human intervention to certain kind of failures.

The operator itself is built with the [Operator framework](https://github.com/operator-framework/operator-sdk).

It inspired by [spotahome/redis-operator](https://github.com/spotahome/redis-operator).

![Redis Cluster atop Kubernetes](/static/redis-sentinel-readme.png)

* Create a statefulset to mange Redis instances (masters and replicas), each redis instance has default PreStop script that can do failover if master is down.
* Create a statefulset to mange Sentinel instances that will control the Redis nodes, each Sentinel instance has default ReadinessProbe script to detect whether the current sentinel's status is ok. When a sentinel pod is not ready, it is removed from Service load balancers.
* Create a Service and a Headless service for Sentinel statefulset.
* Create a Headless service for Redis statefulset.

Table of Contents
=================

   * [redis-operator](#redis-operator)
      * [Overview](#overview)
      * [Prerequisites](#prerequisites)
      * [Features](#features)
      * [Quick Start](#quick-start)
         * [Deploy redis operator](#deploy-redis-operator)
         * [Deploy a sample redis cluster](#deploy-a-sample-redis-cluster)
            * [Resize an Redis Cluster](#resize-an-redis-cluster)
            * [Create redis cluster with password](#create-redis-cluster-with-password)
            * [Dynamically changing redis config](#dynamically-changing-redis-config)
            * [Persistence](#persistence)
            * [Custom SecurityContext](#custom-securitycontext)
         * [Cleanup](#cleanup)
      * [Automatic failover details](#automatic-failover-details)

## Prerequisites

* go version v1.13+.
* Access to a Kubernetes v1.13.10+ cluster.

## Features
In addition to the sentinel's own capabilities, redis-operator can:

* Push events and update status to the Kubernetes when resources have state changes
* Deploy redis operator  watches and manages resources in a single namespace or cluster-wide
* Create redis cluster with password
* Dynamically changing redis config
* False delete automatic recovery
* Persistence
* Custom SecurityContext

## Quick Start

### Deploy redis operator

Register the RedisCluster custom resource definition (CRD).
```
$ kubectl create -f deploy/crds/redis_v1beta1_rediscluster_crd.yaml
```

A namespace-scoped operator watches and manages resources in a single namespace, whereas a cluster-scoped operator watches and manages resources cluster-wide.
You can chose run your operator as namespace-scoped or cluster-scoped.
```
// cluster-scoped
$ kubectl create -f deploy/service_account.yaml
$ kubectl create -f deploy/cluster/cluster_role.yaml
$ kubectl create -f deploy/cluster/cluster_role_binding.yaml
$ kubectl create -f deploy/cluster/operator.yaml

// namespace-scoped
$ kubectl create -f deploy/service_account.yaml
$ kubectl create -f deploy/namespace/role.yaml
$ kubectl create -f deploy/namespace/role_binding.yaml
$ kubectl create -f deploy/namespace/operator.yaml
```

Verify that the redis-operator is up and running:
```
$ kubectl get deployment
NAME             READY   UP-TO-DATE   AVAILABLE   AGE
redis-operator   1/1     1            1           65d
```

### Deploy a sample redis cluster
```
$ cat deploy/cluster/redis_v1beta1_rediscluster_cr.yaml
apiVersion: redis.kun/v1beta1
kind: RedisCluster
metadata:
  annotations:
    # if your operator run as cluster-scoped, add this annotations
    redis.kun/scope: cluster-scoped
  name: test
spec:
  # Add fields here
  size: 3

kubectl apply -f deploy/cluster/redis_v1beta1_rediscluster_cr.yaml
if you run operator as namespace-scoped, do:
kubectl apply -f deploy/namespace/redis_v1beta1_rediscluster_cr.yaml
```

Verify that the cluster instances and its components are running.
```
$ kubectl get rediscluster
NAME   SIZE   STATUS    AGE
test   3      Healthy   4m9s

$ kubectl get all -l app.kubernetes.io/managed-by=redis-operator
NAME                        READY   STATUS    RESTARTS   AGE
pod/redis-cluster-test-0    1/1     Running   0          4m16s
pod/redis-cluster-test-1    1/1     Running   0          3m22s
pod/redis-cluster-test-2    1/1     Running   0          2m40s
pod/redis-sentinel-test-0   1/1     Running   0          4m16s
pod/redis-sentinel-test-1   1/1     Running   0          81s
pod/redis-sentinel-test-2   1/1     Running   0          18s

NAME                                   TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)     AGE
service/redis-cluster-test             ClusterIP   None            <none>        6379/TCP    4m16s
service/redis-sentinel-headless-test   ClusterIP   None            <none>        26379/TCP   4m16s
service/redis-sentinel-test            ClusterIP   10.22.22.34     <none>        26379/TCP   4m16s

NAME                                   READY   AGE
statefulset.apps/redis-cluster-test    3/3     4m16s
statefulset.apps/redis-sentinel-test   3/3     4m16s
```

* redis-cluster-<NAME>: Redis statefulset
* redis-sentinel-<NAME>: Sentinel statefulset
* redis-sentinel-<NAME>: Sentinel service
* redis-sentinel-headless-<NAME>: Sentinel headless service
* redis-cluster-<NAME>: Redis headless service

Describe the Redis Cluster, Viewing Events and Status
```
$ kubectl describe redisclusters test

Name:         test
Namespace:    default
Labels:       <none>
Annotations:  redis.kun/scope: cluster-scoped
API Version:  redis.kun/v1beta1
Kind:         RedisCluster
Metadata:
  Generation:          1
  UID:                 ec0c3be4-b9c5-11e9-8191-6c92bfb35d2e
Spec:
  Image:              redis:5.0.4-alpine
  Resources:
    Limits:
      Cpu:     400m
      Memory:  300Mi
    Requests:
      Cpu:     100m
      Memory:  50Mi
  Size:        3
Status:
  Conditions:
    Last Transition Time:  2019-08-08T10:21:14Z
    Last Update Time:      2019-08-08T10:22:14Z
    Message:               Cluster ok
    Reason:                Cluster available
    Status:                True
    Type:                  Healthy
    Last Transition Time:  2019-08-08T10:18:53Z
    Last Update Time:      2019-08-08T10:18:53Z
    Message:               Bootstrap redis cluster
    Reason:                Creating
    Status:                True
    Type:                  Creating
Events:
  Type    Reason        Age                    From            Message
  ----    ------        ----                   ----            -------
  Normal  Creating      3m22s                  redis-operator  Bootstrap redis cluster
  Normal  Ensure        2m12s (x8 over 3m22s)  redis-operator  Makes sure of redis cluster ready
  Normal  CheckAndHeal  2m12s (x8 over 3m22s)  redis-operator  Check and heal the redis cluster problems
  Normal  Updating      2m12s (x8 over 3m22s)  redis-operator  wait for all redis server start
```

#### Resize an Redis Cluster
The initial cluster size is 3. Modify the file and change size from 3 to 5.
```
$ cat deploy/crds/redis_v1beta1_rediscluster_cr.yaml
apiVersion: redis.kun/v1beta1
kind: RedisCluster
metadata:
  annotations:
    # if your operator run as cluster-scoped, add this annotations
    redis.kun/scope: cluster-scoped
  name: test
spec:
  # Add fields here
  size: 5

kubectl apply -f deploy/crds/redis_v1beta1_rediscluster_cr.yaml
```

The Redis Cluster will scale to 5 members(1 Master with 4 Slaves).

#### Create redis cluster with password

You can setup redis with auth by set `spec.password`.

```
apiVersion: redis.kun/v1beta1
kind: RedisCluster
metadata:
  annotations:
    # if your operator run as cluster-scoped, add this annotations
    redis.kun/scope: cluster-scoped
  name: test
  namespace: default
spec:
  # custom password (null to disable)
  password: asdfsdf
  # custom configurations
  config:
    hz: "10"
    loglevel: verbose
    maxclients: "10000"
  image: redis:5.0.4-alpine
  resources:
    limits:
      cpu: 400m
      memory: 300Mi
    requests:
      cpu: 100m
      memory: 50Mi
  size: 3
```

#### Dynamically changing redis config

If the custom configurations is changed, the operator will use `config set` cmd apply the changes to the redis node without the need of reload the redis node.

```
apiVersion: redis.kun/v1beta1
kind: RedisCluster
metadata:
  annotations:
    # if your operator run as cluster-scoped, add this annotations
    redis.kun/scope: cluster-scoped
  name: test
  namespace: default
spec:
  # custom password (null to disable)
  password: asdfsdf
  # change the configurations
  config:
    hz: "12"
    loglevel: debug
    maxclients: "10000"
  image: redis:5.0.4-alpine
  resources:
    limits:
      cpu: 400m
      memory: 300Mi
    requests:
      cpu: 100m
      memory: 50Mi
  size: 3
```

#### Persistence

The operator has the ability of add persistence to Redis data. By default an emptyDir will be used, so the data is not saved.

In order to have persistence, a PersistentVolumeClaim usage is allowed.

The `spec.disablePersistence:false` flag can automatically configures the persistence parameters.

```
apiVersion: redis.kun/v1beta1
kind: RedisCluster
metadata:
  name: test
  # if your operator run as cluster-scoped, add this annotations
  annotations:
    redis.kun/scope: "cluster-scoped"
  namespace: default
spec:
  image: redis:5.0.4-alpine
  resources:
    limits:
      cpu: 400m
      memory: 300Mi
    requests:
      cpu: 50m
      memory: 30Mi
  size: 3

  # when the disablePersistence set to false, the following configurations will be set automatically:

  # disablePersistence: false
  # config["save"] = "900 1 300 10"
  # config["appendonly"] = "yes"
  # config["auto-aof-rewrite-min-size"] = "536870912"
  # config["repl-diskless-sync"] = "yes"
  # config["repl-backlog-size"] = "62914560"
  # config["aof-load-truncated"] = "yes"
  # config["stop-writes-on-bgsave-error"] = "no"

  # when the disablePersistence set to true, the following configurations will be set automatically:

  # disablePersistence: true
  # config["save"] = ""
  # config["appendonly"] = "no"
  storage:
    # By default, the persistent volume claims will be deleted when the Redis Cluster be delete.
    # If this is not the expected usage, a keepAfterDeletion flag can be added under the storage section
    keepAfterDeletion: true
    persistentVolumeClaim:
      metadata:
        name: test
      spec:
        accessModes:
        - ReadWriteOnce
        resources:
          requests:
            storage: 1Gi
        storageClassName: sc-rbd-x5
        volumeMode: Filesystem
```

#### Custom SecurityContext

You can change `net.core.somaxconn`(default is 128) by uses the pod securityContext to set unsafe sysctls net.core.somaxconn.

```
apiVersion: redis.kun/v1beta1
kind: RedisCluster
metadata:
  annotations:
    # if your operator run as cluster-scoped, add this annotations
    redis.kun/scope: cluster-scoped
  name: test
spec:
  # Add fields here
  size: 3
  securityContext:
      sysctls:
      - name: net.core.somaxconn
        value: "1024"
```

### Cleanup

```
$ kubectl delete -f deploy/cluster/redis_v1beta1_rediscluster_cr.yaml
$ kubectl delete -f deploy/cluster/operator.yaml
$ kubectl delete -f deploy/cluster/cluster_role.yaml
$ kubectl delete -f deploy/cluster/cluster_role_binding.yaml
$ kubectl delete -f deploy/service_account.yaml
$ kubectl delete -f deploy/crds/redis_v1beta1_rediscluster_crd.yaml

or:
$ kubectl delete -f deploy/namespace/redis_v1beta1_rediscluster_cr.yaml
$ kubectl delete -f deploy/namespace/operator.yaml
$ kubectl delete -f deploy/namespace/role.yaml
$ kubectl delete -f deploy/namespace/role_binding.yaml
$ kubectl delete -f deploy/service_account.yaml
$ kubectl delete -f deploy/crds/redis_v1beta1_rediscluster_crd.yaml
```

## Automatic failover details

Redis-operator build a **Highly Available Redis cluster with Sentinel**, Sentinel always checks the MASTER and SLAVE
instances in the Redis cluster, checking whether they working as expected. If sentinel detects a failure in the
MASTER node in a given cluster, Sentinel will start a failover process. As a result, Sentinel will pick a SLAVE
instance and promote it to MASTER. Ultimately, the other remaining SLAVE instances will be automatically reconfigured
to use the new MASTER instance.

operator guarantees the following:
* Only one Redis instance as master in a cluster
* Number of Redis instance(masters and replicas) is equal as the set on the RedisCluster specification
* Number of Sentinels is equal as the set on the RedisCluster specification
* All Redis slaves have the same master
* All Sentinels point to the same Redis master
* Sentinel has not dead nodes

But Kubernetes pods are volatile, they can be deleted and recreated, and pods IP will change when pod be recreated,
and also, the IP will be recycled and redistributed to other pods.
Unfortunately, sentinel cannot delete the sentinel list or redis list in its memory when the pods IP changes.
This can be caused because there’s no way of a Sentinel node to self-deregister from the Sentinel Cluster before die,
provoking the Sentinel node list to increase without any control.

To ensure that Sentinel is working properly, operator will send a **RESET(SENTINEL RESET * )** signal to Sentinel node
one by one (if no failover is being running at that moment).
`SENTINEL RESET mastername` command: they'll refresh the list of replicas within the next 10 seconds, only adding the
ones listed as correctly replicating from the current master INFO output.
During this refresh time, `SENTINEL slaves <master name>` command can not get any result from sentinel, so operator sent
RESET signal to Sentinel one by one and wait sentinel status became ok(monitor correct master and has slaves).
Additional, Each Sentinel instance has default ReadinessProbe script to detect whether the current sentinel's status is ok.
When a sentinel pod is not ready, it is removed from Service load balancers.
Operator also create a headless svc for Sentinel statefulset, if you can not get result from `SENTINEL slaves <master name>` command,
You can try polling the headless domain.
