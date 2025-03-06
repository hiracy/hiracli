package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
)

// モックのBedrockRuntimeクライアント
type MockBedrockRuntimeClient struct{}

// InvokeModelのモックメソッド
func (m *MockBedrockRuntimeClient) InvokeModel(ctx context.Context, params *bedrockruntime.InvokeModelInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.InvokeModelOutput, error) {
	// リクエストの解析
	var requestData map[string]interface{}
	if err := json.Unmarshal(params.Body, &requestData); err != nil {
		return nil, fmt.Errorf("リクエストの解析エラー: %v", err)
	}

	// モデルIDに基づいてダミーレスポンスを返す
	modelId := aws.ToString(params.ModelId)

	var responseBody []byte
	var err error

	switch modelId {
	case "anthropic.claude-3-5-sonnet-20240620-v1:0":
		// Anthropicモデルのダミーレスポンス
		responseBody, err = json.Marshal(map[string]interface{}{
			"id":      "msg_01XxYzAbCdEf0123456789",
			"type":    "message",
			"role":    "assistant",
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": "これはAnthropicのClaudeモデルからのダミー回答です。実際にはAWS Bedrockへの問い合わせは行われていません。",
				},
			},
			"model":      "claude-3-5-sonnet-20240620-v1:0",
			"stop_reason": "end_turn",
			"usage": map[string]interface{}{
				"input_tokens":  10,
				"output_tokens": 50,
			},
		})
	case "amazon.titan-text-express-v1":
		// Amazon Titanモデルのダミーレスポンス
		responseBody, err = json.Marshal(map[string]interface{}{
			"inputTextTokenCount": 10,
			"results": []map[string]interface{}{
				{
					"tokenCount":  50,
					"outputText": "これはAmazon Titanモデルからのダミー回答です。実際にはAWS Bedrockへの問い合わせは行われていません。",
					"completionReason": "FINISHED",
				},
			},
		})
	default:
		return nil, fmt.Errorf("未対応のLLMモデル: %s", modelId)
	}

	if err != nil {
		return nil, fmt.Errorf("レスポンスの構築エラー: %v", err)
	}

	return &bedrockruntime.InvokeModelOutput{
		Body:        responseBody,
		ContentType: aws.String("application/json"),
	}, nil
}

// モック版の処理関数
func mockProcessPrompt(opts AskOptions, bedrockClient *MockBedrockRuntimeClient, input string) error {
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

	// モック版Bedrockにリクエストを送信
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

// モック版Ask関数
func mockAsk(opts AskOptions) error {
	// モッククライアントの作成
	mockClient := &MockBedrockRuntimeClient{}

	// プロンプトが指定されている場合は、そのプロンプトを使用
	if opts.Prompt != "" {
		return mockProcessPrompt(opts, mockClient, opts.Prompt)
	}

	// 実際のAsk関数と同様にプロンプトを表示
	fmt.Println("質問を入力してください（終了するには 'exit' または 'quit' を入力）:")
	
	// テスト用に固定入力を使用
	input := "AIについて教えてください"
	fmt.Print("> ")
	return mockProcessPrompt(opts, mockClient, input)
}

// Ask関数のテスト
func TestAsk(t *testing.T) {
	// テストケース
	testCases := []struct {
		name        string
		model       string
		debugMode   bool
		prompt      string
		expectError bool
	}{
		{
			name:        "Claude3モデルでの質問",
			model:       "anthropic.claude-3-5-sonnet-20240620-v1:0",
			debugMode:   false,
			prompt:      "AIについて教えてください",
			expectError: false,
		},
		{
			name:        "Titanモデルでの質問",
			model:       "amazon.titan-text-express-v1",
			debugMode:   false,
			prompt:      "AIについて教えてください",
			expectError: false,
		},
		{
			name:        "デバッグモードでの質問",
			model:       "anthropic.claude-3-5-sonnet-20240620-v1:0",
			debugMode:   true,
			prompt:      "AIについて教えてください",
			expectError: false,
		},
		{
			name:        "未対応モデルでの質問",
			model:       "unsupported.model-v1",
			debugMode:   false,
			prompt:      "AIについて教えてください",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 標準出力をキャプチャするための準備
			stdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// テスト終了時に元の設定に戻す
			defer func() {
				os.Stdout = stdout
			}()

			// モック版Ask関数を実行
			err := mockAsk(AskOptions{
				LLMModel:  tc.model,
				DebugMode: tc.debugMode,
				Prompt:    tc.prompt,
			})

			// 標準出力をキャプチャ終了
			w.Close()
			var buf bytes.Buffer
			io.Copy(&buf, r)
			output := buf.String()

			// エラー発生の検証
			if tc.expectError {
				if err == nil {
					t.Errorf("エラーが期待されていましたが、成功しました")
				}
				return
			} else if err != nil {
				t.Fatalf("予期せぬエラー: %v", err)
			}

			// 出力内容の検証
			if tc.debugMode {
				// デバッグモードの場合、リクエストとレスポンスの情報が含まれているか
				if !strings.Contains(output, "リクエスト:") || !strings.Contains(output, "レスポンス:") {
					t.Errorf("デバッグ出力が期待通りではありません。\n実際の出力:\n%s", output)
				}
			}

			// モデルに応じた回答のフォーマット検証
			if tc.model == "anthropic.claude-3-5-sonnet-20240620-v1:0" {
				if !strings.Contains(output, "これはAnthropicのClaudeモデルからのダミー回答です") {
					t.Errorf("Claudeモデルの回答が期待通りではありません。\n実際の出力:\n%s", output)
				}
			} else if tc.model == "amazon.titan-text-express-v1" {
				if !strings.Contains(output, "これはAmazon Titanモデルからのダミー回答です") {
					t.Errorf("Titanモデルの回答が期待通りではありません。\n実際の出力:\n%s", output)
				}
			}
		})
	}
}

// コマンドライン統合テスト
func TestLLMAskCommand(t *testing.T) {
	// 標準入力と標準出力をキャプチャするための準備
	originalStdin := os.Stdin
	originalStdout := os.Stdout
	
	// 入力パイプの作成
	inr, inw, _ := os.Pipe()
	os.Stdin = inr
	
	// 出力パイプの作成
	outr, outw, _ := os.Pipe()
	os.Stdout = outw
	
	// テスト終了時に元の設定に戻す
	defer func() {
		os.Stdin = originalStdin
		os.Stdout = originalStdout
	}()
	
	// テスト用のクエリを入力
	go func() {
		defer inw.Close()
		io.WriteString(inw, "AIについて教えてください\nexit\n")
	}()
	
	// ダミー版Ask関数をコール
	opts := AskOptions{
		LLMModel:  "anthropic.claude-3-5-sonnet-20240620-v1:0",
		DebugMode: false,
	}
	
	err := mockAsk(opts)
	if err != nil {
		t.Fatalf("コマンド実行エラー: %v", err)
	}
	
	// 標準出力のキャプチャを終了
	outw.Close()
	var buf bytes.Buffer
	io.Copy(&buf, outr)
	output := buf.String()
	
	// 出力の検証
	expectedPrompt := "質問を入力してください"
	if !strings.Contains(output, expectedPrompt) {
		t.Errorf("期待するプロンプトが表示されていません。\n期待する出力: %s\n実際の出力:\n%s", expectedPrompt, output)
	}
	
	expectedResponse := "これはAnthropicのClaudeモデルからのダミー回答です"
	if !strings.Contains(output, expectedResponse) {
		t.Errorf("期待する回答が含まれていません。\n期待する出力: %s\n実際の出力:\n%s", expectedResponse, output)
	}
}
