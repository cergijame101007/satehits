# さて、羊に戻るとしよう - 予約システム設計書

飲食店「さて、羊に戻るとしよう」のホームページ兼テーブル予約管理システムの設計ドキュメントです。

## ドキュメント一覧

| # | ドキュメント | ファイル | 説明 |
|---|--------------|----------|------|
| 1 | [ドメイン知識](./01_domain_knowledge.md) | 01_domain_knowledge.md | 店舗情報、営業ルール、用語集 |
| 2 | [ユースケース図](./02_use_case.md) | 02_use_case.md | アクター、ユースケース一覧、詳細 |
| 3 | [テーブル設計](./03_table_design.md) | 03_table_design.md | ER図、テーブル定義、DDL |
| 4 | [API設計（OpenAPI）](./openapi.yaml) | openapi.yaml | OpenAPI 3.0形式のAPI仕様 |
| 5 | [アーキテクチャ設計](./05_architecture.md) | 05_architecture.md | レイヤー構成、ディレクトリ構成、コード例 |
| 6 | [シーケンス図](./06_sequence.md) | 06_sequence.md | 主要機能のシーケンス図 |
| 7 | [データフロー図](./07_data_flow.md) | 07_data_flow.md | システム全体・機能別のデータフロー |
| 8 | [画面遷移図](./08_screen_transition.md) | 08_screen_transition.md | 画面一覧、ワイヤーフレーム、遷移マトリクス |
| 9 | [インフラ構成図](./09_infrastructure.md) | 09_infrastructure.md | 技術スタック、環境構成、CI/CD |

## システム概要

### 目的
- 顧客がオンラインで予約申請できる
- オーナーが予約を管理できる（承認/拒否）
- 日別のスケジュール・提供可能数を管理できる
- お取り引き先を紹介できる

### 主要機能

**顧客向け**
- 残り食数の確認
- 月間スケジュール（カレンダー）確認
- 予約の申請
- お取り引き先紹介ページの閲覧

**オーナー向け**
- 予約一覧の確認（日付検索）
- 予約の承認/拒否
- 予約の手動登録（Instagram/電話/知人経由）
- スケジュールの設定（通常/朝営業/イベント/特別メニュー/臨時休業）
- お取り引き先の管理（追加/編集/削除/並び替え）

### 技術スタック

| レイヤー | 技術 |
|----------|------|
| フロントエンド | Next.js 14 (App Router), TypeScript, Tailwind CSS |
| バックエンド | Go (net/http), レイヤードアーキテクチャ |
| データベース | Supabase (PostgreSQL) |
| インフラ | Vercel, Google Cloud Run, Docker |
| CI/CD | GitHub Actions |
| セキュリティ | JWT, reCAPTCHA v3, bcrypt |

### アーキテクチャパターン

| パターン | 用途 |
|----------|------|
| レイヤードアーキテクチャ | 全体構成（Presentation / Application / Domain / Infrastructure） |
| Repository パターン | データアクセスの抽象化、テスト容易性 |
| DI（依存性注入） | 疎結合、テスト時のモック化 |
| UseCase / Domain Service 分離 | 責務の明確化、肥大化防止 |

## OpenAPI仕様の確認方法

`openapi.yaml` をSwagger Editorで確認できます：

1. [Swagger Editor](https://editor.swagger.io/) を開く
2. `File` → `Import file` で `openapi.yaml` をアップロード
3. インタラクティブにAPIドキュメントを確認

## 開発フェーズ

### Phase 1（MVP）
- [ ] 顧客: 予約申請
- [ ] 顧客: 残り食数確認
- [ ] 顧客: 月間スケジュール（カレンダー）表示
- [ ] オーナー: ログイン
- [ ] オーナー: 予約一覧・承認・拒否
- [ ] オーナー: スケジュール設定（タイプ・提供数・イベント情報）
- [ ] オーナー: 予約手動登録（Instagram/電話経由）

### Phase 2
- [ ] 顧客: お取り引き先紹介ページ
- [ ] オーナー: お取り引き先管理
- [ ] 予約承認/拒否時のメール通知

### Phase 3（将来拡張）
- [ ] スケジュールからInstagram用画像を自動生成
- [ ] テーブル管理（テーブルへの予約割り当て）
- [ ] 予約履歴・統計ダッシュボード

## 関連リンク

- Instagram: [@satehits](https://www.instagram.com/satehits/)
- 間借り先: WINE LAB ([@winelab.nk](https://www.instagram.com/winelab.nk/))
