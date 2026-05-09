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

create_topic "url-created"
create_topic "url-clicked"

create_topic "url-shortener.urls.created"
create_topic "url-shortener.urls.deleted"

create_topic "url-shortener.redirects.requested"
create_topic "url-shortener.redirects.resolved"
create_topic "url-shortener.redirects.failed"

create_topic "url-shortener.analytics.aggregated"

echo "All topics created!"
