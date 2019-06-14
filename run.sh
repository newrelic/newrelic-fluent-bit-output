#!/bin/bash
if [ "$SYSLOG" = true ];
then
  HOST=${HOST:-0.0.0.0}
  MODE=${MODE:-UDP}
  PORT=${PORT:-5140}
  PARSER=${PARSER:-syslog-rfc3164}
else
  ./fluent-bit/bin/fluent-bit -c /fluent-bit/etc/fluent-bit.conf -e /fluent-bit/bin/out_newrelic.so
fi



