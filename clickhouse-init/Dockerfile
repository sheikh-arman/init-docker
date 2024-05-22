FROM alpine:3.18.5

RUN apk add --no-cache bash
COPY scripts/entrypoint-yaml.sh /usr/local/bin/entrypoint.sh


RUN chmod -c 755 /usr/local/bin/entrypoint.sh

ENTRYPOINT ["/bin/bash", "/usr/local/bin/entrypoint.sh"]