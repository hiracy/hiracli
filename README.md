# hiracli

Hiracy Swiss Army Command Line Tool

## インストール

### GitHub Releasesからインストール

以下のコマンドで最新バージョンをダウンロードしてインストールできます：

```bash
# Linux (amd64)
curl -L https://github.com/hiracy/hiracli/releases/latest/download/hiracli_linux_amd64.tar.gz | tar xz -C /usr/local/bin

# Linux (arm64)
curl -L https://github.com/hiracy/hiracli/releases/latest/download/hiracli_linux_arm64.tar.gz | tar xz -C /usr/local/bin

# macOS (amd64)
curl -L https://github.com/hiracy/hiracli/releases/latest/download/hiracli_darwin_amd64.tar.gz | tar xz -C /usr/local/bin

# macOS (arm64)
curl -L https://github.com/hiracy/hiracli/releases/latest/download/hiracli_darwin_arm64.tar.gz | tar xz -C /usr/local/bin
```

### ソースからビルド

```bash
go install
```

## セットアップ

1. `.env.example`を`.env`にコピーし、必要な環境変数を設定します：

```bash
cp .env.example .env
```

2. 環境変数を設定：

- `A WS_ACCESS_KEY_ID`: AWSアクセスキーID
- `AWS_SECRET_ACCESS_KEY`: AWSシークレットアクセスキー
- `AWS_REGION`: AWSリージョン（デフォルト: ap-northeast-1）

3. セットアップスクリプトを実行：

```bash
# Goのセットアップ（必要な場合）
./setup.sh --go-setup

# プロジェクトの初期化
./setup.sh --init

# シェル補完のインストール
./setup.sh --apply-completion
```

## 使用方法

AIに質問する：

```bash
hiracli llm ask
hiracli llm ask --llm amazon.titan-text-express-v1
hiracli llm ask --debug
```

利用可能なLLMモデルを表示：

```bash
hiracli llm list
```

## 利用可能なコマンド

### LLM関連

- `llm list`: 利用可能なLLMモデルを表示
- `llm ask`: LLMに質問する
  - オプション：
    - `--llm`: LLMモデルを指定（デフォルト: anthropic.claude-3-5-sonnet-20240620-v1:0）
    - `--debug, -d`: デバッグモードを有効にする

## セットアップスクリプトのオプション

- `-g, --go-setup`: Goのバージョンが異なる場合に再インストール
- `-i, --init`: Goモジュールの初期化
- `-b, --build`: プロジェクトのビルド
- `-f, --fmt`: Goソースコードのフォーマット
- `-c, --copy-git-diff`: gitの差分をクリップボードにコピー
- `-t, --test`: hiraclのテストを実行
- `-a, --apply-completion`: シェル補完のインストール
- `-h, --help`: ヘルプメッセージの表示

## シェル補完

シェル補完機能を有効にするには、以下のコマンドを実行してください：

Bashの場合：

```bash
source ~/.bashrc
```

Zshの場合：

```bash
source ~/.zshrc
```
