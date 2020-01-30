VERSION=`cat version.go | grep VERSION | awk '{gsub(/"/, "", $4); print $4}'`
LOCAL_IMAGE=fluent-plugin
REMOTE_IMAGE=newrelic/newrelic-fluentbit-output
docker build --build-arg reportingSourceType="docker" --build-arg reportingSourceVersion=${VERSION} -t ${LOCAL_IMAGE}:${VERSION} .
docker tag ${LOCAL_IMAGE}:${VERSION} ${REMOTE_IMAGE}:${VERSION}
docker push ${REMOTE_IMAGE}:${VERSION}
