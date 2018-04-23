# consul-sidekick

Automatic peer management for [Consul](https://www.consul.io/) in
[Kubernetes](https://kubernetes.io/).

consul-sidekick is designed to run as a sidecar container in each Consul pod. It
obtains the list of peer pods from the Kubernetes API Server and periodically
syncs the Consul pod accordingly.

Advantages:
* Uniform deployment. All instances of Consul are deployed identically. No need
  to worry about the `-bootstrap` flag.
* Consul instances are treated as [_cattle_](http://cloudscaling.com/blog/cloud-computing/the-history-of-pets-vs-cattle/), avoiding the [limitations of StatefulSets](https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/#limitations).  
* No more stale peers when pods are replaced or deleted.
* No more bootstrapping issues. No need for an external bootstrapping service
  like [Atlas](https://www.consul.io/docs/guides/atlas.html) (now deprecated).

## Install

See an [example](/examples) of how to use it.

## Limitations

For now it assumes that the Consul pods are controlled by a
[ReplicaSet](https://kubernetes.io/docs/concepts/workloads/controllers/replicaset/).
However, it should be easy to extend to other controllers if needed.

## <a name="help"></a>Getting Help

If you have any questions about, feedback for or problems with `consul-sidekick`:

- Invite yourself to the <a href="https://weaveworks.github.io/community-slack/" target="_blank"> #weave-community </a> slack channel.
- Ask a question on the <a href="https://weave-community.slack.com/messages/general/"> #weave-community</a> slack channel.
- Send an email to <a href="mailto:weave-users@weave.works">weave-users@weave.works</a>
- <a href="https://github.com/weaveworks/consul-sidekick/issues/new">File an issue.</a>

Your feedback is always welcome!
