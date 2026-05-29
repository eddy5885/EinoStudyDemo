package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

type getTimeInput struct {
	Timezone string `json:"timezone" jsonschema_description:"IANA 时区，如 Asia/Shanghai"`
}

type calcInput struct {
	A  float64 `json:"a" jsonschema_description:"左操作数"`
	B  float64 `json:"b" jsonschema_description:"右操作数"`
	Op string  `json:"op" jsonschema_description:"运算符 + - * /"`
}

type readFileInput struct {
	Filename string `json:"filename" jsonschema_description:"demo 目录下文件名"`
}

func buildTools(workspace string) ([]tool.BaseTool, error) {
	getTime, err := utils.InferTool("get_time", "获取当前时间", getTime)
	if err != nil {
		return nil, err
	}
	calcTool, err := utils.InferTool("calc", "两数四则运算（调用前需用户审批）", calc)
	if err != nil {
		return nil, err
	}
	readDemo, err := utils.InferTool("read_demo_file", "读取 demo 目录文本文件",
		func(ctx context.Context, in readFileInput) (string, error) {
			return readDemoFile(workspace, in.Filename)
		})
	if err != nil {
		return nil, err
	}
	return []tool.BaseTool{getTime, calcTool, readDemo}, nil
}

func getTime(_ context.Context, in getTimeInput) (string, error) {
	loc := time.Local
	if tz := strings.TrimSpace(in.Timezone); tz != "" {
		var err error
		loc, err = time.LoadLocation(tz)
		if err != nil {
			return "", err
		}
	}
	return time.Now().In(loc).Format(time.RFC3339), nil
}

func calc(_ context.Context, in calcInput) (string, error) {
	var out float64
	switch in.Op {
	case "+":
		out = in.A + in.B
	case "-":
		out = in.A - in.B
	case "*":
		out = in.A * in.B
	case "/":
		if in.B == 0 {
			return "", fmt.Errorf("除数不能为 0")
		}
		out = in.A / in.B
	default:
		return "", fmt.Errorf("不支持的运算符 %q", in.Op)
	}
	return fmt.Sprintf("%g", out), nil
}

func readDemoFile(workspace, filename string) (string, error) {
	filename = filepath.Base(strings.TrimSpace(filename))
	absWorkspace, err := filepath.Abs(workspace)
	if err != nil {
		return "", err
	}
	absTarget, err := filepath.Abs(filepath.Join(absWorkspace, filename))
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(absTarget, absWorkspace+string(os.PathSeparator)) && absTarget != absWorkspace {
		return "", fmt.Errorf("禁止访问 demo 外路径")
	}
	data, err := os.ReadFile(absTarget)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func demoWorkspace() string {
	if v := os.Getenv("LESSON07_DEMO_DIR"); v != "" {
		return v
	}
	return "demo"
}
