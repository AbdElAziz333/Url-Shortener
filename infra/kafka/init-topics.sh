#!/bin/bash
set -e

BOOTSTRAP="kafka:9092"

create_topic() {
  /opt/bitnami/kafka/bin/kafka-topics.sh \
    --bootstrap-server "$BOOTSTRAP" \
    --create --if-not-exists \
    --topic "$1" \
    --partitions 1 \
    --replication-factor 1
  echo "Topic '$1' ready"
}

create_topic "url-created"
create_topic "url-clicked"

echo "All topics created!"