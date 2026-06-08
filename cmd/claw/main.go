// cmd/claw/main.go
package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/yanking/go-tiny-claw/internal/config"
	"github.com/yanking/go-tiny-claw/internal/engine"
	"github.com/yanking/go-tiny-claw/internal/provider"
	"github.com/yanking/go-tiny-claw/internal/reporter"
	"github.com/yanking/go-tiny-claw/internal/tools"
	"github.com/yanking/go-tiny-claw/pkg/conf"
)

var configFile string

func init() {
	flag.StringVar(&configFile, "conf", "configs/config.yaml", "config path, eg: -conf config.yaml")
}
func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(configFile, &c)

	workDir, _ := os.Getwd()
	workDir += "/workspace"

	var llmProvider provider.LLMProvider
	switch c.LLM.Provider {
	case "anthropic":
		llmProvider = provider.NewClaudeProvider(c.LLM.APIKey, c.LLM.BaseURL, c.LLM.Model)
	case "openai":
	default:
		llmProvider = provider.NewOpenAIProvider(c.LLM.APIKey, c.LLM.BaseURL, c.LLM.Model)
	}

	registry := tools.NewRegistry()
	registry.Register(tools.NewReadFileTool(workDir))
	registry.Register(tools.NewWriteFileTool(workDir))
	registry.Register(tools.NewEditFileTool(workDir))
	registry.Register(tools.NewBashTool(workDir))

	eng := engine.NewAgentEngine(llmProvider, registry, workDir, true)

	// 构造 reporter
	var r reporter.Reporter
	r = reporter.NewTerminalReporter()

	// 设定测试任务
	prompt := `
我需要在当前目录下新建一个 ping.go，提供一个简单的 http ping 接口。 
写完之后，帮我把代码用 git 提交一下。
`

	err := eng.Run(context.Background(), prompt, r)
	if err != nil {
		log.Fatalf("引擎运行崩溃: %v", err)
	}
}
