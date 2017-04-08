# consul-sidekick

Automatic peer management for Consul in [Consul](https://www.consul.io/) in
[Kubernetes](https://kubernetes.io/).

consul-sidekick is designed to run as a sidecar container to each Consul pod. It
obtains the list of peer pods from the Kubernetes API Server and periodically
syncs the Consul pod accordingly.

Advantages:
* Uniform deployment. All instances of Consul are deployed identically. No need
  to worry about the `-bootstrap` flag.
* No more stale peers when pods are replaced or deleted.
* No more bootstrapping issues. No need for an external bootstrapping service
  like [Atlas](https://www.consul.io/docs/guides/atlas.html) (now deprecated).

## Install

See an [example](/examples) on how to use it.

## Limitations

For now it assumes that the Consul pods are controlled by a
[ReplicaSet](https://kubernetes.io/docs/concepts/workloads/controllers/replicaset/).
However, it should be easy to extend to other controllers if needed.
