package llm

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
)

// AskOptions は、AIに質問を行う際のオプションを定義する構造体です
type AskOptions struct {
	LLMModel  string
	DebugMode bool
	Prompt    string // プロンプトを直接指定する場合に使用
}

// Ask は、指定されたLLMに対して質問を行い、回答を取得する関数です
func Ask(opts AskOptions) error {
	// AWSの設定を読み込み
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return fmt.Errorf("AWS設定の読み込みエラー: %v", err)
	}

	// BedrockRuntimeクライアントの作成
	bedrockClient := bedrockruntime.NewFromConfig(cfg)

	// プロンプトが指定されている場合は、そのプロンプトを使用
	if opts.Prompt != "" {
		return processPrompt(opts, bedrockClient, opts.Prompt)
	}

	fmt.Println("質問を入力してください（終了するには 'exit' または 'quit' を入力）:")
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		input := scanner.Text()
		if input == "exit" || input == "quit" {
			break
		}

		if input == "" {
			continue
		}

		if err := processPrompt(opts, bedrockClient, input); err != nil {
			return err
		}
	}

	return nil
}

func processPrompt(opts AskOptions, bedrockClient *bedrockruntime.Client, input string) error {
	// モデルに応じてリクエストを構築
	var payload []byte
	var err error

	switch opts.LLMModel {
	case "anthropic.claude-3-5-sonnet-20240620-v1:0":
		payload, err = json.Marshal(map[string]interface{}{
			"anthropic_version": "bedrock-2023-05-31",
			"max_tokens":        1000,
			"messages": []map[string]string{
				{
					"role":    "user",
					"content": input,
				},
			},
		})
	case "amazon.titan-text-express-v1":
		payload, err = json.Marshal(map[string]interface{}{
			"inputText": input,
			"textGenerationConfig": map[string]interface{}{
				"maxTokenCount": 1000,
				"stopSequences": []string{},
				"temperature":   0.7,
				"topP":          0.9,
			},
		})
	default:
		return fmt.Errorf("未対応のLLMモデル: %s", opts.LLMModel)
	}

	if err != nil {
		return fmt.Errorf("リクエストの構築エラー: %v", err)
	}

	if opts.DebugMode {
		fmt.Printf("リクエスト:\n%s\n\n", string(payload))
	}

	// Bedrockにリクエストを送信
	output, err := bedrockClient.InvokeModel(context.TODO(), &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(opts.LLMModel),
		Body:        payload,
		ContentType: aws.String("application/json"),
	})
	if err != nil {
		return fmt.Errorf("モデル呼び出しエラー: %v", err)
	}

	// レスポンスの解析
	var response map[string]interface{}
	if err := json.Unmarshal(output.Body, &response); err != nil {
		return fmt.Errorf("レスポンスの解析エラー: %v", err)
	}

	if opts.DebugMode {
		fmt.Printf("レスポンス:\n%+v\n\n", response)
	}

	// モデルに応じてレスポンスを抽出
	var answer string
	switch opts.LLMModel {
	case "anthropic.claude-3-5-sonnet-20240620-v1:0":
		if content, ok := response["content"].([]interface{}); ok && len(content) > 0 {
			if first, ok := content[0].(map[string]interface{}); ok {
				if text, ok := first["text"].(string); ok {
					answer = text
				}
			}
		}
	case "amazon.titan-text-express-v1":
		if results, ok := response["results"].([]interface{}); ok && len(results) > 0 {
			if first, ok := results[0].(map[string]interface{}); ok {
				if text, ok := first["outputText"].(string); ok {
					answer = text
				}
			}
		}
	}

	if answer == "" {
		return fmt.Errorf("レスポンスから回答を抽出できませんでした")
	}

	fmt.Printf("\n%s\n\n", answer)
	return nil
}
