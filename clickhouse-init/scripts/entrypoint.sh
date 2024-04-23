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


chop="/tmp/chop-generated-hostname-ports.xml"
cat <<EOF > $chop
<clickhouse>
    <!-- Listen wildcard address to allow accepting connections from other containers and host network. -->
    <listen_host>::</listen_host>
    <listen_try>1</listen_try>
    <logger>
        <level>debug</level>
        <log>/var/log/clickhouse-server/clickhouse-server.log</log>
        <errorlog>/var/log/clickhouse-server/clickhouse-server.err.log</errorlog>
        <size>1000M</size>
        <count>3</count>
    </logger>
    <listen_host>0.0.0.0</listen_host>
    <http_port>8123</http_port>
    <tcp_port>9000</tcp_port>
</clickhouse>
EOF

server="/tmp/remote-servers.xml"
cat <<EOF > $server
<clickhouse>
    <remote_servers replace="replace">
        <click_arman>
            <secret>mysecretphrase</secret>
            <shard>
                <internal_replication>true</internal_replication>
                <replica>
                    <host>click-stand-0.click-pods.click.svc.cluster.local</host>
                    <port>9000</port>
                </replica>
                <replica>
                    <host>click-stand-1.click-pods.click.svc.cluster.local</host>
                    <port>9000</port>
                </replica>
            </shard>
        </click_arman>
    </remote_servers>
</clickhouse>
EOF

keeper="/tmp/use-keeper.xml"
cat <<EOF > $keeper
<clickhouse>
    <zookeeper replace="replace">
        <!-- where are the ZK nodes -->
        <node>
            <host>clickhouse-keeper.click-keeper</host>
            <port>2181</port>
        </node>
    </zookeeper>
</clickhouse>
EOF

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