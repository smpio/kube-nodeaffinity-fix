# kube-nodeaffinity-fix

Watches Kubernetes API server for changed Pods and deletes them if `.status.phase = 'Failed'` and `.status.reason = 'NodeAffinity'`.

This is temporary workaround for [issue #98534](https://github.com/kubernetes/kubernetes/issues/98534) with preemptible nodes.


## Usage

```
  -kubeconfig string
    	path to kubeconfig file
  -master string
    	kubernetes api server url
```
