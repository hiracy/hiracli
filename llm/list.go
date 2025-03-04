package llm

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrock"
)

// BedrockClientAPI はBedrock APIのインターフェースです（テスト用にモック可能）
type BedrockClientAPI interface {
	ListFoundationModels(ctx context.Context, params *bedrock.ListFoundationModelsInput, optFns ...func(*bedrock.Options)) (*bedrock.ListFoundationModelsOutput, error)
}

// デフォルトのBedrockクライアント生成関数
var newBedrockClient = func(cfg interface{}) BedrockClientAPI {
	return bedrock.NewFromConfig(cfg.(aws.Config))
}

// ListModels は、利用可能なLLMモデルの一覧を表示する関数です
func ListModels() error {
	// AWSの設定を読み込み
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return fmt.Errorf("AWS設定の読み込みエラー: %v", err)
	}

	// Bedrockクライアントの作成
	bedrockClient := newBedrockClient(cfg)

	// 利用可能なモデルの取得
	output, err := bedrockClient.ListFoundationModels(context.TODO(), &bedrock.ListFoundationModelsInput{})
	if err != nil {
		return fmt.Errorf("モデル一覧の取得エラー: %v", err)
	}

	// モデル情報の表示
	fmt.Println("\n利用可能なLLMモデル:")
	for _, model := range output.ModelSummaries {
		fmt.Printf("\nモデル: %s\n", *model.ModelId)
		if model.ModelName != nil {
			fmt.Printf("名前: %s\n", *model.ModelName)
		}
		if model.ProviderName != nil {
			fmt.Printf("プロバイダー: %s\n", *model.ProviderName)
		}
		if model.InputModalities != nil {
			fmt.Printf("入力モダリティ: %v\n", model.InputModalities)
		}
		if model.OutputModalities != nil {
			fmt.Printf("出力モダリティ: %v\n", model.OutputModalities)
		}
		if len(model.CustomizationsSupported) > 0 {
			fmt.Printf("カスタマイズ可能: %v\n", model.CustomizationsSupported)
		}
	}

	return nil
}
