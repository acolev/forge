package main

import (
	"fmt"
)

type ExamplePlugin struct {
	args []string
}

func (p *ExamplePlugin) Name() string {
	return "Example Plugin"
}

func (p *ExamplePlugin) Execute() error {
	fmt.Println("Executing Example Plugin with args:", p.args)
	return nil
}

func (p *ExamplePlugin) SetArgs(args []string) {
	p.args = args
}

// Exported variable Plugin must be of type ExamplePlugin
var Plugin ExamplePlugin
