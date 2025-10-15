package kairyu

import (
	"encoding/json"
	"log"
	"os"
	"sync"
	"sync/atomic"
)

var (
	currentConfig atomic.Value // Stores *GatewayConfig
	currentRouter atomic.Value // Stores *Router
	pluginsMu     sync.Mutex
	activePlugins []CorePlugin
)

type GatewayConfig struct {
	Schema     string             `json:"schema"`
	Name       string             `json:"name"`
	Version    string             `json:"version"`
	Server     ServerConfig       `json:"server"`
	AdminPanel AdminPanelConfig   `json:"admin_panel"`
	Plugins    []CorePluginConfig `json:"plugins"`
	Routes     []RouteConfig      `json:"routes"`
}

type ServerConfig struct {
	Port    int `json:"port"`
	Timeout int `json:"timeout"`
}
type AdminPanelConfig struct {
	Enable bool `json:"enable"`
	Port   int  `json:"port"`
}

type RouteConfig struct {
	Path      string          `json:"path"`
	Method    string          `json:"method"`
	Plugins   []PluginConfig  `json:"plugins"`
	Backends  []BackendConfig `json:"backends"`
	Aggregate string          `json:"aggregate"`
	Transform string          `json:"transform"`
}

type BackendConfig struct {
	URL     string `json:"url"`
	Method  string `json:"method"`
	Timeout int    `json:"timeout"`
}

type PluginConfig struct {
	Name   string                 `json:"name"`
	Path   string                 `json:"path,omitempty"`
	Config map[string]interface{} `json:"config"`
}

type CorePluginConfig struct {
	Name   string                 `json:"name"`
	Config map[string]interface{} `json:"config"`
}

func LoadConfig(path string) GatewayConfig {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatal("failed to read config file:", err)
	}

	var cfg GatewayConfig
	if err = json.Unmarshal(data, &cfg); err != nil {
		log.Fatal("failed to parse json:", err)
	}

	for _, pcfg := range cfg.Plugins {
		p := createCorePlugin(pcfg.Name)
		if p == nil {
			log.Println("failed to create core plugin:", pcfg.Name)
			continue
		}

		if err = p.Init(pcfg.Config); err != nil {
			log.Fatal("failed to init core plugin:", err)
		}

		if err = p.Start(); err != nil {
			log.Fatal("failed to start core plugin:", err)
		}

		activePlugins = append(activePlugins, p)
	}

	return cfg
}

func ReloadConfig(path string) error {
	cfg := LoadConfig(path)

	newPlugins := make([]CorePlugin, 0, len(cfg.Plugins))
	for _, pcfg := range cfg.Plugins {
		p, err := initCorePlugin(pcfg)
		if err != nil {
			log.Println("failed to init core plugin:", err)
			return err
		}

		if err = p.Start(); err != nil {
			log.Println("failed to start core plugin:", err)
			return err
		}

		newPlugins = append(newPlugins, p)
	}

	router := NewRouter(cfg.Routes)

	pluginsMu.Lock()
	oldPlugins := activePlugins
	pluginsMu.Unlock()

	for _, op := range oldPlugins {
		if err := op.Stop(); err != nil {
			log.Println("failed to stop old core plugin:", err)
			return err
		}
	}

	pluginsMu.Lock()
	activePlugins = newPlugins
	pluginsMu.Unlock()

	currentConfig.Store(&cfg)
	currentRouter.Store(&router)

	log.Printf("configuration reloaded: %d routes, %d core plugins",
		len(cfg.Routes), len(newPlugins))

	return nil
}
