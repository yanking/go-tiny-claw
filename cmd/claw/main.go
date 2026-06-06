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
	registry.Register(tools.NewBashTool(workDir))

	eng := engine.NewAgentEngine(llmProvider, registry, workDir, false)

	// 设定测试任务
	prompt := `
请帮我执行以下操作： 
1. 用 bash 查看一下我当前电脑的 Go 版本。 
2. 帮我写一个简单的 helloworld.go 文件，输出 "Hello, go-tiny-claw!"。 
3. 用 bash 编译并运行这个 go 文件，确认它能正常工作。
`

	err := eng.Run(context.Background(), prompt)
	if err != nil {
		log.Fatalf("引擎运行崩溃: %v", err)
	}
}
