package main

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/joho/godotenv/autoload"
	"github.com/urfave/cli"
)

// build number set at compile-time
var build = "0"

// Version set at compile-time
var Version string

func main() {
	if Version == "" {
		Version = fmt.Sprintf("0.0.2+%s", build)
	}

	app := cli.NewApp()
	app.Name = "Drone ECS deploy"
	app.Usage = "Deploy to ECS by given service & cluster, only update image / env vars"
	app.Copyright = "Copyright (c) 2018 Eric Ho"
	app.Authors = []cli.Author{
		{
			Name:  "Eric Ho",
			Email: "dho.eric@gmail.com",
		},
	}
	app.Action = run
	app.Version = Version
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "cluster",
			Usage:  "ECS cluster",
			EnvVar: "PLUGIN_CLUSTER",
		},
		cli.StringFlag{
			Name:   "service",
			Usage:  "ECS service",
			EnvVar: "PLUGIN_SERVICE",
		},
		cli.StringFlag{
			Name:   "aws_region",
			Usage:  "AWS region of ECS cluster",
			EnvVar: "PLUGIN_AWS_REGION",
		},
		cli.StringFlag{
			Name:   "image_name",
			Usage:  "docker image to be deploy",
			EnvVar: "PLUGIN_IMAGE_NAME",
		},
		cli.StringFlag{
			Name:   "deploy-env-path",
			Usage:  "Path to save the dotenv file",
			EnvVar: "PLUGIN_DEPLOY_ENV_PATH",
			Value:  ".deploy.env",
		},
		cli.BoolFlag{
			Name:   "polling-check-enable",
			Usage:  "Enable checking on removing old task definition (default: false)",
			EnvVar: "PLUGIN_POLLING_CHECK_ENABLE",
		},
		cli.IntFlag{
			Name:   "polling-interval",
			Usage:  "Interval for ensuring old task definition is replaced (default: 10 sec)",
			EnvVar: "PLUGIN_POLLING_INTERVAL",
			Value:  10,
		},
		cli.IntFlag{
			Name:   "polling-timeout",
			Usage:  "Timeout for ensuring old task definition is replaced (default: 10 mins)",
			EnvVar: "PLUGIN_POLLING_TIMEOUT",
			Value:  600,
		},
		cli.StringFlag{
			Name:  "env-file",
			Usage: "source env file",
		},
	}

	// Override a template
	cli.AppHelpTemplate = `
NAME:
   {{.Name}} - {{.Usage}}

USAGE:
   {{.HelpName}} {{if .VisibleFlags}}[global options]{{end}}{{if .Commands}} command [command options]{{end}} {{if .ArgsUsage}}{{.ArgsUsage}}{{else}}[arguments...]{{end}}
   {{if len .Authors}}
AUTHOR:
   {{range .Authors}}{{ . }}{{end}}
   {{end}}{{if .Commands}}
COMMANDS:
{{range .Commands}}{{if not .HideHelp}}   {{join .Names ", "}}{{ "\t"}}{{.Usage}}{{ "\n" }}{{end}}{{end}}{{end}}{{if .VisibleFlags}}
GLOBAL OPTIONS:
   {{range .VisibleFlags}}{{.}}
   {{end}}{{end}}{{if .Copyright }}
COPYRIGHT:
   {{.Copyright}}
   {{end}}{{if .Version}}
VERSION:
   {{.Version}}
   {{end}}
REPOSITORY:
    Github: https://github.com/moneysmartco/drone-ecs-deploy
`

	if err := app.Run(os.Args); err != nil {
		fmt.Println("drone-ecs-deploy Error: ", err)
		os.Exit(1)
	}
}

func run(c *cli.Context) error {
	if c.String("env-file") != "" {
		_ = godotenv.Load(c.String("env-file"))
	}

	plugin := Plugin{
		Config: Config{
			Cluster:            c.String("cluster"),
			Service:            c.String("service"),
			AwsRegion:          c.String("aws_region"),
			ImageName:          c.String("image_name"),
			DeployEnvPath:      c.String("deploy-env-path"),
			PollingCheckEnable: c.Bool("polling-check-enable"),
			PollingInterval:    c.Int("polling-interval"),
			PollingTimeout:     c.Int("polling-timeout"),
		},
	}

	return plugin.Exec()
}
