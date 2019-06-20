# Developing the plugin

## Getting started

* Install go: `brew install go`

## Developing

* Build current plugin: `make all`
* Write tests and production code!
* Run tests: `go test`

## Updating Version

* Update the version in version.go

## Pushing changes to the public repo

After updating the New Relic repo with changes, changes will need to be pushed to the public GitHub repo at: https://github.com/newrelic/newrelic-fluent-bit-output
* `git remote add public git@github.com:newrelic/newrelic-fluent-bit-output.git`
* `git push public master:name-of-branch-to-create`
* Create a PR from that branch in https://github.com/newrelic/newrelic-fluent-bit-output
* Get the PR reviewed, merged, and delete the branch!
]

## Creating a new Docker Image
* see [BUILD_DOCKER_IMAGE.md](BUILD_DOCKER_IMAGE.md)


## Testing it with a local fluent-bit

```fluent-bit -c yourconfig.conf -e path/to/the/newrelic_out.so -i dummy```

## Library Usage

* [Link for official docs](https://github.com/fluent/fluent-bit-go/blob/c4a158a6e3a793166c6ecfa2d5c80d71eada8959/examples/out_gstdout/README.md)
```
