#!/bin/bash


click_custom_config="/tmp/config.xml"
click_config="/etc/clickhouse-server/config.d/config.xml"

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

cat <<EOF > $click_custom_config
<clickhouse>
    <macros>
        <shard>${SHARD}</shard>
        <replica>${REPLICA}</replica>
        <cluster>${CLUSTER}</cluster>
        <installation>${INSTALLATION}</installation>
    </macros>
</clickhouse>
EOF
