package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/tingly-dev/tingly-agentscope/pkg/agent"
	"github.com/tingly-dev/tingly-agentscope/pkg/memory"
	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	anthropic "github.com/tingly-dev/tingly-agentscope/pkg/model/anthropic"
	"github.com/tingly-dev/tingly-agentscope/pkg/tool"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"

	"github.com/FFengIll/tingly-pm/board"
	"github.com/FFengIll/tingly-pm/prompt"
	pmtools "github.com/FFengIll/tingly-pm/tools"
)

func main() {
	mode := flag.String("mode", "chat", "run mode: 'chat' (interactive), 'run' (stdio json), 'serve' (http)")
	addr := flag.String("addr", ":8080", "HTTP listen address (serve mode)")
	dir := flag.String("dir", ".", "project directory")
	configDir := flag.String("config", "", "path to config directory containing config.json")
	flag.Parse()

	projectDir, _ := filepath.Abs(*dir)
	pmDir := filepath.Join(projectDir, ".pm")

	if err := board.EnsureInit(pmDir); err != nil {
		log.Fatalf("failed to init .pm: %v", err)
	}

	cfg := loadConfig(*configDir)

	ag, err := createAgent(pmDir, cfg)
	if err != nil {
		log.Fatalf("failed to create agent: %v", err)
	}

	switch *mode {
	case "serve":
		runHTTP(ag, *addr)
	case "run":
		runStdio(ag)
	case "chat":
		runChat(ag)
	default:
		log.Fatalf("unknown mode: %s (use 'chat', 'run', or 'serve')", *mode)
	}
}

// Config holds model configuration
type Config struct {
	BaseURL   string `json:"base_url"`
	APIKey    string `json:"api_key"`
	Model     string `json:"model"`
	MaxTokens int    `json:"max_tokens"`
}

func loadConfig(dir string) *Config {
	cfg := &Config{
		Model:     "claude-sonnet-4-20250514",
		MaxTokens: 8192,
	}

	if dir == "" {
		cfg.APIKey = os.Getenv("ANTHROPIC_API_KEY")
		return cfg
	}

	path := filepath.Join(dir, "config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("failed to read %s: %v", path, err)
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		log.Fatalf("failed to parse %s: %v", path, err)
	}

	if cfg.APIKey == "" {
		cfg.APIKey = os.Getenv("ANTHROPIC_API_KEY")
	}

	return cfg
}

func createAgent(pmDir string, cfg *Config) (*agent.ReActAgent, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("api_key required (set in config.json or ANTHROPIC_API_KEY env)")
	}

	llm, err := anthropic.NewClient(&anthropic.Config{
		APIKey:    cfg.APIKey,
		BaseURL:   cfg.BaseURL,
		Model:     cfg.Model,
		MaxTokens: cfg.MaxTokens,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create model: %w", err)
	}

	// Tools
	toolkit := tool.NewToolkit()
	pt := pmtools.NewPMTools(pmDir)
	if err := toolkit.RegisterAll(pt); err != nil {
		return nil, fmt.Errorf("failed to register tools: %w", err)
	}

	// Memory
	mem := memory.NewHistory(500)

	// Agent
	ag := agent.NewReActAgent(&agent.ReActAgentConfig{
		Name:          "tingly-pm",
		SystemPrompt:  prompt.SystemPrompt,
		Model:         llm,
		Toolkit:       toolkit,
		Memory:        mem,
		MaxIterations: 10,
	})

	return ag, nil
}

func runChat(ag *agent.ReActAgent) {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	fmt.Println("tingly-pm — AI Project Manager")
	fmt.Println("Type your message and press Enter. /quit to exit.")
	fmt.Println()

	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}
		if input == "/quit" || input == "/exit" {
			fmt.Println("Bye.")
			break
		}

		msg := message.NewMsg("user", input, types.RoleUser)
		resp, err := ag.Reply(context.Background(), msg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			continue
		}

		fmt.Println()
		fmt.Println(resp.GetTextContent())
		fmt.Println()
	}
}

func runStdio(ag *agent.ReActAgent) {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer
	encoder := json.NewEncoder(os.Stdout)

	for scanner.Scan() {
		var req struct {
			Content string `json:"content"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			encoder.Encode(map[string]string{"error": err.Error()})
			continue
		}

		msg := message.NewMsg("user", req.Content, types.RoleUser)
		resp, err := ag.Reply(context.Background(), msg)
		if err != nil {
			encoder.Encode(map[string]string{"error": err.Error()})
			continue
		}

		encoder.Encode(map[string]string{
			"role":    "assistant",
			"content": resp.GetTextContent(),
		})
	}
}

func runHTTP(ag *agent.ReActAgent, addr string) {
	http.HandleFunc("/message", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Content string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		msg := message.NewMsg("user", req.Content, types.RoleUser)
		resp, err := ag.Reply(r.Context(), msg)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"content": resp.GetTextContent(),
		})
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	log.Printf("tingly-pm serving on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
