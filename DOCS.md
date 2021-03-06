---
date: 2017-01-29T00:00:00+00:00
title: ecs-deploy
author: dhoeric
tags: [ deploy, ssh ]
repo: moneysmartco/drone-ecs-deploy
image: moneysmartco/drone-ecs-deploy
---

Use the ecs-deploy plugin to deploy new image and env var on ECS service. The below pipeline configuration demonstrates simple usage:

```yaml
pipeline:
  ecs-deploy:
    image: moneysmartco/drone-ecs-deploy:0.0.1
    cluster: ecs-cluster
    service: ecs-service
    aws_region: ap-southeast-1
    deploy_env_path: ./.deploy.env
    image_name: xxx.dkr.ecr.xxx.amazonaws.com/simple_app:${DRONE_COMMIT:0:8}
    custom_resource_limit_enable: true
    cpu_limit: 512
    memory_limit: 512
    secrets:
      - aws_access_key_id
      - aws_secret_access_key
```

# Parameter Reference

cluster
: ECS cluster to be deployed

service
: ECS service to be deployed

aws_region
: AWS region of the ECS service

deploy_env_path
: Path of dotenv file (default: `./.deploy.env`)

image_name
: Docker image to be deployed on ECS service

custom_resource_limit_enable
: Customize CPU and memory limit (default: false)

cpu_limit
: The number of cpu units reserved for the container. (default: 512)

memory_limit
: The hard limit (in MiB) of memory to present to the container. (default: 512)

# Secret Reference

aws_access_key_id
: AWS access to update ECS service

aws_secret_access_key
: AWS access to update ECS service

