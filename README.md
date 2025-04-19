# krang

![krang logo, dude.](https://github.com/dougbtv/krang/blob/main/doc/krang-logo.png)

An Kubernetes-enabled CNI.next runtime.

It can mutate stuff.

## `krangctl`

You can use `krangctl` like this:

```
$ ./krangctl create --binary-path /usr/src/bin/cni/macvlan --cni-type macvlan --name macvlan --image "quay.io/dosmith/cni-plugins:v1.6.2"
$ ./krangctl get 
kube-system/macvlan
  - kind-worker2: ready (ready: true)
  - kind-worker: ready (ready: true)
  - kind-control-plane: ready (ready: true)
```

## Demo.

```
./krangctl create --binary-path /usr/src/multus-cni/bin/passthru --cni-type passthru --name passthru --image "quay.io/dosmith/multus-thick:cnisubdirA"
```