# tnnl

`tnnl` は、Amazon ECS の `exec` / `port forward` 操作をインタラクティブに実行するための CLI ツールです。  
通常は `cluster`、`task`、`container` を個別指定する必要がありますが、`tnnl` では一覧から選択して実行できます。

## 主な機能

- `exec`: ECS Exec（`ecs execute-command` 相当）を実行
- `portforward`: `AWS-StartPortForwardingSession` を使ってタスクへポートフォワード
- `remoteportforward`: `AWS-StartPortForwardingSessionToRemoteHost` を使ってリモートホストへポートフォワード
- `update`: GitHub Releases から最新バージョンを取得して `tnnl` 自身を更新
- `make-input-file`: 各コマンド用の入力 JSON テンプレートを生成

## 前提条件

- AWS 認証情報が利用可能であること（AWS SDK のデフォルト設定を利用）
- `session-manager-plugin` コマンドがインストールされ、`PATH` から実行できること
- ECS / SSM を実行する IAM 権限があること
- `exec` 対象タスクで ECS Exec が有効化されていること

## インストール

```bash
go install github.com/wim-web/tnnl@latest
```

またはリリースバイナリを利用:  
https://github.com/wim-web/tnnl/releases

## 使い方

### バージョン確認

```bash
tnnl version
tnnl -v
```

### 1. exec

ECS タスクのコンテナへシェルコマンドを実行します。

```bash
tnnl exec --command sh
```

主なオプション:

- `--command` (デフォルト: `sh`)
- `--wait` (秒。タスク起動待ち時間。デフォルト: `0`)
- `--input-file` (入力 JSON ファイルパス)

### 2. portforward

タスクのポートへローカルからフォワードします。

```bash
tnnl portforward -t 5432 -l 15432
```

主なオプション:

- `-t, --target-port` (必須)
- `-l, --local-port` (省略時は空きポートを自動割当)
- `--input-file` (入力 JSON ファイルパス)

### 3. remoteportforward

タスク経由でリモートホストのポートへフォワードします。

```bash
tnnl remoteportforward -r 3306 --host db.example.local -l 13306
```

主なオプション:

- `-r, --remote-port` (必須)
- `--host` (必須)
- `-l, --local-port` (省略時は空きポートを自動割当)
- `--input-file` (入力 JSON ファイルパス)

### 4. update

`tnnl` の実行バイナリを最新リリースへ更新します。

```bash
tnnl update
```

補足:

- GitHub Releases へアクセスできるネットワーク環境が必要です。
- 実行中の `tnnl` バイナリ配置先に書き込み権限が必要です。

## インタラクティブ選択の挙動

- `cluster` 未指定時はクラスタ一覧から選択
- `service` 指定時はそのサービスに紐づく実行中タスクに絞り込み
- タスクが 1 件のみなら自動選択、複数なら一覧選択
- コンテナが 1 件のみなら自動選択、複数なら一覧選択
- 一覧画面は `q` または `Ctrl+C` で終了可能

## 入力ファイル（JSON）

`--input-file` で JSON を渡すと、対話入力の一部を省略できます。  
テンプレートは次のコマンドで生成できます。

```bash
tnnl exec make-input-file
tnnl portforward make-input-file
tnnl remoteportforward make-input-file
```

### exec-input.json

```json
{
  "cluster": "my-cluster",
  "service": "my-service",
  "command": "sh",
  "wait": 0
}
```

### portforward-input.json

```json
{
  "cluster": "my-cluster",
  "service": "my-service",
  "target_port_number": "5432",
  "local_port_number": "15432"
}
```

### remoteportforward-input.json

```json
{
  "cluster": "my-cluster",
  "service": "my-service",
  "remote_port_number": "3306",
  "local_port_number": "13306",
  "host": "db.example.local"
}
```

## 補足

- `cluster` / `service` を入力ファイルに書かない場合は、実行時に一覧選択されます。
- `portforward` / `remoteportforward` のローカルポートは、省略すると自動で空きポートが選ばれます。
