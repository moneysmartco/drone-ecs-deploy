FROM alpine:3.4

RUN apk update && \
  apk add -U --no-cache \
  ca-certificates && \
  rm -rf /var/cache/apk/*

LABEL org.label-schema.version=latest
LABEL org.label-schema.vcs-url="https://github.com/moneysmartco/drone-ecs-deploy.git"
LABEL org.label-schema.name="drone-ecs-deploy"
LABEL org.label-schema.vendor="Eric Ho"
LABEL org.label-schema.schema-version="1.0"

ADD release/linux/amd64/drone-ecs-deploy /bin/
ENTRYPOINT ["/bin/drone-ecs-deploy"]
