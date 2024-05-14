#!/bin/bash


click_custom_config="/tmp/macros.yaml"
macros="macros.yaml"

SHARD=$(yq eval ".[${CLICKHOUSE_POD_NAME}].SHARD" $macros)
REPLICA=$(yq eval ".[${CLICKHOUSE_POD_NAME}].REPLICA" $macros)
CLUSTER=$(yq eval ".[${CLICKHOUSE_POD_NAME}].CLUSTER" $macros)
INSTALLATION=$(yq eval ".[${CLICKHOUSE_POD_NAME}].INSTALLATION" $macros)

cat <<EOF > $click_custom_config
      macros:
        shard: "${SHARD}"
        replica: "${REPLICA}"
        cluster: "${CLUSTER}"
        installation: "${INSTALLATION}"
EOF


chop="/tmp/hostname-ports.yaml"
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

keeper="/tmp/use-keeper.yaml"
cat <<EOF > $keeper
      zookeeper:
        "@replace": replace
        node:
          host: clickhouse-keeper.click-keeper
          port: 2181
EOF

cp "/tmp/config/ch-cluster.yaml" "/tmp/ch-cluster.yaml"


