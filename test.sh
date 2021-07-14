#!/bin/bash
set -e
clean_up () {
    ARG=$?

    echo "Cleaning up"
    rm -r ./test/testdata || true
    docker-compose -f ./test/docker-compose.yml down

    exit $ARG
}
trap clean_up EXIT

function check_logs {
  echo "Hello!" >> ./test/testdata/fbtest.log
  curl -X PUT -s --fail "http://localhost:1080/mockserver/verify" -d '{
    "httpRequest": {
      "path": "/log/v1"
    }
  }'
  return $?
}

# Create testdata folder and log file
mkdir ./test/testdata || true
touch ./test/testdata/fbtest.log

# Initialize
if [ ${SKIP_BUILD:-no} = "no" ]; then
  echo "Building docker image"
  docker build -f ${DOCKERFILE:-Dockerfile} -t fb-output-plugin .
fi

echo "Starting docker compose"
docker-compose -f ./test/docker-compose.yml up -d
sleep 5

# Wait logs to arrive
echo "Sending logs an waiting for arrive"
echo "Hello!" >> ./test/testdata/fbtest.log
sleep 1

max_retry=10
counter=0
while ! check_logs
do
  echo "Logs not found trying again in 5s. Try #$counter"
  sleep 5
  [[ counter -eq $max_retry ]] && echo "Failed!" && exit 1
  ((counter++))
done
echo "Success!"
