package main

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/joho/godotenv"
)

type (
	// Config for the plugin.
	Config struct {
		Cluster            string
		Service            string
		AwsRegion          string
		ImageName          string
		DeployEnvPath      string
		PollingCheckEnable bool
		PollingInterval    int
		PollingTimeout     int
	}

	// Plugin structure
	Plugin struct {
		Config Config
	}
)

func (p Plugin) readDotEnv() (envVarMap []ecs.KeyValuePair, err error) {
	envMap, err := godotenv.Read(p.Config.DeployEnvPath)
	if err != nil {
		return
	}

	for k, v := range envMap {
		envVarMap = append(envVarMap, ecs.KeyValuePair{
			Name:  aws.String(k),
			Value: aws.String(v),
		})
	}
	return
}

func (p Plugin) getTaskDefinitionDetail(ecsSvc *ecs.ECS, taskDefName *string) (taskDefinition *ecs.TaskDefinition, err error) {
	taskDefReq := ecsSvc.DescribeTaskDefinitionRequest(&ecs.DescribeTaskDefinitionInput{
		TaskDefinition: taskDefName,
	})
	taskDefInfo, err := taskDefReq.Send()
	if err != nil {
		return
	}
	taskDefinition = taskDefInfo.TaskDefinition

	return
}

func (p Plugin) updateTaskDefinition(ecsSvc *ecs.ECS, taskDef *ecs.TaskDefinition) (updatedTaskDef *ecs.TaskDefinition, err error) {
	envs, err := p.readDotEnv()
	if err != nil {
		return
	}

	updatedContainerDef := taskDef.ContainerDefinitions
	updatedContainerDef[0].Image = aws.String(p.Config.ImageName)
	updatedContainerDef[0].Environment = envs

	createTaskDefReq := ecsSvc.RegisterTaskDefinitionRequest(&ecs.RegisterTaskDefinitionInput{
		ContainerDefinitions:    updatedContainerDef,
		Cpu:                     taskDef.Cpu,
		ExecutionRoleArn:        taskDef.ExecutionRoleArn,
		Family:                  taskDef.Family,
		Memory:                  taskDef.Memory,
		NetworkMode:             taskDef.NetworkMode,
		PlacementConstraints:    taskDef.PlacementConstraints,
		RequiresCompatibilities: taskDef.RequiresCompatibilities,
		TaskRoleArn:             taskDef.TaskRoleArn,
		Volumes:                 taskDef.Volumes,
	})
	newTaskDefOutput, err := createTaskDefReq.Send()
	if err != nil {
		return nil, err
	}
	updatedTaskDef = newTaskDefOutput.TaskDefinition

	return
}

func (p Plugin) waitDeploymentUntilFinish(ecsSvc *ecs.ECS, ecs_cluster string, ecs_service string, targetTaskDefinition string, interval int, timeout int) (err error) {
	fmt.Printf("\nWait for deployment...\n")
	deploySuccess := false
	start_ts := time.Now()

	// Looping and check till old TaskDef removed
	for {
		ecsReq := ecsSvc.DescribeServicesRequest(&ecs.DescribeServicesInput{
			Cluster: aws.String(ecs_cluster),
			Services: []string{
				ecs_service,
			},
		})
		ecsInfo, err := ecsReq.Send()
		if err != nil {
			return err
		}

		if len(ecsInfo.Services[0].Deployments) == 1 &&
			strings.Compare(*ecsInfo.Services[0].Deployments[0].TaskDefinition, targetTaskDefinition) == 0 {

			deploySuccess = true
			break
		}

		time.Sleep(time.Duration(interval) * time.Second)
		end_ts := time.Now()
		fmt.Printf("Time elapsed: %v\n", end_ts.Sub(start_ts))

		if end_ts.Sub(start_ts) > time.Duration(timeout)*time.Second {
			fmt.Printf("Timeout and abort.\n")
			break
		}
	}

	if deploySuccess == false {
		return errors.New("Deployment timeout")
	}

	return
}

// Exec executes the plugin.
func (p Plugin) Exec() (err error) {
	fmt.Println("============================")
	fmt.Println("= Here is drone-ecs-deploy =")
	fmt.Println("============================")
	fmt.Println("Deploy target: ")
	fmt.Println("Cluster: ", p.Config.Cluster)
	fmt.Println("Service: ", p.Config.Service)
	fmt.Println()

	// Create aws config
	awsCfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return
	}
	if len(p.Config.AwsRegion) != 0 {
		awsCfg.Region = p.Config.AwsRegion
	}

	// Create ECS service
	ecsSvc := ecs.New(awsCfg)
	ecsReq := ecsSvc.DescribeServicesRequest(&ecs.DescribeServicesInput{
		Cluster: &p.Config.Cluster,
		Services: []string{
			p.Config.Service,
		},
	})
	ecsInfo, err := ecsReq.Send()
	if err != nil {
		return
	}
	currentCount := ecsInfo.Services[0].DesiredCount

	// Get and modify task definition
	taskDef, err := p.getTaskDefinitionDetail(ecsSvc, ecsInfo.Services[0].Deployments[0].TaskDefinition)
	if err != nil {
		return
	}

	fmt.Println("Current Task Definition ARN: ", *taskDef.TaskDefinitionArn)
	updatedTaskDef, err := p.updateTaskDefinition(ecsSvc, taskDef)
	if err != nil {
		return
	}

	// Update existing service
	updateSvcReq := ecsSvc.UpdateServiceRequest(&ecs.UpdateServiceInput{
		Cluster:        &p.Config.Cluster,
		Service:        &p.Config.Service,
		TaskDefinition: updatedTaskDef.TaskDefinitionArn,
		DesiredCount:   currentCount,
	})

	fmt.Println("Updating with new Task Definition...")
	updateSvcOutput, err := updateSvcReq.Send()
	if err != nil {
		return err
	}
	fmt.Println("Deployed version: ", *updateSvcOutput.Service.TaskDefinition)

	// Polling until finish
	if p.Config.PollingCheckEnable {
		err := p.waitDeploymentUntilFinish(
			ecsSvc,
			p.Config.Cluster,
			p.Config.Service,
			*updateSvcOutput.Service.TaskDefinition,
			p.Config.PollingInterval,
			p.Config.PollingTimeout,
		)
		if err != nil {
			return err
		}
	}

	fmt.Println("======================")
	fmt.Println("= Deploy is finished =")
	fmt.Println("======================")

	return
}
