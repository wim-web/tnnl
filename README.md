# tnnl

`tnnl`は、Amazon ECSの実行中タスクとコンテナを対話的に選び、ECS Execや
ポートフォワードを開始するCLIです。

## Quickstart

次のものを用意してください。

- Go 1.25以降
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
