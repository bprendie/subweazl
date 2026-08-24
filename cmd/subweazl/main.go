package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/bprendie/subweazl/internal/config"
	"github.com/bprendie/subweazl/internal/remote"
	"github.com/bprendie/subweazl/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "remote" {
		if err := runRemote(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "subweazl remote: %v\n", err)
			os.Exit(1)
		}
		return
	}
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}
	if len(os.Args) > 1 && os.Args[1] == "--configure-llm" {
		if err := configureLLM(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "llm config: %v\n", err)
			os.Exit(1)
		}
		return
	}

	model := tui.New(cfg)
	model.EnableRemote()
	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	server, err := remote.Listen(func(command remote.Command) { p.Send(command) })
	if err != nil {
		fmt.Fprintf(os.Stderr, "subweazl: %v\n", err)
		os.Exit(1)
	}
	defer server.Close()
	defer remote.WriteSnapshot(remote.Snapshot{Running: false, State: "stopped"})
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "subweazl: %v\n", err)
		os.Exit(1)
	}
}

func runRemote(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: subweazl remote <status|toggle|next|previous|stop|mode|quit>")
	}
	if args[0] == "status" {
		snapshot, err := remote.ReadSnapshot()
		if err != nil {
			return err
		}
		output, err := json.MarshalIndent(snapshot, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(output))
		return nil
	}
	commands := map[string]remote.Command{
		"toggle": remote.Toggle, "play-pause": remote.Toggle,
		"next": remote.Next, "previous": remote.Previous, "prev": remote.Previous,
		"stop": remote.Stop,
		"mode": remote.CycleMode,
		"quit": remote.Quit,
	}
	command, ok := commands[args[0]]
	if !ok {
		return fmt.Errorf("unknown command %q", args[0])
	}
	return remote.Send(command)
}

func configureLLM(cfg config.Config) error {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Subweazl optional LLM curator setup")
	fmt.Println("Providers: omarchy, ollama, vllm. Leave blank to disable AI.")
	provider := strings.ToLower(ask(reader, "Provider", cfg.LLM.Provider))
	if provider == "" {
		cfg.LLM = config.LLMConfig{}
		if err := config.Save(cfg); err != nil {
			return err
		}
		fmt.Println("LLM curator disabled.")
		return nil
	}
	if provider == "omarchy" {
		cfg.LLM = config.LLMConfig{Provider: provider}
		if err := config.Save(cfg); err != nil {
			return err
		}
		fmt.Println("DJ-Weazl will use the current Omarchy default agent.")
		return nil
	}
	if provider != "ollama" && provider != "vllm" {
		return fmt.Errorf("provider must be omarchy, ollama, or vllm")
	}
	cfg.LLM = config.LLMConfig{Provider: provider}
	cfg.LLM.BaseURL = askRequired(reader, "Base URL", cfg.LLM.BaseURL)
	cfg.LLM.ChatPath = askRequired(reader, "Chat completion path", cfg.LLM.ChatPath)
	cfg.LLM.ModelsPath = ask(reader, "Model listing path", cfg.LLM.ModelsPath)
	cfg.LLM.Model = askRequired(reader, "Model", cfg.LLM.Model)
	cfg.LLM.ContextWindow = askInt(reader, "Context window", cfg.LLM.ContextWindow)
	cfg.LLM.APIKey = askSecret(reader, "API key env value or literal", cfg.LLM.APIKey)
	if err := config.Save(cfg); err != nil {
		return err
	}
	fmt.Println("LLM curator config saved.")
	return nil
}

func askRequired(reader *bufio.Reader, label, current string) string {
	for {
		value := ask(reader, label, current)
		if value != "" {
			return value
		}
		fmt.Println("Required.")
	}
}

func ask(reader *bufio.Reader, label, current string) string {
	if current != "" {
		fmt.Printf("%s [%s]: ", label, current)
	} else {
		fmt.Printf("%s: ", label)
	}
	text, _ := reader.ReadString('\n')
	text = strings.TrimSpace(text)
	if text == "" {
		return current
	}
	return text
}

func askSecret(reader *bufio.Reader, label, current string) string {
	if current != "" {
		fmt.Printf("%s [configured, blank keeps it]: ", label)
	} else {
		fmt.Printf("%s [blank for none]: ", label)
	}
	text, _ := reader.ReadString('\n')
	text = strings.TrimSpace(text)
	if text == "" {
		return current
	}
	return text
}

func askInt(reader *bufio.Reader, label string, current int) int {
	for {
		currentText := ""
		if current > 0 {
			currentText = strconv.Itoa(current)
		}
		value := ask(reader, label, currentText)
		if value == "" {
			return 0
		}
		n, err := strconv.Atoi(value)
		if err == nil && n > 0 {
			return n
		}
		fmt.Println("Enter a positive number or leave blank.")
	}
}
