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
	Timezone string `json:"timezone" jsonschema_description:"IANA 时区，如 Asia/Shanghai；留空则用本地时区"`
}

type calcInput struct {
	A  float64 `json:"a" jsonschema_description:"左操作数"`
	B  float64 `json:"b" jsonschema_description:"右操作数"`
	Op string  `json:"op" jsonschema_description:"运算符：+ - * / 之一"`
}

type readFileInput struct {
	Filename string `json:"filename" jsonschema_description:"demo 目录下的文件名，如 notes.txt"`
}

func buildTools(workspace string) ([]tool.BaseTool, error) {
	getTime, err := utils.InferTool("get_time", "获取指定时区的当前日期时间", getTime)
	if err != nil {
		return nil, err
	}

	calcTool, err := utils.InferTool("calc", "对两个数字做四则运算", calc)
	if err != nil {
		return nil, err
	}

	readDemo, err := utils.InferTool("read_demo_file",
		"读取 demo 目录下的文本文件（仅允许 demo 内文件）",
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
			return "", fmt.Errorf("无效时区 %q: %w", tz, err)
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
		return "", fmt.Errorf("不支持的运算符 %q，请用 + - * /", in.Op)
	}
	return fmt.Sprintf("%g", out), nil
}

func readDemoFile(workspace, filename string) (string, error) {
	filename = filepath.Base(strings.TrimSpace(filename))
	if filename == "" || filename == "." {
		return "", fmt.Errorf("filename 不能为空")
	}

	absWorkspace, err := filepath.Abs(workspace)
	if err != nil {
		return "", err
	}
	absTarget, err := filepath.Abs(filepath.Join(absWorkspace, filename))
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(absTarget, absWorkspace+string(os.PathSeparator)) && absTarget != absWorkspace {
		return "", fmt.Errorf("禁止访问 demo 目录外的路径")
	}

	data, err := os.ReadFile(absTarget)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func demoWorkspace() string {
	if v := os.Getenv("LESSON05_DEMO_DIR"); v != "" {
		return v
	}
	return "demo"
}
