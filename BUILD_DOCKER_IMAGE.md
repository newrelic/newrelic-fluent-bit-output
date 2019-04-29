So you want to update the Fluent Bit Plugin's Docker image?

First things first, you should have Docker installed, and you should have access to [IBM Cloud](https://cloud.ibm.com/) and [quay.io].

Clone this repo, and `cd` into it.

### If you're testing your Docker image: 

Come up with a name for your latest build. Check [IBM Cloud](https://cloud.ibm.com/kubernetes/registry/main/images) to see what the latest tag is. Increment yours by 1. 

Run `docker build -t BUILD_NAME:YOUR_TAG .` 

Run `docker tag BUILD_NAME:YOUR_TAG us.icr.io/fluent-plugin-test/newrelic-fluent-bit-output:YOUR_TAG`

Run `docker push us.icr.io/fluent-plugin-test/newrelic-fluent-bit-output:YOUR_TAG`. 

To test, update line 32 in `new-relic-fluent-plugin.yml` in the [kubernetes logging repo](https://source.datanerd.us/logging/newrelic-kubernetes-logging), then delete and re-apply the `new-relic-fluent-plugin.yml` file as described [here](https://source.datanerd.us/logging/newrelic-kubernetes-logging/blob/master/DEVELOPER.md#now-youre-ready-for-development).




### If you're releasing a new version: 

Come up with a name for your latest build. Check [quay.io](https://quay.io/repository/newrelic/fluent-bit-output?tab=info) to see what the latest tag is. Increment yours by 1. 

Update version.go file with a version bump
Run `./build.sh` 

If you had tested and changed the image name on line 32 in `new-relic-fluent-plugin.yml` in the [kubernetes logging repo](https://source.datanerd.us/logging/newrelic-kubernetes-logging), change it back to `quay.io/newrelic/fluent-bit-output`.

That's basically it. If you want to play with it in Kubernetes, delete and re-apply the `new-relic-fluent-plugin.yml` file as described [here](https://source.datanerd.us/logging/newrelic-kubernetes-logging/blob/master/DEVELOPER.md#now-youre-ready-for-development).
