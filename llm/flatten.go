package llm

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"
)

// FlattenOptions は、flatten-srcコマンドのオプションを定義する構造体です
type FlattenOptions struct {
	Pattern        string // ファイルマッチングの正規表現パターン
	Extension      string // ファイル拡張子でのフィルタリング（*.goなど）
	MaxInputTokens int    // 最大トークン数（デフォルト: 200000）
	DepthLimit     int    // サブディレクトリの探索深さ制限（デフォルト: 10）
	CurrentTokens  int    // 現在のトークン数をトラッキング
	IncludedFiles  int    // 処理したファイル数
	DebugMode      bool   // デバッグモードフラグ
	BasePath       string // ベースディレクトリ
}

// FlattenSrc は、指定したパターンに一致するファイルを見つけ、
// それらのファイルのパスとコンテンツを表示する関数です
func FlattenSrc(opts FlattenOptions) error {
	// デフォルト値の設定
	if opts.MaxInputTokens <= 0 {
		opts.MaxInputTokens = 200000
	}

	if opts.DepthLimit <= 0 {
		opts.DepthLimit = 10
	}

	// ベースディレクトリを設定
	baseDir := opts.BasePath
	if baseDir == "" {
		// BasePath が指定されていない場合はカレントディレクトリを使用
		var err error
		baseDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("現在のディレクトリの取得エラー: %v", err)
		}
	}

	// 正規表現パターンのコンパイル
	pattern, err := regexp.Compile(opts.Pattern)
	if err != nil {
		return fmt.Errorf("正規表現パターンのコンパイルエラー: %v", err)
	}

	// 結果を格納する文字列ビルダー
	var result strings.Builder

	// ファイルを収集
	var files []string
	err = filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 相対パスの取得
		relPath, err := filepath.Rel(baseDir, path)
		if err != nil {
			return err
		}

		// ディレクトリの場合は深さをチェック
		if info.IsDir() {
			// カレントディレクトリの場合はスキップしない
			if path == baseDir {
				return nil
			}

			// ディレクトリの深さを計算
			depth := strings.Count(relPath, string(os.PathSeparator)) + 1
			if depth > opts.DepthLimit {
				fmt.Fprintf(os.Stderr, "深さ制限によりスキップ: %s (深さ: %d)\n", relPath, depth)
				return filepath.SkipDir
			}
			return nil
		}

		// 隠しディレクトリ内のファイルはスキップ
		pathParts := strings.Split(relPath, string(os.PathSeparator))
		for _, part := range pathParts {
			if strings.HasPrefix(part, ".") && part != "." && part != ".." {
				return nil
			}
		}

		// パターンと拡張子のフィルタリング
		// パターンマッチングをチェック
		patternMatched := pattern.MatchString(path)

		// 拡張子マッチングをチェック
		extensionMatched := true
		if opts.Extension != "" {
			// ワイルドカードパターンを正規表現に変換
			extPattern := convertWildcardToRegexp(opts.Extension)
			extRe, err := regexp.Compile(extPattern)
			if err != nil {
				fmt.Fprintf(os.Stderr, "警告: 拡張子パターンが不正です: %v\n", err)
				extensionMatched = false
			} else {
				extensionMatched = extRe.MatchString(filepath.Base(path))
			}
		}

		// 両方の条件を満たす場合にファイルを追加
		if patternMatched && extensionMatched {
			files = append(files, path)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("ファイル検索エラー: %v", err)
	}

	// ファイルが見つからなかった場合のメッセージ
	if len(files) == 0 {
		return fmt.Errorf("指定したパターン '%s' に一致するファイルが見つかりませんでした", opts.Pattern)
	}

	// ファイル内容の処理
	for _, file := range files {
		relPath, err := filepath.Rel(baseDir, file)
		if err != nil {
			continue
		}

		// ファイルの読み込み
		content, err := os.ReadFile(file)
		if err != nil {
			if opts.DebugMode {
				fmt.Fprintf(os.Stderr, "警告: ファイル '%s' の読み込みエラー: %v\n", relPath, err)
			}
			continue
		}

		// UTF-8でない場合はスキップ
		if !utf8.Valid(content) {
			if opts.DebugMode {
				fmt.Fprintf(os.Stderr, "警告: ファイル '%s' はUTF-8でないためスキップします\n", relPath)
			}
			continue
		}

		// ファイルの内容をトークン数に変換（簡易的な推定）
		fileContent := string(content)
		fileTokens := estimateTokens(fileContent)

		// トークン数の制限をチェック
		if opts.CurrentTokens+fileTokens > opts.MaxInputTokens {
			if opts.IncludedFiles > 0 {
				if opts.DebugMode {
					fmt.Fprintf(os.Stderr, "警告: トークン制限（%d）に達したため、一部のファイルは含まれていません\n", opts.MaxInputTokens)
					fmt.Fprintf(os.Stderr, "処理したファイル数: %d\n", opts.IncludedFiles)
				}
				break
			} else {
				if opts.DebugMode {
					fmt.Fprintf(os.Stderr, "警告: 最初のファイル '%s' が大きすぎます（推定 %d トークン）\n", relPath, fileTokens)
				}
				// 最初のファイルが大きすぎる場合でも、一部だけでも含める
				fileContent = truncateContent(fileContent, opts.MaxInputTokens)
				fileTokens = opts.MaxInputTokens
			}
		}

		// ファイルパスとコンテンツをフォーマット
		result.WriteString(fmt.Sprintf("### %s\n```\n%s\n```\n\n", relPath, fileContent))

		// トークン数と処理ファイル数を更新
		opts.CurrentTokens += fileTokens
		opts.IncludedFiles++

		if opts.CurrentTokens >= opts.MaxInputTokens {
			break
		}
	}

	// 結果の表示
	fmt.Println(result.String())

	// 統計情報の表示（デバッグモード時のみ）
	if opts.DebugMode {
		fmt.Fprintf(os.Stderr, "統計情報:\n")
		fmt.Fprintf(os.Stderr, "- 処理したファイル数: %d\n", opts.IncludedFiles)
		fmt.Fprintf(os.Stderr, "- 使用トークン数（推定）: %d / %d\n", opts.CurrentTokens, opts.MaxInputTokens)
		fmt.Fprintf(os.Stderr, "- 探索深さ制限: %d\n", opts.DepthLimit)
		fmt.Fprintf(os.Stderr, "- 検索ディレクトリ: %s\n", opts.BasePath)
		if opts.Pattern != "" {
			fmt.Fprintf(os.Stderr, "- 検索パターン: %s\n", opts.Pattern)
		}
		if opts.Extension != "" {
			fmt.Fprintf(os.Stderr, "- 拡張子フィルタ: %s\n", opts.Extension)
		}
	}

	return nil
}

// estimateTokens は文字列のトークン数を推定する関数
// 簡易的な推定方法として、単語数とソースコードの特殊文字を考慮して計算
func estimateTokens(text string) int {
	// 単語数をカウント
	words := len(strings.Fields(text))

	// 特殊文字（記号や括弧など）をカウント
	symbols := 0
	for _, char := range text {
		if strings.ContainsRune("{}[]()<>+-*/=,.:;\"'!?@#$%^&", char) {
			symbols++
		}
	}

	// 改行文字をカウント
	newlines := strings.Count(text, "\n")

	// トークン数の推定（単語 + 記号の半分 + 改行数）
	// 実際のトークナイズはモデルによって異なりますが、これはおおよその推定値
	tokens := words + (symbols / 2) + newlines

	// 最低でも文字数の1/4はトークンとしてカウント
	minTokens := len(text) / 4
	if tokens < minTokens {
		tokens = minTokens
	}

	return tokens
}

// truncateContent は、コンテンツをトークン制限に合わせて切り詰める関数
func truncateContent(content string, maxTokens int) string {
	scanner := bufio.NewScanner(strings.NewReader(content))
	var truncated strings.Builder
	currentTokens := 0

	for scanner.Scan() {
		line := scanner.Text()
		lineTokens := estimateTokens(line)

		if currentTokens+lineTokens > maxTokens {
			truncated.WriteString("... (内容が長すぎるため切り詰められました)\n")
			break
		}

		truncated.WriteString(line)
		truncated.WriteString("\n")
		currentTokens += lineTokens
	}

	return truncated.String()
}

// convertWildcardToRegexp は、ワイルドカードパターンを正規表現パターンに変換する関数です
func convertWildcardToRegexp(pattern string) string {
	// 特殊文字をエスケープ
	pattern = regexp.QuoteMeta(pattern)

	// ワイルドカード記号を正規表現に変換
	// *は任意の文字列にマッチ
	pattern = strings.ReplaceAll(pattern, "\\*", ".*")
	// ?は任意の1文字にマッチ
	pattern = strings.ReplaceAll(pattern, "\\?", ".")

	// 完全一致にするために文字列の先頭と末尾に ^ と $ を追加
	return "^" + pattern + "$"
}
