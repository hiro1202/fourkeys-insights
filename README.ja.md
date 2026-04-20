# Four Keys Insights

GitHubのPRデータからDORA Four Keysメトリクスを算出するローカルダッシュボード。単一コンテナ、外部インフラ不要。`docker compose up` だけで起動。

**[English README](README.md)**

## 何ができるか

GitHubリポジトリのマージ済みPRを取得し、DORAの4つのメトリクスを計算してブラウザに表示します。

| メトリクス | 算出方法 |
|-----------|---------|
| **変更リードタイム** | 最初のコミット（またはIssue/PR作成）からPRマージまでの中央値 |
| **デプロイ頻度** | 期間内のPRマージ数（マージ=デプロイとして扱う） |
| **変更障害率** | インシデントルールに一致するPRの割合 |
| **サービス復元時間 (MTTR)** | インシデントPRのリードタイム中央値 |

各メトリクスにDORAレベル（Elite / High / Medium / Low）が付きます。

## クイックスタート

### 必要なもの

- Docker と Docker Compose
- GitHub Personal Access Token（fine-grained）
  - 必要な権限: **Pull requests (read)**、**Contents (read)**
  - Organizationリポを含める場合: トークンの **Resource owner** を個人アカウントではなくOrganizationに設定してください。Organization管理者によるトークンリクエストの承認が必要な場合があります
  - オプション: **Organization Members (read)** 権限を追加すると、将来のチームベースフィルタリング機能で使用できます

### 1. クローンと設定

```bash
git clone https://github.com/hiro1202/fourkeys-insights.git
cd fourkeys-insights
cp .env.example .env
```

`.env` にGitHubトークンを追加:

```
GITHUB_TOKEN=github_pat_xxxxxxxxxxxx
```

### 2. 起動

```bash
docker compose up
```

ブラウザで http://localhost:8080 を開きます。

### 3. セットアップウィザード

1. **トークン検証** - 「検証」ボタンでPATを確認
2. **リポジトリ選択** - 検索してトラッキングしたいリポを選択。PATでアクセス可能なリポのみ表示されます。OrganizationリポにはトークンのResource ownerをOrganizationに設定してください
3. **グループ作成** - グループ名を入力して同期開始

同期完了後、ダッシュボードが表示されます。

## 機能

- **マルチリポグルーピング** - 複数リポジトリを1つのチームとしてメトリクス集計
- **URLベースのグループ永続化** - 選択中のグループがURLパスに保存され、ブックマークやページリロードでコンテキストを維持
- **集計単位** - 週次（月〜日）または月次の期間でメトリクスカードとトレンドチャートを表示
- **リードタイム開始点の選択** - 最初のコミット / リンクされたIssue作成日 / PR作成日（変更リードタイムとMTTRで個別に設定可能）
- **トレンドチャート** - リードタイム、デプロイ頻度、変更障害率、MTTRの推移を3/6/12ヶ月で表示
- **リポ別フォールバックマーカー** - 設定ページでフォールバック開始点を使用しているリポを表示（変更リードタイムとMTTRを個別に表示）
- **クエリ時インシデント判定** - タイトル・ブランチ・ラベルのキーワードルールを設定可能。ルール変更時に再同期不要
- **Issue紐づけMTTR** - PRボディの `Closes #N` をパースし、Issue作成日をMTTR開始点として使用
- **ETag条件付きリクエスト** - 再同期時に変更のないPR一覧の再取得をスキップ
- **CSVエクスポート** - メトリクスサマリーとPR詳細のZIP（Excel互換UTF-8 BOM付き）
- **DORAリファレンスリンク** - ヘッダーからGoogle Cloud公式のDORA Four Keys解説記事へリンク（言語ごとにローカライズ）
- **DORAバッジ** - `/api/v1/groups/:id/badge` でSVGバッジを取得
- **ダークモード** - トグルまたはOS設定に追従
- **多言語対応** - 英語・日本語

## 設定

### 環境変数

| 変数名 | 説明 | デフォルト |
|-------|------|-----------|
| `GITHUB_TOKEN` | GitHub PAT（必須） | - |
| `APP_PORT` | サーバーポート | `8080` |
| `APP_BIND` | バインドアドレス | `localhost` |
| `LOG_LEVEL` | ログレベル (debug/info/warn/error) | `info` |

### 設定ファイル

`config/config.yaml` でも設定可能:

```yaml
app:
  port: 8080
  bind: "localhost"

github:
  token: ""
  api_base_url: "https://api.github.com"

log:
  level: "info"

fetch:
  concurrency: 1
```

優先順位: 環境変数 > config.yaml > デフォルト値

### グループ単位の設定（設定ページ）

- **集計単位** - 週次または月次
- **変更リードタイム開始点** - 最初のコミット / Issue作成日 / PR作成日
- **サービス復元時間開始点** - 同じ選択肢、変更リードタイムとは独立して設定可能。障害時のIssue自動起票と組み合わせると精度向上
- **インシデント検出ルール** - タイトルキーワード、ブランチキーワード、ラベル一致

## API

| メソッド | エンドポイント | 説明 |
|---------|--------------|------|
| POST | `/api/v1/auth/validate` | PAT検証 |
| GET | `/api/v1/repos` | アクセス可能なリポ一覧 |
| GET | `/api/v1/repos/:id/settings` | リポ設定取得 |
| PUT | `/api/v1/repos/:id/settings` | リポ設定更新 |
| GET | `/api/v1/groups` | グループ一覧 |
| POST | `/api/v1/groups` | グループ作成 |
| PUT | `/api/v1/groups/:id` | グループ更新 |
| DELETE | `/api/v1/groups/:id` | グループ削除 |
| GET | `/api/v1/groups/:id/metrics` | Four Keysメトリクス取得 |
| GET | `/api/v1/groups/:id/trends` | トレンドデータ取得 |
| GET | `/api/v1/groups/:id/settings` | グループ設定取得 |
| PUT | `/api/v1/groups/:id/settings` | グループ設定更新 |
| GET | `/api/v1/groups/:id/pulls` | PR一覧（ページネーション） |
| GET | `/api/v1/groups/:id/export` | CSVエクスポート（ZIP） |
| GET | `/api/v1/groups/:id/badge` | DORAレベルSVGバッジ |
| POST | `/api/v1/groups/:id/sync` | 同期ジョブ開始 |
| GET | `/api/v1/jobs/:id` | ジョブ状態取得 |
| POST | `/api/v1/jobs/:id/cancel` | ジョブキャンセル |

## 技術スタック

| レイヤー | 技術 |
|---------|------|
| バックエンド | Go (chi, go-github, zap, viper) |
| フロントエンド | React, Vite, Tailwind CSS, ECharts, TanStack Query |
| データベース | SQLite (WALモード) |
| コンテナ | Docker (マルチステージビルド, go:embed) |

## 開発

### バックエンド

```bash
cd backend
go run ./cmd/server/
```

### フロントエンド（ホットリロード付き開発サーバー）

```bash
cd frontend
npm install
npm run dev
```

Viteが `/api` を `localhost:8080` にプロキシします。

### テスト

```bash
# Goテスト（36テスト）
cd backend && CGO_ENABLED=1 go test ./...

# i18nキー検証
cd frontend && node scripts/check-i18n.js

# E2Eテスト（5テスト）
cd e2e && npm install && npx playwright install chromium && npx playwright test
```

## ドキュメント

- [DESIGN.md](DESIGN.md) - アーキテクチャ、DBスキーマ、API設計、メトリクス定義
- [TODOS.md](TODOS.md) - 延期された機能とPhase 2ロードマップ

## ライセンス

MIT Licensed. See [LICENSE](LICENSE) for full details.
