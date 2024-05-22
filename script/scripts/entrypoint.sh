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
macros="/ch-tmp/macros.yaml"
CLICKHOUSE_REPLICA=$(echo "$CLICKHOUSE_POD_NAME" | grep -oE '[0-9]+' | tail -1)
((CLICKHOUSE_REPLICA++))
cat <<EOF > $macros
      macros:
        shard: "${CLICKHOUSE_SHARD}"
        replica: "${CLICKHOUSE_REPLICA}"
        cluster: "${CLICKHOUSE_CLUSTER}"
        installation: "${CLICKHOUSE_INSTALLATION}"
EOF

echo "Config Files Generated Successfully !!!"