package llm

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/bprendie/subweazl/internal/config"
)

func ResolveConfig(ctx context.Context, cfg config.LLMConfig) (config.LLMConfig, error) {
	cfg.NormalizeForClient()
	if cfg.Provider != "omarchy" {
		return cfg, nil
	}
	agent, err := resolveDefaultAgent(ctx)
	if err != nil {
		return cfg, err
	}
	cfg.Model = agent
	return cfg, nil
}

func resolveDefaultAgent(ctx context.Context) (string, error) {
	output, err := exec.CommandContext(ctx, "omarchy", "default", "agent").Output()
	if err != nil {
		return "", fmt.Errorf("resolve Omarchy default agent: %w", err)
	}
	agent := strings.TrimSpace(string(output))
	if agent == "" {
		return "", errors.New("choose a default agent with: omarchy default agent <name>")
	}
	if _, err := exec.LookPath(agent); err != nil {
		return "", fmt.Errorf("Omarchy default agent %q is not installed", agent)
	}
	return agent, nil
}

func completeWithAgent(ctx context.Context, agent string, messages []Message, maxTokens int) (string, error) {
	if strings.TrimSpace(agent) == "" {
		return "", errors.New("Omarchy agent is not resolved")
	}
	workDir, err := os.MkdirTemp("", "subweazl-agent-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(workDir)
	prompt := agentPrompt(messages, maxTokens)
	name, args, err := agentCommand(agent, prompt)
	if err != nil {
		return "", err
	}
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = workDir
	output, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			detail := strings.TrimSpace(string(exitErr.Stderr))
			if detail == "" {
				detail = exitErr.Error()
			}
			return "", fmt.Errorf("Omarchy agent %s failed: %s", agent, detail)
		}
		return "", fmt.Errorf("run Omarchy agent %s: %w", agent, err)
	}
	result := strings.TrimSpace(string(output))
	if result == "" {
		return "", fmt.Errorf("Omarchy agent %s returned no content", agent)
	}
	return result, nil
}

func agentPrompt(messages []Message, maxTokens int) string {
	var prompt strings.Builder
	prompt.WriteString("Return only the requested structured response. Do not use tools, inspect files, or modify the system.\n")
	if maxTokens > 0 {
		prompt.WriteString("Keep the response within " + strconv.Itoa(maxTokens) + " tokens.\n")
	}
	for _, message := range messages {
		role := strings.ToUpper(strings.TrimSpace(message.Role))
		if role == "" {
			role = "USER"
		}
		prompt.WriteString("\n<" + role + ">\n")
		prompt.WriteString(message.Content)
		prompt.WriteString("\n</" + role + ">\n")
	}
	return prompt.String()
}

func agentCommand(agent, prompt string) (string, []string, error) {
	switch agent {
	case "codex":
		return agent, []string{"exec", "--sandbox", "read-only", "--ephemeral", "--skip-git-repo-check", "--color", "never", prompt}, nil
	default:
		return "", nil, fmt.Errorf("unsupported Omarchy default agent %q; select Codex or configure Ollama/vLLM", agent)
	}
}
