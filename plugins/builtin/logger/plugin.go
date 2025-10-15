package main

import (
	"fmt"

	"github.com/starwalkn/kairyu"
)

type Plugin struct {
	kairyu.BasePlugin
}

func NewPlugin() kairyu.Plugin {
	return &Plugin{}
}

func (p *Plugin) Name() string {
	return "logger"
}

func (p *Plugin) Type() kairyu.PluginType {
	return kairyu.PluginTypeRequest
}

func (p *Plugin) Init(cfg map[string]any) {}

func (p *Plugin) Execute(ctx *kairyu.Context) {
	fmt.Printf("[logger] %s %s\n", ctx.Request.Method, ctx.Request.URL.Path)
}
