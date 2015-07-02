package main

import (
	"encoding/json"
	"fmt"

	"github.com/convox/cli/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/cli/stdcli"
)

type Process struct {
	App         string
	Command     string
	CPU         int64
	Id          string
	Memory      int64
	Name        string
	ServiceType string
}

type Processes []Process

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "ps",
		Description: "list an app's processes",
		Usage:       "",
		Action:      cmdPs,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "app",
				Usage: "app name. Inferred from current directory if not specified.",
			},
		},
	})
}

func cmdPs(c *cli.Context) {
	_, app, err := stdcli.DirApp(c, ".")

	if err != nil {
		stdcli.Error(err)
		return
	}

	data, err := ConvoxGet(fmt.Sprintf("/apps/%s/processes", app))

	if err != nil {
		stdcli.Error(err)
		return
	}

	var processes *Processes
	err = json.Unmarshal(data, &processes)

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Printf("%-13s %-15s %-5s %-5s %s\n", "ID", "PROCESS", "CPU", "MEM", "COMMAND")

	for _, ps := range *processes {
		fmt.Printf("%-13s %-15s %-5d %-5d %s\n", ps.Id, ps.Name, ps.CPU, ps.Memory, ps.Command)
	}
}
