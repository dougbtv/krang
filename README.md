# krang

![krang logo, dude.](https://github.com/dougbtv/krang/blob/main/doc/krang-logo.png)

A Kubernetes-enabled CNI.next runtime. It enables CNI actions during pod lifecycle, and maybe gives a little better bindings to K8s.

It can mutate stuff, dude.

## What's it do?

`krang` is a Kubernetes-enabled CNI runtime that enables dynamic network operations throughout pod lifecycle. Whether you’re installing CNI plugins at runtime, mutating active network namespaces, or extending how networking behaves across your cluster, krang gives you fine-grained, programmable control over CNI behavior -- right from the Kubernetes control plane.

For example -- say you want to do something to pod networking AFTER all the pods are up. Sure, you could restart them all, but do you want to? Nah. What if you could essentially CNI `UPDATE` them, and request the update with your k8s-enabled CNI plugin?

krang comes in two parts: A node-local daemon that can delegate CNI calls, and `krangctl` to make it a little easier to use.

Inspired by [the technorganic villain](https://en.wikipedia.org/wiki/Krang), krang doesn’t *do* the fighting, krang puppeteers the exosuit. Likewise, this daemon doesn't replace your plugins -- it orchestrates them. Think of it as the brains behind your CNI plugins brawn: coordinating plugin installs, executing mutations, and enabling on-the-fly changes to your pod networks -- all without leaving the comfort of Kubernetes.

Maybe it can inspire some thinking about the next generation of CNI and its integration with Kubernetes. But guess what? krang doesn't require anything special or any mods to Kubernetes itself.

Now with more pizza 🍕 and less other stuff.

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

## Outstanding stuff.

* Basically everything.
* Only does conflists.
* `$(more.)`

## Further ideas.

* A community-led ecosystem complete with:
  * Community-defined CRDs
  * CNI.next sample plugins
* Netlink listener in netns'es with metadata reflection in k8s objects 
  * `network-status` annotation, or maybe DRA resourceclaim status, perhaps?
* DRA integration, or a DRA framework for CNI developers.
* Krang-as-primary-CNI, complete with chaining configuration.
* New optional YAML configurations for CNI, along with validation.
