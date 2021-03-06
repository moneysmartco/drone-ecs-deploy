package main

import (
	"errors"
	"fmt"
	"strconv"
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
		Cluster                   string
		Service                   string
		AwsRegion                 string
		ImageName                 string
		DeployEnvPath             string
		CustomEnvs                map[string]string
		PollingCheckEnable        bool
		PollingInterval           int
		PollingTimeout            int
		CustomResourceLimitEnable bool
		CPULimit                  int64
		MemoryLimit               int64
	}

	// Plugin structure
	Plugin struct {
		Config Config
	}
)

func (p Plugin) readEnv() (envVarMap []ecs.KeyValuePair, err error) {
	// Read from dotenv file
	envMap, err := godotenv.Read(p.Config.DeployEnvPath)
	if err != nil {
		return
	}

	// Add or Update from CustomEnvs
	for k, v := range p.Config.CustomEnvs {
		envMap[strings.ToUpper(k)] = v
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
	envs, err := p.readEnv()
	if err != nil {
		return
	}

	updatedContainerDef := taskDef.ContainerDefinitions
	updatedContainerDef[0].Image = aws.String(p.Config.ImageName)
	updatedContainerDef[0].Environment = envs

	var taskDefCPULimit *string
	var taskDefMemoryLimit *string
	if !p.Config.CustomResourceLimitEnable {
		// Task Definition
		taskDefCPULimit = taskDef.Cpu
		taskDefMemoryLimit = taskDef.Memory
	} else {
		// Task Definition
		taskDefCPULimit = aws.String(strconv.FormatInt(p.Config.CPULimit, 10))
		taskDefMemoryLimit = aws.String(strconv.FormatInt(p.Config.MemoryLimit, 10))

		// Container Definition
		updatedContainerDef[0].Cpu = &p.Config.CPULimit
		updatedContainerDef[0].Memory = &p.Config.MemoryLimit
	}

	createTaskDefReq := ecsSvc.RegisterTaskDefinitionRequest(&ecs.RegisterTaskDefinitionInput{
		ContainerDefinitions:    updatedContainerDef,
		Cpu:                     taskDefCPULimit,
		ExecutionRoleArn:        taskDef.ExecutionRoleArn,
		Family:                  taskDef.Family,
		Memory:                  taskDefMemoryLimit,
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
		return errors.New("deployment timeout, please check application log")
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
	fmt.Println(fmt.Sprintf("- CPU: %s", *taskDef.Cpu))
	fmt.Println(fmt.Sprintf("- Memory: %s MiB", *taskDef.Memory))

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

	fmt.Println("\n= Updating with new Task Definition...")
	updateSvcOutput, err := updateSvcReq.Send()
	if err != nil {
		return err
	}
	fmt.Println("Deployed version: ", *updateSvcOutput.Service.TaskDefinition)
	fmt.Println(fmt.Sprintf("- CPU: %s", *updatedTaskDef.Cpu))
	fmt.Println(fmt.Sprintf("- Memory: %s MiB", *updatedTaskDef.Memory))

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
