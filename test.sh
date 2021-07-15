#!/bin/bash
set -e
clean_up () {
    ARG=$?

    if [[ $ARG -ne 0 ]]; then
      echo "Test failed showing docker logs"
      echo "\n- Mockserver\n"
      docker-compose -f ./test/docker-compose.yml logs mockserver
      echo "\n- Fluent Bit\n"
      docker-compose -f ./test/docker-compose.yml logs newrelic-fluent-bit-output
    fi

    echo "Cleaning up"
    rm -r ./test/testdata || true
    docker-compose -f ./test/docker-compose.yml down

    exit $ARG
}
trap clean_up EXIT

function check_logs {
  curl -X PUT -s --fail "http://localhost:1080/mockserver/verify" -d @test/verification.json
  return $?
}

# Create testdata folder and log file
mkdir ./test/testdata || true
touch ./test/testdata/fbtest.log

# Initialize
if [ ${CI:-no} = "no" ]; then
  echo "Building docker image"
  docker build -f ${DOCKERFILE:-Dockerfile} -t fb-output-plugin .
fi

echo "Starting docker compose"
docker-compose -f ./test/docker-compose.yml up -d

# Sending some logs
echo "Sending logs an waiting for arrive"
for i in {1..5}; do
echo "Hello!\n" >> ./test/testdata/fbtest.log
done

max_retry=3
counter=0
while ! check_logs
do
  echo "Logs not found trying again in 5s. Try #$counter"
  sleep 5
  [[ counter -eq $max_retry ]] && echo "Failed!" && exit 1
  ((counter++))
done
echo "Success!"
