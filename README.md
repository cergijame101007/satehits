# さて、羊に戻るとしよう 🐑

飲食店「さて、羊に戻るとしよう」のホームページ兼テーブル予約管理システムです。

## 概要

顧客がオンラインで予約申請でき、オーナーが予約を管理（承認/拒否）できるシステムです。

### 主な機能

**顧客向け**
- 残り食数の確認
- 月間スケジュール（カレンダー）確認
- 予約の申請
- お取り引き先紹介ページの閲覧

**オーナー向け**
- 予約一覧の確認・承認・拒否
- 予約の手動登録（Instagram/電話/知人経由）
- スケジュールの設定（通常/朝営業/イベント/特別メニュー/臨時休業）
- お取り引き先の管理

## 技術スタック

| レイヤー | 技術 |
|----------|------|
| フロントエンド | Astro 5 + React 19 (Islands Architecture), TypeScript, Tailwind CSS v4 |
| バックエンド | Go (net/http), レイヤードアーキテクチャ |
| データベース | Supabase (PostgreSQL) |
| ドメイン/DNS | Cloudflare Registrar |
| インフラ | Cloudflare Pages, Google Cloud Run, Docker |
| メール配信 | Resend |
| CI/CD | GitHub Actions |

## ドキュメント

| ドキュメント | 説明 |
|--------------|------|
| [セットアップ](./docs/setup.md) | 開発環境の構築手順 |
| [ドメイン知識](./docs/domain_knowledge.md) | 店舗情報、営業ルール、用語集 |
| [API設計](./docs/api_design.md) | API仕様の概要 |
| [OpenAPI仕様](./docs/openapi.yaml) | OpenAPI 3.0形式のAPI仕様 |
| [アーキテクチャ](./docs/architecture.md) | レイヤー構成、ディレクトリ構成 |
| [シーケンス図](./docs/sequence.md) | 主要機能のシーケンス図 |
| [画面遷移図](./docs/screen_transition.md) | 画面一覧、遷移図、ワイヤーフレーム |
| [インフラ構成](./docs/infrastructure.md) | 技術スタック、環境構成、CI/CD |
| [テスト設計](./docs/test_design.md) | テスト方針、テストケース一覧 |
| [技術選定理由書](./docs/architecture_decision_records.md) | ADR（Architecture Decision Records） |

## クイックスタート

```bash
# リポジトリのクローン
git clone https://github.com/cergijame101007/satehits.git
cd satehits

# 環境変数の設定
cp .env.example .env
# .env を編集

# 開発サーバーの起動
make dev
```

詳細は [セットアップガイド](./docs/setup.md) を参照してください。

## 関連リンク

- Instagram: [@satehits](https://www.instagram.com/satehits/)
- 間借り先: WINE LAB ([@winelab.nk](https://www.instagram.com/winelab.nk/))
