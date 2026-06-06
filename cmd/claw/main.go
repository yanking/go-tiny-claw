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
	readFileTool := tools.NewReadFileTool(workDir)
	registry.Register(readFileTool)

	eng := engine.NewAgentEngine(llmProvider, registry, workDir, false)

	// 设定测试任务
	prompt := "当前工作区目录下 hello.txt 文件的内容是什么"

	err := eng.Run(context.Background(), prompt)
	if err != nil {
		log.Fatalf("引擎运行崩溃: %v", err)
	}
}
