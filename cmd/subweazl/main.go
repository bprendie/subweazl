package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/bprendie/subweazl/internal/config"
	"github.com/bprendie/subweazl/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
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

	p := tea.NewProgram(tui.New(cfg), tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "subweazl: %v\n", err)
		os.Exit(1)
	}
}

func configureLLM(cfg config.Config) error {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Subweazl optional LLM curator setup")
	fmt.Println("Leave provider label blank to disable AI.")
	provider := ask(reader, "Provider label", cfg.LLM.Provider)
	if provider == "" {
		cfg.LLM = config.LLMConfig{}
		if err := config.Save(cfg); err != nil {
			return err
		}
		fmt.Println("LLM curator disabled.")
		return nil
	}
	cfg.LLM.Provider = provider
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
