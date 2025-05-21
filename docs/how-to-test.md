# How To Test

## Pre-requisites

You will need a working OCM Agent image that the `ocm-agent-operator` will deploy.

Follow the build instructions of [OCM Agent](https://github.com/openshift/ocm-agent) to build your own OCM Agent image.

## Running directly on-cluster using image URL

- Apply the CRD from the `deploy/crds` directory.

```
oc create -f deploy/crds/ocmagent.managed.openshift.io_ocmagents.yaml
```

- Edit the `test/deploy/50_ocm-agent-operator.Deployment.yaml` file to set the `ocm-agent-operator` image to the image URL you wish to test with.

- Apply all of the resources within the `test/deploy` directory.

```
find test/deploy -type f -name '*.yaml' -exec oc create -f {} \;
```

## Running directly on-cluster using make

- Login to the cluster as cluster admin using `oc login` command. 
- Run `make run` to deploy local changes
 
