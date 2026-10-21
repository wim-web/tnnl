# tnnl

`tnnl`は、Amazon ECSの実行中タスクとコンテナを対話的に選び、ECS Execや
ポートフォワードを開始するCLIです。

## Quickstart

次のものを用意してください。

- Go 1.27.3以降（1.26系は1.26.10以降）
- AWSの認証情報とRegion
- [Session Manager Plugin](https://docs.aws.amazon.com/systems-manager/latest/userguide/session-manager-working-with-install-plugin.html)
- ECS Execが有効なECSタスク

~~~bash
go install github.com/wim-web/tnnl@latest

export AWS_PROFILE=dev
export AWS_REGION=ap-northeast-1

tnnl exec
~~~

cluster、実行可能なtask、containerを順に選択します。`q`または`Ctrl+C`で選択を
中止できます。

利用できるコマンドとオプションはhelpを参照してください。

~~~bash
tnnl --help
tnnl exec --help
~~~

ビルド済みバイナリは[GitHub Releases](https://github.com/wim-web/tnnl/releases)からも
ダウンロードできます。対応対象はDarwin/Linuxのamd64/arm64です。

## Update

最新releaseへ更新するには、次を実行します。

~~~bash
tnnl update
~~~

実行ファイルの配置先ディレクトリへの書き込み権限と、GitHub Releasesへの
ネットワークアクセスが必要です。

## セキュリティ更新（2026年10月）

[eMudhra中間CAの秘密鍵漏えい](https://emudhra.we-have-a-problem.com/)に対して、
[Go 1.27.3 / 1.26.10](https://go.dev/doc/devel/release#go1.27.3)で証明書検証の対策が入りました。
tnnlはAWS APIやGitHub ReleasesへHTTPS接続するため、修正版Goでの再ビルドが必要です。
`go.mod`の`go`行は言語の最低要件です。Goのインストール更新だけでは、既存のtnnl実行ファイルは更新されません。

ソースから導入する場合は、修正版Goへ更新してから`go install github.com/wim-web/tnnl@latest`を
再実行してください。ビルド済みバイナリを使う場合は、修正版Goで再ビルドされたリリースへ更新してください。
2026年10月21日時点の最新版v0.6.33はGo 1.27.2でビルドされており、再配布が必要です。
実行ファイルのビルドに使われたGoは、次のコマンドで確認できます。

~~~bash
go version -m "$(command -v tnnl)"
~~~

開発・リリースのGoは`aqua.yaml`で固定しています。CIでは`GOTOOLCHAIN=local`を設定し、
公開前に`script/verify-release-artifacts.sh`で全配布アーカイブのGoバージョンを検査します。
