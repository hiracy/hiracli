#!/usr/bin/env bash
#set -xv
set -e

PROJECT_ROOT=`dirname $0`
BIN_NAME="hiracli"
BIN_DIR=${PROJECT_ROOT}/bin
SRC_DIR=${PROJECT_ROOT}/src
GO_VERSION=${GO_VERSION:-1.23.6}
TEMP_DIR=$(mktemp -d)

# バージョン管理関連の関数
get_current_version() {
    git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0"
}

bump_version() {
    current_version=$(get_current_version | sed 's/^v//')
    major=$(echo "$current_version" | cut -d. -f1)
    minor=$(echo "$current_version" | cut -d. -f2)
    patch=$(echo "$current_version" | cut -d. -f3)
    
    case "$1" in
        major)
            major=$((major + 1))
            minor=0
            patch=0
            ;;
        minor)
            minor=$((minor + 1))
            patch=0
            ;;
        patch|*)
            patch=$((patch + 1))
            ;;
    esac
    
    echo "v$major.$minor.$patch"
}

generate_changelog() {
    last_tag=$(git describe --tags --abbrev=0 2>/dev/null || git rev-list --max-parents=0 HEAD)
    git log --pretty=format:"%s" "$last_tag..HEAD" | grep -v '^Merge' || true
}

# スクリプト終了時に一時ディレクトリを削除する設定
trap "rm -rf ${TEMP_DIR}" EXIT

# OSとアーキテクチャの検出を改善
OS=$(uname -s)
ARCH=$(uname -m)

# OSの判定
case "$OS" in
  "Darwin")
    GOOS="darwin"
    ;;
  "Linux")
    GOOS="linux"
    ;;
  *)
    echo "Error: Unsupported OS $OS"
    exit 1
    ;;
esac

# アーキテクチャの判定
case "$ARCH" in
  "x86_64")
    GOARCH="amd64"
    ;;
  "arm64"|"aarch64")
    GOARCH="arm64"
    ;;
  *)
    echo "Error: Unsupported architecture $ARCH"
    exit 1
    ;;
esac

# Goのインストールチェックと再インストール処理をcase文の中に移動
case "$1" in
  "-g"|"--go-setup")
    CURRENT_GO_VERSION=$(go version 2>/dev/null | awk '{print $3}' | sed 's/go//')
    echo "Current Go version: $CURRENT_GO_VERSION"
    echo "Target Go version: $GO_VERSION"
    if [ "$CURRENT_GO_VERSION" != "$GO_VERSION" ]; then
      echo "Reinstalling Golang..."
      if ! curl -L https://go.dev/dl/go${GO_VERSION}.${GOOS}-${GOARCH}.tar.gz -o ${TEMP_DIR}/go.tar.gz; then
        echo "Error: Failed to download Go ${GO_VERSION}"
        exit 1
      fi
      if [ -d "/usr/local/go${GO_VERSION}" ]; then
        sudo rm -rf /usr/local/go${GO_VERSION}
      fi
      if ! sudo tar -C /usr/local -xzf ${TEMP_DIR}/go.tar.gz; then
        echo "Error: Failed to extract Go archive"
        exit 1
      fi
      sudo mv /usr/local/go /usr/local/go${GO_VERSION}
      rm -rf ${TEMP_DIR}
      # シンボリックリンクを作成
      sudo ln -sf /usr/local/go${GO_VERSION}/bin/go /usr/local/bin/go
      if ! go version; then
        echo "Error: Go installation verification failed"
        exit 1
      fi
    fi
    ;;
  "-t"|"--test")
    echo "Running tests for hiracli..."

    # # Check if go.mod exists and has required dependencies
    # if [ ! -f "go.mod" ] || ! grep -q "github.com/hiracy/hiracli" "go.mod"; then
    #   echo "Error: Required dependencies not found."
    #   echo "Please run '$0 --init' first to set up the project dependencies."
    #   exit 1
    # fi

    # Run Go unit tests first
    echo "\nRunning Go unit tests..."
    if ! go test -v ./...; then
      echo "✗ Go unit tests failed"
      exit 1
    fi
    echo "✓ Go unit tests passed"

    # Build the binary
    echo "\nBuilding hiracli binary..."
    mkdir -p ${BIN_DIR}
    GOOS=${GOOS} GOARCH=${GOARCH} go build -o ${BIN_DIR}/hiracli ./cmd/hiracli
    chmod +x ${BIN_DIR}/hiracli

    echo "\nRunning integration tests..."

    # TODO: インテグレーションテストを追加
  
    # Summary
    echo "\nTest Summary:"
    if [ $FAILED_TESTS -eq 0 ]; then
      echo "All integration tests passed! ✓"
      exit 0
    else
      echo "$FAILED_TESTS integration test(s) failed ✗"
      exit 1
    fi
    ;;
  "-i"|"--init")
    MODULE_NAME=${MODULE_NAME:-hiracli}
    cd "${PROJECT_ROOT}"

    # Initialize go module if not exists
    if [ ! -f "go.mod" ]; then
      echo "Initializing go module..."
      go mod init $MODULE_NAME
    fi

    # Get required dependencies
    echo "Getting dependencies..."
    go get -u github.com/PagerDuty/go-pagerduty
    go get github.com/joho/godotenv
    go get github.com/aws/aws-sdk-go-v2/config
    go get github.com/aws/aws-sdk-go-v2/service/bedrock
    go mod tidy

    echo "Project initialization completed successfully."
    ;;
  "-a"|"--apply-completion")
    # シェル補完スクリプトの生成
    COMPLETION_DIR="${HOME}/.hiracli"
    mkdir -p "${COMPLETION_DIR}"
    
    # bashの補完スクリプト
    cat > "${COMPLETION_DIR}/hiracli.bash" << 'EOF'
_hiracli_completion() {
    local cur prev opts
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
    opts="llm git help"

    case "${prev}" in
        "llm")
            COMPREPLY=( $(compgen -W "list ask flatten-src help" -- ${cur}) )
            return 0
            ;;
        "git")
            COMPREPLY=( $(compgen -W "diff-comment help" -- ${cur}) )
            return 0
            ;;
        *)
            if [[ ${cur} == -* ]]; then
                case "${COMP_WORDS[1]}" in
                    "git")
                        case "${COMP_WORDS[2]}" in
                            "diff-comment")
                                COMPREPLY=( $(compgen -W "--llm" -- ${cur}) )
                                ;;
                        esac
                        ;;
                    "llm")
                        case "${COMP_WORDS[2]}" in
                            "ask")
                                COMPREPLY=( $(compgen -W "--llm --debug -d" -- ${cur}) )
                                ;;
                            "flatten-src")
                                COMPREPLY=( $(compgen -W "--pattern --extension --path -p --depth-limit --max-input-tokens --debug -d" -- ${cur}) )
                                ;;
                        esac
                        ;;
                esac
                return 0
            fi
            COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
            return 0
            ;;
    esac
}

complete -F _hiracli_completion hiracli
EOF

    # zshの補完スクリプト
    cat > "${COMPLETION_DIR}/hiracli.zsh" << 'EOF'
#compdef hiracli

_hiracli() {
    local -a commands subcmds
    commands=(
        'llm:LLM関連のコマンド'
        'git:Git関連のコマンド'
        'help:ヘルプの表示'
    )

    _arguments -C \
        '1: :->command' \
        '*:: :->args'

    case $state in
        command)
            _describe 'command' commands
            ;;
        args)
            case $words[1] in
                git)
                    subcmds=(
                        'diff-comment:Git差分からコミットメッセージを生成'
                        'help:Gitコマンドのヘルプ'
                    )
                    _describe 'git commands' subcmds
                    case $words[2] in
                        diff-comment)
                            _arguments \
                                '--llm[LLMモデルを指定]:model:(anthropic.claude-3-5-sonnet-20240620-v1:0 amazon.titan-text-express-v1)'
                            ;;
                    esac
                    ;;
                llm)
                    subcmds=(
                        'list:利用可能なLLMモデルの一覧表示'
                        'ask:LLMに質問する'
                        'flatten-src:ファイルをLLMチャットに適した形式で表示'
                        'help:LLMコマンドのヘルプ'
                    )
                    _describe 'llm commands' subcmds
                    case $words[2] in
                        ask)
                            _arguments \
                                '--llm[LLMモデルを指定]:model:(anthropic.claude-3-5-sonnet-20240620-v1:0 amazon.titan-text-express-v1)' \
                                '(-d --debug)'{-d,--debug}'[デバッグモードを有効にする]'
                            ;;
                        flatten-src)
                            _arguments \
                                '--pattern[ファイルを検索する正規表現パターン]:pattern:' \
                                '--extension[ファイル拡張子でフィルタリング]:extension:' \
                                '(-p --path)'{-p,--path}'[検索を開始するディレクトリパス]:directory:_files -/' \
                                '--depth-limit[ディレクトリ探索の深さ制限]:depth:(5 10 15 20)' \
                                '--max-input-tokens[最大トークン数]:tokens:(50000 100000 200000 300000)' \
                                '(-d --debug)'{-d,--debug}'[デバッグモードを有効にする]'
                            ;;
                    esac
                    ;;
            esac
            ;;
    esac
}

_hiracli
EOF

    # シェル設定ファイルに補完スクリプトを追加
    echo "\nSetting up shell completion..."
    
    # bashの場合
    if [ -f "${HOME}/.bashrc" ]; then
        if ! grep -q "hiracli.bash" "${HOME}/.bashrc"; then
            echo "source ${COMPLETION_DIR}/hiracli.bash" >> "${HOME}/.bashrc"
            echo "Added bash completion to .bashrc"
        fi
    fi
    
    # zshの場合
    if [ -f "${HOME}/.zshrc" ]; then
        if ! grep -q "hiracli.zsh" "${HOME}/.zshrc"; then
            echo "fpath=(${COMPLETION_DIR} \$fpath)" >> "${HOME}/.zshrc"
            echo "autoload -U compinit && compinit" >> "${HOME}/.zshrc"
            echo "Added zsh completion to .zshrc"
        fi
    fi

    # 現在のシェルを判別して適切なメッセージを表示
    CURRENT_SHELL=$(basename "$SHELL")
    case "$CURRENT_SHELL" in
        "bash")
            echo "\nシェル補完のセットアップが完了しました。"
            echo "以下のコマンドを実行して補完機能を有効化してください："
            echo "  source ~/.bashrc"
            ;;
        "zsh")
            echo "\nシェル補完のセットアップが完了しました。"
            echo "以下のコマンドを実行して補完機能を有効化してください："
            echo "  source ~/.zshrc"
            ;;
        *)
            echo "\nシェル補完のセットアップが完了しました。"
            echo "シェルを再起動するか、以下のいずれかのコマンドを実行して補完機能を有効化してください："
            echo "Bashの場合: source ~/.bashrc"
            echo "zshの場合:  source ~/.zshrc"
            ;;
    esac
    ;;
  "-h"|"--help")
    echo "Usage: $0 [OPTION]"
    echo "Options:"
    echo "  -g|--go-setup     Reinstall Golang if the version is different"
    echo "  -i|--init         Initialize the Go module"
    echo "  -u|--bump-up-version Bump up version and create a new release tag"
    echo "  -b|--build        Build the Go project"
    echo "  -f|--fmt          Format the Go source code"
    echo "  -t|--test         Run tests for hiracli"
    echo "  -a|--apply-completion Install shell completion for hiracli"
    echo "  -h|--help         Display this help message"
    ;;
  "")
    mkdir -p ${BIN_DIR}
    GOOS=${GOOS} GOARCH=${GOARCH} go build -o ${BIN_DIR}/hiracli ./cmd/hiracli
    chmod +x ${BIN_DIR}/hiracli
    ;;
  "-u"|"--bump-up-version")
    shift
    VERSION_TYPE=${1:-patch}
    NEW_VERSION=$(bump_version "$VERSION_TYPE")
    CHANGELOG=$(generate_changelog)
    
    echo "Current version: $(get_current_version)"
    echo "New version: $NEW_VERSION"
    echo "\nChangelog:\n$CHANGELOG"
    
    # タグを作成してプッシュ
    git tag -a "$NEW_VERSION" -m "Release $NEW_VERSION\n\n$CHANGELOG"
    git push origin "$NEW_VERSION"
    ;;
  "-b"|"--build")
    mkdir -p ${BIN_DIR}
    GOOS=${GOOS} GOARCH=${GOARCH} go build -o ${BIN_DIR}/hiracli ./cmd/hiracli
    chmod +x ${BIN_DIR}/hiracli
    ;;
  "-f"|"--fmt")
    go fmt ./...
    ;;
  *)
    # go get -u github.com/***/***
    go mod tidy
    ;;
esac
