#!/bin/bash
set -e
IMAGE=${1:-"fb-output-plugin"}

clean_up () {
    ARG=$?

    if [[ $ARG -ne 0 ]]; then
      echo "Test failed, showing docker logs"
      echo "- Mockserver"
      docker-compose -f ./test/docker-compose.yml logs mockserver
      echo "- Fluent Bit"
      docker-compose -f ./test/docker-compose.yml logs newrelic-fluent-bit-output
    fi

    echo "Cleaning up"
    rm -r ./test/testdata || true
    docker-compose -f ./test/docker-compose.yml down

    exit $ARG
}
trap clean_up EXIT

function check_logs {
  curl -X PUT -s --fail "http://localhost:1080/mockserver/verify" -d @test/verification.json >> /dev/null
  RESULT=$?
  return $RESULT
}

function check_mockserver {
  curl -X PUT -s --fail "http://localhost:1080/mockserver/status" >> /dev/null
  RESULT=$?
  return $RESULT
}

function run_test {
  echo "Starting test for image ${NR_FB_IMAGE}"

  echo "Creating testdata folder and log file"
  mkdir ./test/testdata || true
  touch ./test/testdata/fbtest.log

  echo "Starting docker compose"
  docker-compose -f ./test/docker-compose.yml up -d

  # Waiting mockserver to be ready
  max_retry=10
  counter=0
  until check_mockserver
  do
    echo "Waiting mockserver to be ready. Trying again in 2s. Try #$counter"
    sleep 2
    [[ $counter -eq $max_retry ]] && echo "Mockserver failed to start!" && exit 1
    counter+=1
  done

  # Sending some logs
  echo "Sending logs an waiting for arrive"
  for i in {1..5}; do
    echo "Hello!" >> ./test/testdata/fbtest.log
  done

  # This updates the modified date of the log file, it should
  # be updated with the echo but looks like it doesn't. A reason
  # could be that we're putting this file as a volume and writing
  # small changes so fast, if we add more echoes it works as well.
  touch ./test/testdata/fbtest.log

  max_retry=10
  counter=0
  until check_logs
  do
    echo "Logs not found trying again in 2s. Try #$counter"
    sleep 2
    [[ $counter -eq $max_retry ]] && echo "Logs do not reach the server!" && exit 1
    counter+=1
  done
  echo "Success!"

  echo "Tearing down test for image ${NR_FB_IMAGE}"
  rm -r ./test/testdata || true
  docker-compose -f ./test/docker-compose.yml down
}


# We use the CI env var that GH set to true for every job in the pipeline.
# It will be false when executing this script locally.
if [ ${CI:-"no"} = "no" ]; then
  echo "Building docker image"
  # To avoid requiring QEMU and creating a buildx builder, we simplify the testing
  # to just use the amd64 architecture
  docker build -f ${DOCKERFILE:-Dockerfile} -t $IMAGE .

  NR_FB_IMAGES=($IMAGE)
else
  # We skip re-building the docker image in GH, since we already build it on previous step
  # and make it available on a local registry
  echo "Inspecting Fluent Bit + New Relic image"
  docker buildx imagetools inspect $IMAGE --raw

  NR_FB_IMAGES=( $(docker buildx imagetools inspect $IMAGE --raw | jq -r "if (.mediaType | contains(\"list\")) then \"$IMAGE@\" + .manifests[].digest else \"$IMAGE\" end") )
fi

for imageName in "${NR_FB_IMAGES[@]}"
do
   echo -e "\nTesting image ${imageName}"
   export NR_FB_IMAGE=$imageName
   run_test
done

exit 0


