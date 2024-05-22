#!/bin/bash

if [ -z "${CLICKHOUSE_TOPOLOGY}" ]; then
  echo "CLICKHOUSE_TOPOLOGY is not set"
  exit 1
else
  if [ "${CLICKHOUSE_TOPOLOGY}" = "standalone" ]; then
    echo "CLICKHOUSE_TOPOLOGY is set to ${CLICKHOUSE_TOPOLOGY}"
    exit 0
  fi
fi

if [ -z "${CLICKHOUSE_SHARD}" ]; then
  echo "CLICKHOUSE_SHARD is not set"
  exit 1
fi

if [ -z "${CLICKHOUSE_CLUSTER}" ]; then
  echo "CLICKHOUSE_CLUSTER is not set"
  exit 1
fi

if [ -z "${CLICKHOUSE_INSTALLATION}" ]; then
  echo "CLICKHOUSE_INSTALLATION is not set"
  exit 1
fi

if [ -z "${CLICKHOUSE_KEEPER_HOST}" ]; then
  echo "CLICKHOUSE_KEEPER_HOST is not set"
  exit 1
fi

if [ -z "${CLICKHOUSE_KEEPER_PORT}" ]; then
  echo "CLICKHOUSE_KEEPER_PORT is not set"
  exit 1
fi

macros="/ch-tmp/macros.yaml"
chop="/ch-tmp/hostname-ports.yaml"
keeper="/ch-tmp/use-keeper.yaml"

CLICKHOUSE_REPLICA=$(echo "$CLICKHOUSE_POD_NAME" | grep -oE '[0-9]+' | tail -1)
((CLICKHOUSE_REPLICA++))
cat <<EOF > $macros
      macros:
        shard: "${CLICKHOUSE_SHARD}"
        replica: "${CLICKHOUSE_REPLICA}"
        cluster: "${CLICKHOUSE_CLUSTER}"
        installation: "${CLICKHOUSE_INSTALLATION}"
EOF
cat <<EOF > $chop
      listen_host:
        - "::"
        - 0.0.0.0
      listen_try: 1
      logger:
        level: debug
        log: /var/log/clickhouse-server/clickhouse-server.log
        errorlog: /var/log/clickhouse-server/clickhouse-server.err.log
        size: 1000M
        count: 3
      http_port: 8123
      tcp_port: 9000
EOF
cat <<EOF > $keeper
      zookeeper:
        "@replace": replace
        node:
          host: "${CLICKHOUSE_KEEPER_HOST}"
          port: "${CLICKHOUSE_KEEPER_PORT}"
EOF

echo "Config Files Generated Successfully !!!"