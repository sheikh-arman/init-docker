#!/bin/bash


click_custom_config="/tmp/config.yaml"

export CLUSTER="click_arman"
export INSTALLATION="clickhouse"


if [ "$POD_NAME" == "click-stand-0" ]; then
  SHARD="1"
  REPLICA="1"
  echo "Pod1->>>>"
elif [ "$POD_NAME" == "click-stand-1" ]; then
  SHARD="1"
  REPLICA="2"
  echo "Pod2->>>>"
fi


chop="/tmp/chop-generated-hostname-ports.yaml"
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

server="/tmp/remote-servers.yaml"
cat <<EOF > $server
      remote_servers:
        "@replace": replace
        click_arman:
          secret: mysecretphrase
          shard:
            internal_replication: true
            replica:
              host: click-stand-0.click-pods.click.svc.cluster.local
              port: 9000
            replica:
              host: click-stand-1.click-pods.click.svc.cluster.local
              port: 9000
EOF

keeper="/tmp/use-keeper.yaml"
cat <<EOF > $keeper
      zookeeper:
        "@replace": replace
        node:
          host: clickhouse-keeper.click-keeper
          port: 2181
EOF

cat <<EOF > $click_custom_config
      macros:
        shard: "${SHARD}"
        replica: "${REPLICA}"
        cluster: "${CLUSTER}"
        installation: "${INSTALLATION}"
EOF