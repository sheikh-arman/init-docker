#!/bin/bash

macros="/ch-tmp/macros.yaml"
chop="/ch-tmp/hostname-ports.yaml"
keeper="/ch-tmp/use-keeper.yaml"


if [ -z "${CLICKHOUSE_TOPOLOGY}" ]; then
  echo "CLICKHOUSE_TOPOLOGY is not set"
else
  if [ "${CLICKHOUSE_TOPOLOGY}" = "standalone" ]; then
    echo "CLICKHOUSE_TOPOLOGY is set to ${CLICKHOUSE_TOPOLOGY}"
    exit 0
  fi
fi

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
          host: clickhouse-keeper.click-keeper
          port: 2181
EOF

echo "Config Files Generated Successfully !!!"