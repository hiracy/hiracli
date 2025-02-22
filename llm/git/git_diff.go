package git

import (
	"bytes"
	"fmt"
	"os/exec"

	"hiracli/llm"
)

type GitDiffOptions struct {
	LLMModel string
}

func GetGitDiff() (string, error) {
	cmd := exec.Command("git", "diff")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("git diffの実行に失敗しました: %v", err)
	}
	return out.String(), nil
}

func GitDiffComment(opts GitDiffOptions) error {
	diff, err := GetGitDiff()
	if err != nil {
		return err
	}

	if diff == "" {
		return fmt.Errorf("git diffの結果が空です。変更がありません")
	}

	prompt := fmt.Sprintf(`# git diff
%s
日本語のコミットメッセージを作って`, diff)

	askOpts := llm.AskOptions{
		LLMModel:  opts.LLMModel,
		DebugMode: false,
		Prompt:    prompt,
	}

	return llm.Ask(askOpts)
}
