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
	case "flatten-src":
		flattenCmd := flag.NewFlagSet("llm flatten-src", flag.ExitOnError)
		pattern := flattenCmd.String("pattern", "", "ファイルを検索する正規表現パターン")
		extension := flattenCmd.String("extension", "", "ファイル拡張子でフィルタリング（例: *.go）")
		maxTokens := flattenCmd.Int("max-input-tokens", 200000, "最大トークン数（デフォルト: 200000）")
		depthLimit := flattenCmd.Int("depth-limit", 10, "ディレクトリ探索の深さ制限（デフォルト: 10）")
		debug := flattenCmd.Bool("debug", false, "デバッグモードを有効にする")
		flattenCmd.BoolVar(debug, "d", false, "デバッグモードを有効にする (shorthand)")
		path := flattenCmd.String("path", "", "検索を開始するディレクトリパス（デフォルト: カレントディレクトリ）")
		flattenCmd.StringVar(path, "p", "", "検索を開始するディレクトリパス（デフォルト: カレントディレクトリ）")

		if err := flattenCmd.Parse(args[1:]); err != nil {
			fmt.Printf("引数のパースエラー: %v\n", err)
			os.Exit(1)
		}

		if *pattern == "" && *extension == "" {
			fmt.Println("エラー: --pattern または --extension のいずれかは必須です")
			flattenCmd.PrintDefaults()
			os.Exit(1)
		}

		// パスが指定されていない場合はカレントディレクトリを使用
		basePath := *path
		if basePath == "" {
			var err error
			basePath, err = os.Getwd()
			if err != nil {
				fmt.Printf("エラー: カレントディレクトリの取得に失敗しました: %v\n", err)
				os.Exit(1)
			}
		}

		opts := llm.FlattenOptions{
			Pattern:        *pattern,
			Extension:      *extension,
			MaxInputTokens: *maxTokens,
			DepthLimit:     *depthLimit,
			DebugMode:      *debug,
			BasePath:       basePath,
		}

		if err := llm.FlattenSrc(opts); err != nil {
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
	fmt.Println("  list         利用可能なLLMモデルを表示")
	fmt.Println("  ask          LLMに質問する")
	fmt.Println("  flatten-src  ファイルをLLMチャットに適した形式で表示")
	fmt.Println("               [--pattern pattern] [--extension *.ext] [--path|-p dir]")
	fmt.Println("               [--depth-limit n] [--max-input-tokens n] [--debug|-d]")
	fmt.Println("\n詳細なヘルプは各サブコマンドに -h または --help オプションを付けて実行してください")
}

func printGitHelp() {
	fmt.Println("使用方法: hiracli git <subcommand> [options]")
	fmt.Println("\nサブコマンド:")
	fmt.Println("  diff-comment   Git差分からコミットメッセージを生成")
	fmt.Println("\n詳細なヘルプは各サブコマンドに -h または --help オプションを付けて実行してください")
}
