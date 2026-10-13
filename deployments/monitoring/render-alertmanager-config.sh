#!/bin/sh
set -eu

escape_sed() {
  printf '%s' "$1" | sed 's/[\\&#]/\\&/g'
}

sed \
  -e "s#__SMTP_HOST__#$(escape_sed "${ALERT_SMTP_HOST:-mailpit:1025}")#g" \
  -e "s#__SMTP_FROM__#$(escape_sed "${ALERT_SMTP_FROM:-stream-aggregator@localhost}")#g" \
  -e "s#__SMTP_TO__#$(escape_sed "${ALERT_SMTP_TO:-alerts@localhost}")#g" \
  -e "s#__SMTP_USERNAME__#$(escape_sed "${ALERT_SMTP_USERNAME:-}")#g" \
  -e "s#__SMTP_REQUIRE_TLS__#$(escape_sed "${ALERT_SMTP_REQUIRE_TLS:-false}")#g" \
  /etc/alertmanager/alertmanager.yml.tmpl > /tmp/alertmanager.yml

exec /bin/alertmanager \
  --config.file=/tmp/alertmanager.yml \
  --web.external-url=http://localhost:9093
