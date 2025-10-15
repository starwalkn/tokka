package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/starwalkn/kairyu"
	"github.com/starwalkn/kairyu/admin"

	_ "github.com/starwalkn/kairyu/internal/plugin/ratelimit"
)

func main() {
	cfg := kairyu.LoadConfig(os.Getenv("KAIRYU_CONFIG"))

	kairyu.SetRouter(kairyu.NewRouter(cfg.Routes))

	addr := fmt.Sprintf(":%d", cfg.Server.Port)

	printConfig(cfg)

	go admin.StartServer(&cfg)
	go watchReloadSignal(os.Getenv("KAIRYU_CONFIG"))

	if err := http.ListenAndServe(addr, &kairyu.DynamicRouter{}); err != nil {
		log.Fatal(err)
	}
}

func watchReloadSignal(configPath string) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGHUP)

	for {
		<-sigs
		log.Println("🔄 Reload signal received")
		if err := kairyu.ReloadConfig(configPath); err != nil {
			log.Println("Failed to reload config:", err)
		}
	}
}

func printConfig(cfg kairyu.GatewayConfig) {
	fmt.Println("🚀 Kairyu API Gateway started at\n", fmt.Sprintf(":%d", cfg.Server.Port))
	fmt.Printf("\n🚀 Loaded Kairyu Gateway configuration:\n")
	fmt.Printf("────────────────────────────────────────────\n")
	fmt.Printf("Name:    %s\n", cfg.Name)
	fmt.Printf("Version: %s\n", cfg.Version)
	fmt.Printf("Schema:  %s\n", cfg.Schema)
	fmt.Printf("Server:  port=%d  timeout=%dms\n\n", cfg.Server.Port, cfg.Server.Timeout)

	if len(cfg.Plugins) > 0 {
		fmt.Println("🔌 Global plugins:")
		for _, p := range cfg.Plugins {
			fmt.Printf("   • %s\n", p.Name)
		}
		fmt.Println()
	}

	fmt.Println("📦 Routes:")
	for i, route := range cfg.Routes {
		fmt.Printf(" %d) %s %s\n", i+1, route.Method, route.Path)

		if len(route.Plugins) > 0 {
			fmt.Println("    ├─ Plugins:")
			for _, p := range route.Plugins {
				fmt.Printf("    │   • %s\n", p.Name)
				if len(p.Config) > 0 {
					fmt.Printf("    │     Config:\n")
					for k, v := range p.Config {
						fmt.Printf("    │       - %s: %v\n", k, v)
					}
				}
			}
		}

		if len(route.Backends) > 0 {
			fmt.Println("    ├─ Backends:")
			for _, b := range route.Backends {
				timeout := ""
				if b.Timeout > 0 {
					timeout = fmt.Sprintf(" (timeout=%dms)", b.Timeout)
				}
				fmt.Printf("    │   • %s %s%s\n", b.Method, b.URL, timeout)
			}
		}

		if route.Aggregate != "" || route.Transform != "" {
			fmt.Println("    └─ Options:")
			if route.Aggregate != "" {
				fmt.Printf("        • aggregate: %s\n", route.Aggregate)
			}
			if route.Transform != "" {
				fmt.Printf("        • transform: %s\n", route.Transform)
			}
		}
		fmt.Println()
	}
	fmt.Printf("────────────────────────────────────────────\n\n")
}
