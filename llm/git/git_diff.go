package git

import (
	"bytes"
	"fmt"
	"os/exec"

	"hiracli/llm"
)

type GitDiffOptions struct {
	LLMModel string
	Cached   bool
}

func GetGitDiff(cached bool) (string, error) {
	args := []string{"diff"}
	if cached {
		args = append(args, "--cached")
	}
	cmd := exec.Command("git", args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("git diffの実行に失敗しました: %v", err)
	}
	return out.String(), nil
}

func GitDiffComment(opts GitDiffOptions) error {
	diff, err := GetGitDiff(opts.Cached)
	if err != nil {
		return err
	}

	if diff == "" {
		if opts.Cached {
			return fmt.Errorf("git diff --cachedの結果が空です。ステージングされた変更がありません")
		}
		return fmt.Errorf("git diffの結果が空です。変更がありません")
	}

	var diffType string
	if opts.Cached {
		diffType = " --cached"
	}
	prompt := fmt.Sprintf(`# git diff%s
%s
日本語のコミットメッセージを作って`, diffType, diff)

	askOpts := llm.AskOptions{
		LLMModel:  opts.LLMModel,
		DebugMode: false,
		Prompt:    prompt,
	}

	return llm.Ask(askOpts)
}
