# Developing the plugin
 
## Getting started

* Install go: `brew install go`

## Developing

* Build current plugin `make all`
* Write tests and production code!
* Run tests: `go test`

## Updating Version 

* update in out_newrelic.go
* update in out_newrelic_test.go
* update in the VERSION file

## Pushing changes to the public repo

After updating the New Relic repo with changes, changes will need to be pushed to the public GitHub repo at: https://github.com/newrelic/newrelic-fluent-bit-output
* `git remote add public git@github.com:newrelic/newrelic-fluent-bit-output.git`
* `git push public master:name-of-branch-to-create`
* Create a PR from that branch in https://github.com/newrelic/newrelic-fluent-bit-output
* Get the PR reviewed, merged, and delete the branch!

## Testing it with a local fluent-bit

```fluent-bit -c yourconfig.conf -e path/to/the/newrelic_out.so -i dummy```

## Libray Usage

Every output plugin go through four callbacks associated to different phases:

| Plugin Phase        | Callback                   |
|---------------------|----------------------------|
| Registration        | FLBPluginRegister()        |
| Initialization      | FLBPluginInit()            |
| Runtime Flush       | FLBPluginFlush()           |
| Exit                | FLBPluginExit()            |

### Plugin Registration

When Fluent Bit loads a Golang plugin, it lookup and load the registration callback that aims to populate the internal structure with plugin name and description among others:

```go
//export FLBPluginRegister
func FLBPluginRegister(ctx unsafe.Pointer) int {
	return output.FLBPluginRegister(ctx, "gstdout", "Stdout GO!")
}
```

This function is invoked at start time _before_ any configuration is done inside the engine.

### Plugin Initialization

Before the engine starts, it initialize all plugins that were requested to start. Upon initialization a configuration context already exists, so the plugin can ask for configuration parameters or do any other internal checks. E.g:

```go
//export FLBPluginInit
func FLBPluginInit(ctx unsafe.Pointer) int {
	return output.FLB_OK
}
```

The function must return FLB\_OK when it initialized properly or FLB\_ERROR if something went wrong. If the plugin reports an error, the engine will _not_ load the instance.

### Runtime Flush

Upon flush time, when Fluent Bit want's to flush it buffers, the runtime flush callback will be triggered.

The callback will receive a raw buffer of msgpack data with it proper bytes length and the tag associated.

```go
//export FLBPluginFlush
func FLBPluginFlush(data unsafe.Pointer, length C.int, tag *C.char) int {
    return output.FLB_OK
}
```

> for more details about how to process the incoming msgpack data, refer to the [out_gstdout.go](out_gstdout.go) file.

When done, there are three returning values available:

| Return value  | Description                                    |
|---------------|------------------------------------------------|
| FLB\_OK       | The data have been processed normally.         |
| FLB\_ERROR    | An internal error have ocurred, the plugin will not handle the set of records/data again. |
| FLB\_RETRY    | A recoverable error have ocurred, the engine can try to flush the records/data later.|

### Plugin Exit

When Fluent Bit will stop using the instance of the plugin, it will trigger the exit callback. e.g:

```go
//export FLBPluginExit
func FLBPluginExit() int {
	return output.FLB_OK
}
```
