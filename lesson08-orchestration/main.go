//go:debug x509negativeserial=1

// Lesson 08: Chain / Graph / Workflow 编排
//
// 对比 Lesson 07：
//   - 不再只靠 Agent 隐式循环，而是用 compose 显式定义数据流
//   - Chain：线性；Graph：分支；Workflow：声明依赖的多节点流水线
//   - Workflow 可编译为 Runnable，再封装成 Agent 的 Tool（Graph Tool 思想）
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
)

func main() {
	demo := flag.String("demo", "all", "chain | graph | workflow | agent | all")
	flag.Parse()

	ctx := context.Background()

	var err error
	switch *demo {
	case "chain":
		err = runChainDemo(ctx)
	case "graph":
		err = runGraphDemo(ctx)
	case "workflow":
		err = runWorkflowDemo(ctx)
	case "agent":
		err = runAgentDemo(ctx)
	case "all":
		if err = runChainDemo(ctx); err != nil {
			break
		}
		if err = runGraphDemo(ctx); err != nil {
			break
		}
		if err = runWorkflowDemo(ctx); err != nil {
			break
		}
		fmt.Println("\n（all 模式跳过 agent 交互；单独运行: go run . -demo agent）")
	default:
		fmt.Fprintf(os.Stderr, "未知 demo: %s\n", *demo)
		os.Exit(2)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
