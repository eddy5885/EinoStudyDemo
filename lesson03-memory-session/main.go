//go:debug x509negativeserial=1

// Lesson 03: Memory / Session（JSONL 持久化）
//
// 对比 Lesson 02：
//   - Lesson 02：history 只在内存，进程退出即丢失
//   - Lesson 03：业务层 Store/Session 把每轮 user/assistant 写入 JSONL，可跨进程恢复
//
// 注意：Memory/Session 是业务层设计，不是 Eino 框架内置组件；Runner 仍只接收 []Message。
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"

	"github.com/eddy/526eino/lesson03-memory-session/mem"
)

func main() {
	var (
		sessionID = flag.String("session", "", "恢复已有会话 ID；留空则创建新会话")
		listOnly  = flag.Bool("list", false, "列出已保存的会话")
	)
	flag.Parse()

	sessionDir := envOr("SESSION_DIR", "./data/sessions")
	store, err := mem.NewStore(sessionDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "store: %v\n", err)
		os.Exit(1)
	}

	if *listOnly {
		printSessionList(store)
		return
	}

	sess, created, err := store.GetOrCreate(*sessionID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "session: %v\n", err)
		os.Exit(1)
	}

	if created {
		fmt.Printf("新会话: %s\n", sess.ID)
	} else {
		fmt.Printf("恢复会话: %s（已有 %d 条消息）\n", sess.ID, len(sess.Messages()))
	}
	fmt.Printf("标题: %s\n", sess.Title())
	fmt.Printf("存储: %s\n", filepath.Join(sessionDir, sess.ID+".jsonl"))
	fmt.Println("空行退出。恢复本会话：")
	fmt.Printf("  go run . -session %s\n\n", sess.ID)

	ctx := context.Background()
	runner := newRunner(ctx)

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("you> ")
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			break
		}

		if err := sess.Append(schema.UserMessage(line)); err != nil {
			fmt.Fprintf(os.Stderr, "save user: %v\n", err)
			os.Exit(1)
		}

		events := runner.Run(ctx, sess.Messages())
		reply, err := collectAssistant(events)
		if err != nil {
			fmt.Fprintf(os.Stderr, "run agent: %v\n", err)
			os.Exit(1)
		}

		if err := sess.Append(schema.AssistantMessage(reply, nil)); err != nil {
			fmt.Fprintf(os.Stderr, "save assistant: %v\n", err)
			os.Exit(1)
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "stdin: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n会话已保存: %s\n", sess.ID)
}

func newRunner(ctx context.Context) *adk.Runner {
	cm := newChatModel(ctx)
	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "Lesson03Agent",
		Description: "Console agent with JSONL session memory.",
		Instruction: "你是简洁的中文助教。若用户之前说过名字或偏好，要记住并引用。",
		Model:       cm,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "create agent: %v\n", err)
		os.Exit(1)
	}
	return adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
	})
}

func printSessionList(store *mem.Store) {
	metas, err := store.List()
	if err != nil {
		fmt.Fprintf(os.Stderr, "list: %v\n", err)
		os.Exit(1)
	}
	if len(metas) == 0 {
		fmt.Println("（暂无会话）")
		return
	}
	for _, m := range metas {
		fmt.Printf("%s  %s  %s\n", m.CreatedAt.Format("2006-01-02 15:04"), m.ID, m.Title)
	}
}
