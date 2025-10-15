package kairyu

import (
	"fmt"
	"log"
	"plugin"
)

type Plugin interface {
	Name() string
	Init(cfg map[string]any)
	Type() PluginType
	Execute(ctx *Context)
}

type PluginType int

const (
	PluginTypeRequest  = iota // JWT, rate limit, logging
	PluginTypeResponse        // Transform, mask, log
)

type BasePlugin struct {
	pluginType PluginType
}

func (bp *BasePlugin) SetType(t PluginType) { bp.pluginType = t }
func (bp *BasePlugin) Type() PluginType     { return bp.pluginType }

type CorePlugin interface {
	Name() string
	Init(cfg map[string]interface{}) error // инициализация, чтение конфигурации
	Start() error                          // запуск/подключение к шлюзу
	Stop() error                           // остановка/отключение при reload/shutdown
}

var coreRegistry = make(map[string]func() CorePlugin)

func RegisterCorePlugin(name string, factory func() CorePlugin) {
	coreRegistry[name] = factory
}

func createCorePlugin(name string) CorePlugin {
	if f, ok := coreRegistry[name]; ok {
		return f()
	}
	return nil
}

func initCorePlugin(cfg CorePluginConfig) (CorePlugin, error) {
	p := createCorePlugin(cfg.Name)
	if p == nil {
		return nil, fmt.Errorf("core plugin %s not found", cfg.Name)
	}

	if err := p.Init(cfg.Config); err != nil {
		return nil, err
	}

	return p, nil
}

func loadPluginFromSO(path string) Plugin {
	p, err := plugin.Open(path)
	if err != nil {
		log.Printf("failed to open plugin %s: %v", path, err)
		return nil
	}

	sym, err := p.Lookup("NewPlugin")
	if err != nil {
		log.Printf("plugin %s does not export NewPlugin: %v", path, err)
		return nil
	}

	factory, ok := sym.(func() Plugin)
	if !ok {
		log.Printf("plugin %s: NewPlugin has wrong signature", path)
		return nil
	}

	pl := factory()
	log.Printf("plugin %s loaded successfully", pl.Name())

	return pl
}
