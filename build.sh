VERSION=`cat VERSION`
LOCAL_IMAGE=fluent-plugin
REMOTE_IMAGE=quay.io/newrelic/fluent-bit-output
docker build -t ${LOCAL_IMAGE}:${VERSION} .
docker tag ${LOCAL_IMAGE}:${VERSION} ${REMOTE_IMAGE}:${VERSION}
docker push ${REMOTE_IMAGE}:${VERSION}
