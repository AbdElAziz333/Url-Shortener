#!/bin/bash
set -e

BOOTSTRAP="kafka:9092"

create_topic() {
  /opt/bitnami/kafka/bin/kafka-topics.sh \
    --bootstrap-server "$BOOTSTRAP" \
    --create --if-not-exists \
    --topic "$1" \
    --partitions 1 \
    --replication-factor 1 # 1 for local dev, 3 for prod
  echo "Topic '$1' ready"
}

create_topic "urls.created"
create_topic "urls.deleted"

create_topic "redirects.requested"
create_topic "redirects.resolved"
create_topic "redirects.failed"

create_topic "analytics.aggregated"

echo "All topics created!"
