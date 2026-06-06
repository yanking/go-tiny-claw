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
	registry.Register(tools.NewEditFileTool(workDir))
	registry.Register(tools.NewBashTool(workDir))

	eng := engine.NewAgentEngine(llmProvider, registry, workDir, true)

	// 设定测试任务
	prompt := `
我当前目录下有 a.txt, b.txt, c.txt 三个文件。 
为了节省时间，请你同时一次性读取这三个文件，并将它们的内容综合起来，告诉我它们分别记录了什么领域的信息。
`

	err := eng.Run(context.Background(), prompt)
	if err != nil {
		log.Fatalf("引擎运行崩溃: %v", err)
	}
}
