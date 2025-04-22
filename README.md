# krang

![krang logo, dude.](https://github.com/dougbtv/krang/blob/main/doc/krang-logo.png)

A Kubernetes-enabled CNI.next runtime. It enables CNI actions during pod lifecycle, and maybe enables some better K8s bindings for CNI, making it easier to write k8s-enabled CNI plugins.

It can mutate your pods' network namespaces, dude.

## What's it do?

`krang` is a Kubernetes-enabled CNI runtime that enables dynamic network operations throughout pod lifecycle. Whether you’re installing CNI plugins at runtime, mutating active network namespaces, or extending how networking behaves across your cluster, krang gives you fine-grained, programmable control over CNI behavior -- right from the Kubernetes control plane.

For example -- say you want to do something to pod networking AFTER all the pods are up. Sure, you could restart them all, but do you want to? Nah. What if you could essentially CNI `UPDATE` them, and request the update with your k8s-enabled CNI plugin?

Check out the demo on asciinema:

[![asciicast](https://asciinema.org/a/DSNTIQIg5VM2mGh5oFK7YlGUW.svg)](https://asciinema.org/a/DSNTIQIg5VM2mGh5oFK7YlGUW)

krang comes in two parts: A node-local daemon that can delegate CNI calls, and `krangctl` to make it a little easier to use.

Inspired by [the technorganic villain](https://en.wikipedia.org/wiki/Krang), krang doesn’t *do* the fighting, krang puppeteers the exosuit and finances the foot soldiers. Likewise, this daemon doesn't replace your plugins -- it orchestrates them. Think of it as the brains behind your CNI plugins' brawn: coordinating plugin installs, executing mutations, and enabling on-the-fly changes to your pod networks -- all without leaving the comfort of Kubernetes.

Maybe it can inspire some thinking about the next generation of CNI and its integration with Kubernetes. But guess what? **krang doesn't require any mods to Kubernetes itself**! It works right out of the box with vanilla k8s.

Now with more pizza 🍕 and less other stuff.

## Did you know?

CNI is "container orchestration agnostic" -- it doesn't have a bias towards any one container orchestration system. Should that be the case? Kubernetes developers want to develop on the k8s API, maybe we need some common ground between k8s and CNI.

## Installing `krangctl`

Cowboy style (installs to `/usr/local/bin`, so you need privs to write there):

```bash
curl -sSfL https://raw.githubusercontent.com/dougbtv/krang/main/getkrang.sh | bash
```

The "*I might skim the script*"-style:

```bash
git clone https://github.com/dougbtv/krang.git
cd krang
./getkrang.sh
```

Or you can `go install` it:

```bash
go install github.com/dougbtv/krang/cmd/krangctl@latest
```

Try:

```
krangctl --help
```

## Installing `krang` on Kubernetes.

First clone the repo...

```
git clone https://github.com/dougbtv/krang.git && cd krang
```

Then:

```
kubectl apply \
  -f manifests/crd/k8s.cni.cncf.io_cnimutationrequests.yaml \
  -f manifests/crd/k8s.cni.cncf.io_cnipluginregistrations.yaml \
  -f manifests/daemonset.yml
```

## Demo.

Requirements:

* `krangctl` installed.
* a clone of this repo (and it's your current directory).
* and `kind`.

```bash
# Create a few pods.
kubectl create -f manifests/testing/replicaset.yml
# Check their sysctls.
kubectl exec $(kubectl get pods | grep "demotuning" | head -n1 | awk '{print $1}') -- sysctl -n net.ipv4.conf.eth0.arp_filter
# Install tuning CNI and passthru CNI (it's for an empty head to exec a CNI chain on top of)
krangctl create --binary-path /cni-plugins/bin/tuning --cni-type tuning --name tuning --image "quay.io/dosmith/cni-plugins:v1.6.2a"
krangctl create --binary-path /usr/src/multus-cni/bin/passthru --cni-type passthru --name passthru --image "ghcr.io/k8snetworkplumbingwg/multus-cni:snapshot-thick"
# Show the installed plugins.
watch krangctl get
# Start a mutation request
kubectl create -f manifests/testing/mutation-request.yml
# Check the logs, if you must.
# Now see the mutated sysctl!
kubectl exec $(kubectl get pods | grep "demotuning" | head -n1 | awk '{print $1}') -- sysctl -n net.ipv4.conf.eth0.arp_filter
```

You can use `krangctl` like this:

```
$ ./krangctl create --binary-path /usr/src/bin/cni/macvlan --cni-type macvlan --name macvlan --image "quay.io/dosmith/cni-plugins:v1.6.2"
$ ./krangctl get 
kube-system/macvlan
  - kind-worker2: ready (ready: true)
  - kind-worker: ready (ready: true)
  - kind-control-plane: ready (ready: true)
```

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
  * Representation of CNI configurations on disk, e.g. `krangctl get nodeconfig` or something.
* New optional YAML configurations for CNI, along with validation.
* Enable further observability, expose prom metrics, show history of actions against a netns?
* Enable a way for k8s-enabled CNI plugins to compute everything prior to pod creation (e.g. at schedule time, or maybe prepare device time in DRA, stuff like that)
  * So CNI doesn't have to `stop the world` (and [melt with you](https://www.youtube.com/watch?v=LuN6gs0AJls))
* CNI execution in containers.
  * Execute terminal CNI plugins in containers and collect the logs?
  * This breaks my brain, a little, but there's something here.
* CNI Cache
  * Need to keep output, and decide how to use it
  * It also uses the existing `ADD` cache, is that the right approach? Could be done other ways.