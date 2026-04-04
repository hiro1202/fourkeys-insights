# TODOS

## Deferred from v1

### SSO Error Handling
- **What:** Fine-grained PATでSSO必須のorgのリポにアクセスした際の明確なエラーメッセージを追加
- **Why:** SSOが有効なorgではPATに追加のSSO承認が必要。これがないとAPI 403が返るが、ユーザーには「権限不足」としか見えず原因が分からない
- **Pros:** SSO有効な組織のユーザーが自己解決できる
- **Cons:** SSO検出ロジックの実装が必要（403レスポンスのヘッダーからSSO要求を判別）
- **Context:** GitHub APIは SSO要求時に `X-GitHub-SSO` ヘッダーを返す。これをパースしてUIに「SSO承認が必要です」と表示する
- **Depends on:** v1のリポ検出機能が完成していること

### Repo Drilldown View
- **What:** グループ集計ダッシュボードからリポ単位のメトリクスにドリルダウンできるビュー
- **Why:** どのリポがグループ全体の数値を引き上げ/引き下げているかを可視化。現状は「1リポのグループ」で代替可能だが、UXが悪い
- **Pros:** チーム内のボトルネックリポを特定できる
- **Cons:** UI実装とリポ単位のメトリクス計算APIが必要
- **Context:** DBにはリポ単位のPRデータが既にあるので、バックエンドは `GET /api/v1/repos/:id/metrics` を追加するだけ。フロントはグループダッシュボードにドリルダウンリンクを追加
- **Depends on:** v1のグループ集計ダッシュボードが完成していること

### Incident Management Tool Integration
- **What:** PagerDuty, Jira等の障害管理ツールと連携し、本物のMTTR（障害検知→サービス復旧）を計算
- **Why:** v1のissue紐づけMTTRは良いプロキシだが、障害検知の時刻はGitHubからは取得できない。外部ツールの連携が必要
- **Pros:** DORAの定義に忠実なMTTRが計算可能に
- **Cons:** 外部ツールごとのAPI統合が必要。認証フローが複雑化
- **Context:** v1ではissue.created_atをMTTR開始時刻として使用。将来はPagerDutyのincident.created_atやJiraのissue作成日時で代替するインターフェースを設計
- **Depends on:** v1のメトリクス計算基盤が安定していること

### DESIGN.md Full Creation
- **Completed:** feat/dashboard-gaps (2026-04-05)
- **What:** エンジニアリングレビューとデザインレビューで確定した全決定事項を反映した正式なDESIGN.mdを作成
- **Context:** DESIGN.mdを作成済み。アーキテクチャ図、DBスキーマ、API設計（16エンドポイント）、UIフロー、i18n方針、テスト戦略を含む

## Phase 2 (post-v1)

### PostgreSQL対応
- **What:** db/パッケージのinterfaceにPostgreSQL実装を追加。docker-compose.ymlにPostgreSQLサービスを追加し、環境変数で切替可能にする
- **Why:** SQLiteはDocker volumeで永続化可能だが、マルチユーザー対応やCI連携を考えるとPostgreSQLが必要。ユーザーからも優先度高との判断
- **Pros:** データ永続性の安定化、将来のマルチユーザー対応、CI/CD連携の基盤
- **Cons:** docker compose upの「シンプルさ」がやや失われる（PostgreSQLコンテナ追加）
- **Context:** v1のdb/パッケージはinterface定義を持ち、SQLite実装はその実装の1つ。PostgreSQL実装を追加するだけの設計。SQLiteもデフォルトとして残す（ゼロ設定起動を維持）
- **Effort:** M (human) → S (CC+gstack)
- **Priority:** P1
- **Depends on:** v1完成

### Slack/Webhook通知
- **What:** メトリクス同期完了時にconfig.yamlで指定したWebhook URLにPOSTする
- **Why:** 週次レポートの自動化への布石。Slack/Teams/Discord等に結果を通知
- **Pros:** レポート作成の手間を更に削減
- **Cons:** Webhook認証やペイロード形式の設計が必要
- **Context:** config.yamlに `notification.webhook_url` を追加。同期完了時にメトリクスサマリーをJSON POSTする。テンプレートはSlack Block Kit互換
- **Effort:** S (human) → S (CC+gstack)
- **Priority:** P2
- **Depends on:** v1完成
