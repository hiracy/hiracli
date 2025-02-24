package main

import (
	"flag"
	"fmt"
	"os"

	"hiracli/llm"
	gitllm "hiracli/llm/git"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
	}

	if len(os.Args) < 2 {
		printHelp()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "llm":
		handleLLMCommand(os.Args[2:])
	case "git":
		handleGitCommand(os.Args[2:])
	default:
		printHelp()
		os.Exit(1)
	}
}

func handleGitCommand(args []string) {
	if len(args) < 1 {
		printGitHelp()
		os.Exit(1)
	}

	switch args[0] {
	case "diff-comment":
		gitDiffCmd := flag.NewFlagSet("git diff-comment", flag.ExitOnError)
		llmModel := gitDiffCmd.String("llm", "anthropic.claude-3-5-sonnet-20240620-v1:0", "LLMのモデルを指定")
		cached := gitDiffCmd.Bool("cached", false, "ステージングされた変更の差分を使用")

		if err := gitDiffCmd.Parse(args[1:]); err != nil {
			fmt.Printf("引数のパースエラー: %v\n", err)
			os.Exit(1)
		}

		opts := gitllm.GitDiffOptions{
			LLMModel: *llmModel,
			Cached:   *cached,
		}

		if err := gitllm.GitDiffComment(opts); err != nil {
			fmt.Printf("エラー: %v\n", err)
			os.Exit(1)
		}
	default:
		printGitHelp()
		os.Exit(1)
	}
}

func handleLLMCommand(args []string) {
	if len(args) < 1 {
		printLLMHelp()
		os.Exit(1)
	}

	switch args[0] {
	case "list":
		if err := llm.ListModels(); err != nil {
			fmt.Printf("エラー: %v\n", err)
			os.Exit(1)
		}
	case "ask":
		llmAskCmd := flag.NewFlagSet("llm ask", flag.ExitOnError)
		llmModel := llmAskCmd.String("llm", "anthropic.claude-3-5-sonnet-20240620-v1:0", "LLMのモデルを指定")
		debug := llmAskCmd.Bool("debug", false, "デバッグモードを有効にする")
		llmAskCmd.BoolVar(debug, "d", false, "デバッグモードを有効にする (shorthand)")

		if err := llmAskCmd.Parse(args[1:]); err != nil {
			fmt.Printf("引数のパースエラー: %v\n", err)
			os.Exit(1)
		}

		opts := llm.AskOptions{
			LLMModel:  *llmModel,
			DebugMode: *debug,
		}

		if err := llm.Ask(opts); err != nil {
			fmt.Printf("エラー: %v\n", err)
			os.Exit(1)
		}
	default:
		printLLMHelp()
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println("使用方法: hiracli <command> [options]")
	fmt.Println("\nコマンド:")
	fmt.Println("  llm    LLM関連のコマンド")
	fmt.Println("  git    Git関連のコマンド")
	fmt.Println("\n詳細なヘルプは各コマンドに -h または --help オプションを付けて実行してください")
}

func printLLMHelp() {
	fmt.Println("使用方法: hiracli llm <subcommand> [options]")
	fmt.Println("\nサブコマンド:")
	fmt.Println("  list   利用可能なLLMモデルを表示")
	fmt.Println("  ask    LLMに質問する")
	fmt.Println("\n詳細なヘルプは各サブコマンドに -h または --help オプションを付けて実行してください")
}

func printGitHelp() {
	fmt.Println("使用方法: hiracli git <subcommand> [options]")
	fmt.Println("\nサブコマンド:")
	fmt.Println("  diff-comment   Git差分からコミットメッセージを生成")
	fmt.Println("\n詳細なヘルプは各サブコマンドに -h または --help オプションを付けて実行してください")
}
