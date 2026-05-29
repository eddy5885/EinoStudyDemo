// Package mem 是业务层的对话存储（非 Eino 框架内置组件）。
package mem

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
)

// SessionMeta 会话列表摘要。
type SessionMeta struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
}

// Session 一次完整对话，消息落盘为 JSONL。
type Session struct {
	ID        string
	CreatedAt time.Time

	filePath string
	mu       sync.Mutex
	messages []*schema.Message
}

// Append 追加消息并写入磁盘。
func (s *Session) Append(msg *schema.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.messages = append(s.messages, msg)

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(s.filePath, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = fmt.Fprintf(f, "%s\n", data)
	return err
}

// Messages 返回历史快照（供 runner.Run 使用）。
func (s *Session) Messages() []*schema.Message {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]*schema.Message, len(s.messages))
	copy(out, s.messages)
	return out
}

// Title 用第一条用户消息作为会话标题。
func (s *Session) Title() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, msg := range s.messages {
		if msg != nil && msg.Role == schema.User && msg.Content != "" {
			title := msg.Content
			if r := []rune(title); len(r) > 40 {
				return string(r[:40]) + "..."
			}
			return title
		}
	}
	return "New Session"
}

// Store 管理多个 Session 的 JSONL 文件。
type Store struct {
	dir   string
	mu    sync.Mutex
	cache map[string]*Session
}

// NewStore 创建存储目录（若不存在）。
func NewStore(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create session dir: %w", err)
	}
	return &Store{
		dir:   dir,
		cache: make(map[string]*Session),
	}, nil
}

// GetOrCreate 按 ID 加载或新建会话；id 为空则生成新 UUID。
func (st *Store) GetOrCreate(id string) (*Session, bool, error) {
	if id == "" {
		id = uuid.NewString()
	}

	st.mu.Lock()
	defer st.mu.Unlock()

	if sess, ok := st.cache[id]; ok {
		return sess, false, nil
	}

	filePath := filepath.Join(st.dir, id+".jsonl")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		sess, err := createSession(id, filePath)
		if err != nil {
			return nil, false, err
		}
		st.cache[id] = sess
		return sess, true, nil
	}

	sess, err := loadSession(filePath)
	if err != nil {
		return nil, false, err
	}
	st.cache[id] = sess
	return sess, false, nil
}

// List 列出所有会话元数据。
func (st *Store) List() ([]SessionMeta, error) {
	entries, err := os.ReadDir(st.dir)
	if err != nil {
		return nil, err
	}

	var metas []SessionMeta
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".jsonl")

		st.mu.Lock()
		sess, cached := st.cache[id]
		st.mu.Unlock()

		if !cached {
			var loadErr error
			sess, loadErr = loadSession(filepath.Join(st.dir, e.Name()))
			if loadErr != nil {
				continue
			}
		}

		metas = append(metas, SessionMeta{
			ID:        id,
			Title:     sess.Title(),
			CreatedAt: sess.CreatedAt,
		})
	}
	return metas, nil
}

type sessionHeader struct {
	Type      string    `json:"type"`
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

func createSession(id, filePath string) (*Session, error) {
	header := sessionHeader{
		Type:      "session",
		ID:        id,
		CreatedAt: time.Now().UTC(),
	}
	data, err := json.Marshal(header)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(filePath, append(data, '\n'), 0o644); err != nil {
		return nil, err
	}
	return &Session{
		ID:        id,
		CreatedAt: header.CreatedAt,
		filePath:  filePath,
		messages:  make([]*schema.Message, 0, 8),
	}, nil
}

func loadSession(filePath string) (*Session, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		return nil, fmt.Errorf("empty session file: %s", filePath)
	}

	var header sessionHeader
	if err := json.Unmarshal(scanner.Bytes(), &header); err != nil {
		return nil, fmt.Errorf("bad session header: %w", err)
	}

	sess := &Session{
		ID:        header.ID,
		CreatedAt: header.CreatedAt,
		filePath:  filePath,
		messages:  make([]*schema.Message, 0, 16),
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var msg schema.Message
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue
		}
		sess.messages = append(sess.messages, &msg)
	}
	return sess, scanner.Err()
}
