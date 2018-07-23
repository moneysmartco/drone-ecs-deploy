# drone-ecs-deploy

Drone plugin to deploy on ECS by given service & cluster, which updating image / env vars.
For the env vars file, you can use with [moneysmartco/drone-vault-exporter](https://www.github.com/moneysmartco/drone-vault-exporter).
For the usage information and a listing of the available options please take a look at [the docs](DOCS.md).

## Build

Build the binary with the following commands:

```
make install
make build
```

## Docker

Build the docker image with the following commands:

```
make linux_amd64 docker_image docker_deploy tag=X.X.X
```

## Usage

Execute from the working directory:

```sh
AWS_ACCESS_KEY_ID=xxxx \
AWS_SECRET_ACCESS_KEY=xxxx \
PLUGIN_CLUSTER=ecs-cluster \
PLUGIN_SERVICE=ecs-service \
PLUGIN_AWS_REGION=ap-southeast-1 \
PLUGIN_POLLING_CHECK_ENABLE=true \
PLUGIN_CUSTOM_ENVS='{"foo": "BAR"}' \
PLUGIN_IMAGE_NAME=xxxx.dkr.ecr.xxx.amazon.com/simple_app:latest \
  ./drone-ecs-deploy
```
