# Developer reference document

## Prerequisites
- Fluent Bit
- A Go environment. On Mac,you can install it with `brew install go`
- Cross-compilation requirements (optional, only required when building for non-native platforms):
    - For `linux/arm64` cross-compilation: `aarch64-linux-gnu-gcc` and `aarch64-linux-gnu-g++` compilers.
    - For `linux/arm/v7` cross-compilation: `arm-linux-gnueabihf-gcc` and `arm-linux-gnueabihf-g++` compilers.
    - For `windows/amd64` cross-compilation: `x86_64-w64-mingw32-gcc` and `x86_64-w64-mingw32-g++` compilers.
    - For `windows/386` cross-compilation: `i686-w64-mingw32-gcc` and `i686-w64-mingw32-g++` compilers
    - On an Ubuntu machine, you can install all the above compilers with: `sudo apt install mingw-w64 g++-arm-linux-gnueabihf g++-aarch64-linux-gnu gcc-arm-linux-gnueabihf gcc-aarch64-linux-gnu`

## Building the out_newrelic plugin
1. Clone [https://github.com/newrelic/newrelic-fluent-bit-output](https://github.com/newrelic/newrelic-fluent-bit-output)
2. Go into the cloned directory: `cd newrelic-fluent-bit-output
3. Build plugin: `make {PLATFORM}`. Supported platforms are: `linux/amd64`, `linux/arm64`, `linux/arm/v7`, `windows/amd64`, `windows/386`.

## Developing cycle
1. Write tests and production code
2. Update the version in `version.go`
3. Run tests: `go test ./...`
4. Compile the plugin: `make linux/amd64`. See [this section](#compiling-the-out_newrelic-plugin) for more details
5. Run Fluent Bit with the plugin using the template config file: `FILE_PATH=/usr/local/var/log/test.log API_KEY=(your-api-key) BUFFER_SIZE= MAX_RECORDS= fluent-bit -c ./fluent-bit.conf -e ./out_newrelic.so`
6. Cause a change that you've configured Fluent Bit to pick up: (`echo "FluentBitTest" >> /usr/local/var/log/test.log`)
7. Look in `https://one.newrelic.com/launcher/logger.log-launcher` for your log message ("FluentBitTest")

## Docker image building and execution

### Single architecture (native)
To build and locally test the Docker image with your native architecture, you just need to:
```
docker build -t <YOUR-IMAGE-NAME>:<YOUR-TAG> .
docker run -e "FILE_PATH=/var/log/*" -e "API_KEY=<YOUR-API-KEY>" <YOUR-IMAGE-NAME>:<YOUR-TAG>
```

### Multiarchitecture

To build and locally test the Docker image for multiple architectures, you need to:
1. Create a `buildx` builder: `docker buildx create --name multiarch-builder --driver docker-container --driver-opt network=host --use`
2. Inspect and bootstrap the created builder: `docker buildx inspect --bootstrap --builder multiarch-builder`. Pay attention to the supported architectures of your builder. If you need more of them, you'll probably need to install QEMU. But if you use Docker Desktop, it typically comes with all the architectures you'll need. 
3. Start a local Docker registry, to avoid pushing to Dockerhub (this is required because multiarch images are not included into `docker images`): `docker run -d -p 5000:5000 --restart=always --name registry registry:2`
4. Build the image and push it to the local registry: `docker buildx build --tag localhost:5000/fb-output-plugin:latest --platform linux/amd64,linux/arm64,linux/arm/v7 --file ./Dockerfile --builder multiarch-builder -o type=registry ./`
5. To inspect the generated image, you can use: `docker buildx imagetools inspect localhost:5000/fb-output-plugin --raw | jq`
6. You can obtain the absolute reference to all the available images with the following handy command: `docker buildx imagetools inspect localhost:5000/fb-output-plugin --raw | jq -r 'if (.mediaType | contains("list")) then "localhost:5000/fb-output-plugin@" + .manifests[].digest else "localhost:5000/fb-output-plugin" end'`
7. Then start the desired image as in the single architecture section. Note that for the image name and tag, you'll have to use `localhost:5000/fb-output-plugin@SOME_SHA256`

This plugin comes with a Dockerfile and sample config that will let you get started with the plugin fairly easily.

## Integration Testing using the Docker images

To build and do an integration test using the Docker image just run `bash test.sh`. Bear in mind that docker-compose is required to run the tests.

Under the hood, the above script does these steps:

1. Determine whether it is running on a local machine or in a GitHub Action, based on the `CI` environment variable (`CI=yes/no`).
    - On local machines, it will build the Docker image (default dockerfile is ./Dockerfile but you can set the one you want with DOCKERFILE env) using the local **native architecture**, resulting in a single-arch image. 
    - On a GitHub Action, it will use the Docker image generated in the previous job step. It supports both single and multi-arch images.
2. Create test/testdata folder for store temporal configurations and the log file
3. Run the docker-compose (./test/docker-compose.yml) with the following instances:
   - A mockserver with expectations from ./test/expectations.json
   - The built docker image with fluent bit configuration from ./test/fluent-bit.conf
4. Send some logs
5. Verify that logs are reaching the mockserver
   - Mockserver requests are verified using ./test/verification.json
6. Cleanup

Note that when using multi-architecture images, the script will perform steps 2-6 for each of the image architecture variants.

## Miscellaneous
### Update the kubernetes image version
The Kubernetes plugin is available as a Helm chart [here](https://github.com/newrelic/helm-charts/tree/master/charts/newrelic-logging), as well as classic K8s manifests [here](https://github.com/newrelic/helm-charts/tree/master/charts/newrelic-logging/k8s).
Update the image version number [here](https://github.com/newrelic/helm-charts/blob/master/charts/newrelic-logging/k8s/new-relic-fluent-plugin.yml#L44) and [here](https://github.com/newrelic/helm-charts/blob/master/charts/newrelic-logging/Chart.yaml#L5) to make them use the latest Newrelic Fluent Bit output Docker image.

### Cross compiling in the pipeline
* The pipeline uses a Linux machine to compile the plugins
* Go has a built-in way to do [cross compiling](https://github.com/golang/go/wiki/WindowsCrossCompiling)
* To cross compile our plugin we will need the set the `CGO_ENABLED` variable to `1` as we are building a C-shared library. When we set this, Go will also allow us to provide our own C compiler as `CC` and a C++ compiler as `CXX`. 
* In the pipeline script we use [Mingw-w64](http://mingw-w64.org/doku.php/start) as our compiler because it supports both x86 and x64 Windows architectures.
* Refer to the [Makefile](Makefile) file for the available cross-compiling targets.


