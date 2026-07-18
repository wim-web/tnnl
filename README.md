# tnnl

`tnnl` は、Amazon ECS の実行中タスクとコンテナを対話的に選び、ECS Exec、
ポートフォワード、リモートホストへのポートフォワードを開始するCLIです。
同じservice/groupのタスクが複数あっても、完全なTask ARNで選択対象を区別します。

## Quickstart

Go 1.25+、AWSの認証情報とRegion、Session Manager Pluginを用意します。

~~~bash
go version
go install github.com/wim-web/tnnl@latest

session-manager-plugin --version

export AWS_PROFILE=dev
export AWS_REGION=ap-northeast-1

tnnl version
tnnl exec --wait 30 --command sh
~~~

最後のコマンドでは、cluster、readyなtask、containerを順に選択します。`q`または
`Ctrl+C`で選択を中止できます。`tnnl`が見つからない場合は、次のインストール節で
`PATH`を確認してください。

## インストール

### Goからインストール

このリポジトリはGo 1.25+を必要とします。

~~~bash
go install github.com/wim-web/tnnl@latest
tnnl version
~~~

`go install`には、実行したGo toolchainが対応するmodule versionが埋め込まれます。
`tnnl: command not found`になる場合は、インストール先と`PATH`を確認します。

~~~bash
go env GOBIN
go env GOPATH
export PATH="$(go env GOPATH)/bin:$PATH"
command -v tnnl
~~~

`GOBIN`を明示している場合は、そのディレクトリを`PATH`へ追加してください。

### Release archiveからインストール

[GitHub Releases](https://github.com/wim-web/tnnl/releases)からOS/architectureに合う
`tnnl_<os>_<arch>.tar.gz`と`checksums.txt`を取得します。対応対象はDarwin/Linuxの
amd64/arm64です。archiveを展開し、`tnnl`を自分の`PATH`上の書き込み可能な
ディレクトリへ配置してください。

~~~bash
tar -xzf tnnl_linux_arm64.tar.gz
mkdir -p "$HOME/.local/bin"
install -m 0755 tnnl "$HOME/.local/bin/tnnl"
command -v tnnl
tnnl version
~~~

配布物を手動で検証する場合は、`checksums.txt`にある対象archiveのSHA-256と
ダウンロードしたファイルを照合してください。

## AWSコンテキスト

AWS context is caller-owned. 認証情報とRegionはAWS SDKのdefault configuration
chainから読み込みます。環境変数、共有`config`/`credentials`、SSOや実行環境の
IAM roleなど、AWS SDKが通常利用する設定を呼び出し前に用意してください。

`tnnl` intentionally has no profile/region flags（`--profile`、`--region`）です。
clusterとserviceは入力JSONで指定するか、実行時に対話選択します。

環境変数を使う例:

~~~bash
export AWS_PROFILE=dev
export AWS_REGION=ap-northeast-1
tnnl exec --wait 0
~~~

`aws-vault`を使う例:

~~~bash
aws-vault exec dev -- env AWS_REGION=ap-northeast-1 tnnl exec
~~~

実行前の確認:

~~~bash
aws sts get-caller-identity
aws configure list
session-manager-plugin --version
~~~

Session Manager Pluginの導入方法はAWS公式の
[Install the Session Manager plugin for the AWS CLI](https://docs.aws.amazon.com/systems-manager/latest/userguide/session-manager-working-with-install-plugin.html)
を参照してください。

## IAM: callerとECS task role

`tnnl`を起動するcallerのIAM権限と、対象containerが利用するECS task roleの権限は
別物です。以下は`tnnl`が呼び出すaction inventoryであり、全環境向けの
copy-and-paste policyではありません。実際の`Resource`と`Condition`はaccount、
cluster、task、session document、KMS keyに合わせて最小権限に絞ってください。

### Caller IAM

| Action | 使用箇所 |
| --- | --- |
| `ecs:ListClusters` | 全session commandのcluster選択 |
| `ecs:ListTasks` | 全session commandのtask検索 |
| `ecs:DescribeTasks` | readiness確認。execではsession作成後のruntime再取得にも使用 |
| `ecs:ExecuteCommand` | `tnnl exec`のみ |
| `ssm:StartSession` | `portforward`と`remoteportforward` |
| `ssm:TerminateSession` | remote session作成後のhandoff/plugin失敗時のcleanup。拒否された場合、そのcleanup errorは元のerrorへjoinされる |

Customer managed KMS keyでECS Exec通信を追加暗号化する場合、callerには
`kms:GenerateDataKey`が必要です。

### ECS task role

対象taskのtask role（task execution roleではありません）には、managed SSM agentが
channelを作成・接続するための次のactionが必要です。

- `ssmmessages:CreateControlChannel`
- `ssmmessages:CreateDataChannel`
- `ssmmessages:OpenControlChannel`
- `ssmmessages:OpenDataChannel`

Customer managed KMS keyによるECS Exec暗号化を使う場合は、task roleに
`kms:Decrypt`も必要です。

権限と前提条件の一次資料:

- [Amazon ECS Exec](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/ecs-exec.html)
- [Session Manager prerequisites](https://docs.aws.amazon.com/systems-manager/latest/userguide/session-manager-prerequisites.html)
- [Amazon ECS task IAM role: ECS Exec permissions](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/task-iam-roles.html#ecs-exec-permissions)
- [ECS actions, resources, and condition keys](https://docs.aws.amazon.com/service-authorization/latest/reference/list_ecs.html)
- [SSM actions, resources, and condition keys](https://docs.aws.amazon.com/service-authorization/latest/reference/list_ssm.html)

## Target readinessと選択表示

`exec`、`portforward`、`remoteportforward`はいずれも、次をすべて満たすtargetだけを
候補にします。

- taskの`enableExecuteCommand`が`true`で、task statusが`RUNNING`、Task ARNが存在する。
- container nameとruntime IDが存在し、container statusが`RUNNING`である。
- containerのmanaged agent名が`ExecuteCommandAgent`で、そのstatusが`RUNNING`である。

taskの表示labelにはservice/groupとshort task IDを含めます。内部のhidden valueは
full unique Task ARNです。同じservice/groupを持つ複数taskでも、選択した行のARNが
そのまま利用されます。

候補が1件なら自動選択し、複数なら一覧を表示します。eligibleなtaskまたはcontainerが
ない場合、remote sessionを作成せずに理由を返します。

## コマンド

### exec

container内でinteractive commandを実行します。command defaultは`sh`です。

~~~bash
tnnl exec --command sh
tnnl exec --wait 30 --command "sh -l"
tnnl exec --input-file exec-input.json --command bash
~~~

`--wait`の詳細は後述します。明示した`--command`と`--wait`は入力ファイルの値より
優先されます。

### portforward

containerのportへlocal portをforwardします。`--local-port`を省略するか、入力JSONの
`local_port_number`を空文字にすると、一時的に確保して解放した空きportを自動選択します。
literal `0`は自動選択の指定ではなく、範囲外としてerrorになります。

~~~bash
tnnl portforward --target-port 5432 --local-port 15432
tnnl portforward --target-port 5432
tnnl portforward --input-file portforward-input.json
~~~

### remoteportforward

ECS taskを経由してremote hostのportへforwardします。`--host`と`--remote-port`は
必須です。local portはflagを省略するか入力JSONを空文字にすると自動選択され、literal
`0`は範囲外としてerrorになります。

~~~bash
tnnl remoteportforward --host db.internal.example --remote-port 3306 --local-port 13306
tnnl remoteportforward --host db.internal.example --remote-port 3306
tnnl remoteportforward --input-file remoteportforward-input.json
~~~

### update

最新releaseを検証してから、実行中の`tnnl`を置換します。

~~~bash
tnnl update
~~~

配置先ディレクトリへの書き込み権限と、GitHub Releasesへのnetwork accessが必要です。
検証順序と失敗時の挙動は「Updateの検証と置換」を参照してください。

## Strict input JSONと優先順位

設定の優先順位は次のとおりです。

~~~text
explicit CLI flag > input file > default
~~~

この優先順位はcommand固有のflagに適用されます。clusterとserviceは入力JSONに書くか、
空のまま対話選択します。入力ファイルは`--input-file`で指定します。

decoderはstrictです。unknown keyと、1つ目のJSON objectに続くtrailing JSON documentは
errorになり、AWS APIを呼ぶ前に終了します。portはASCII decimal 1-65535のみ有効です。
`wait`は0以上でなければなりません。

入力フィールド:

| Command | JSON fields |
| --- | --- |
| 共通 | `cluster`、`service` |
| `exec` | `command`、`wait` |
| `portforward` | `target_port_number`、`local_port_number` |
| `remoteportforward` | `remote_port_number`、`local_port_number`、`host` |

templateは次のコマンドで生成します。

~~~bash
tnnl exec make-input-file
tnnl portforward make-input-file
tnnl remoteportforward make-input-file
~~~

default outputはそれぞれ`exec-input.json`、`portforward-input.json`、
`remoteportforward-input.json`です。既存ファイルは暗黙に上書きしません。
置換する場合だけ`--force`を明示します。

~~~bash
tnnl exec make-input-file --output exec-input.json --force
~~~

## Waitの正確な挙動

`--wait`は`exec`だけが持ち、cluster選択後のeligible task検索を制御します。

- `tnnl exec --wait 0`は論理的なeligible task lookupを一度だけ実行し、eligible taskが
  なければ直ちにerrorを返します。その1 lookup内でもpaginationとDescribeTasksの
  chunkingにより、AWS API callは複数回になり得ます。
- `tnnl exec --wait 30`のようなpositive waitは、eligible taskが現れるまでreadinessを
  pollし、指定秒数のtimeoutまたはcaller contextのcancellationで終了します。
- 入力JSONにserviceがあれば全pollをそのserviceへ絞り、なければ選択cluster内の
  eligible taskを待ちます。
- 成功時のtask/container候補は最後のDescribeTasks結果から作るため、wait前の古い
  runtime IDを再利用しません。

## Updateの検証と置換

`tnnl update`はlatest releaseのarchiveと`checksums.txt`をprivate temporary directoryへ
downloadし、対象asset名のSHA-256を選びます。

1. archiveをSHA-256で検証する。`checksum mismatch`なら展開しない。
2. 検証済みarchiveからcandidate binaryを展開する。
3. candidateの`version`をrelease tagと比較する。`candidate version mismatch`なら
   現行binaryを置換しない。
4. executableと同じdirectoryにunique temporary fileを作り、copy、chmod、sync、close後に
   atomic renameする。

checksum、展開、candidate version検証、rename前の置換処理で失敗した場合は、現行binaryの
内容とmodeを維持し、update自身が作ったtemporary fileだけをcleanupします。置換には
executable fileだけでなく、その親directoryへの書き込み権限が必要です。

SHA-256 checksumは破損やrelease manifestとの不一致を検出しますが、cryptographic
signature（署名）ではありません。`tnnl update`はsignature verificationや自動rollbackを
提供すると主張しません。

## トラブルシューティング

| 症状 | 主な原因 | 次に確認するコマンド・設定 |
| --- | --- | --- |
| credentialsまたはRegionが見つからない | caller-owned AWS contextが未設定、profile名違い、Region未設定 | `aws sts get-caller-identity`と`aws configure list`を実行し、effectiveなprofile・Region・取得元を確認して、`AWS_PROFILE`、`AWS_REGION`または共有AWS configを修正する |
| plugin not found、またはversion check失敗 | Session Manager Pluginが未導入、古い、`PATH`外 | `command -v session-manager-plugin`と`session-manager-plugin --version`を実行し、AWS公式install手順で導入・更新する |
| `AccessDenied` | caller IAMまたはtask roleのaction/resource/condition不足 | `aws sts get-caller-identity`でprincipalを確定し、errorに出たactionを上のCaller IAM/task role inventoryと照合する |
| eligible taskがない | task停止、ECS Exec無効、service指定違い | `aws ecs list-tasks --cluster CLUSTER`で候補を確認し、続けて`aws ecs describe-tasks --cluster CLUSTER --tasks TASK_ID`を実行する |
| agentまたはruntimeがreadyでない | container停止、`ExecuteCommandAgent`停止、container name/runtime ID欠落 | `aws ecs describe-tasks --cluster CLUSTER --tasks TASK_ID --query "tasks[].{exec:enableExecuteCommand,status:lastStatus,containers:containers[].{name:name,status:lastStatus,runtime:runtimeId,agents:managedAgents}}"`を実行する |
| wait timeout | 指定時間内にeligible taskが現れない、入力JSONのserviceが違う | 入力JSONの`service`を確認し、`tnnl exec --wait 30`の秒数を調整する。上の`describe-tasks`でもreadinessを確認する |
| local port conflict | explicit local portを別processがlisten中 | macOSでは`lsof -nP -iTCP:15432 -sTCP:LISTEN`、Linuxでは`ss -ltn`を確認する。必要なら`--local-port`を省略して自動選択する |
| `checksum mismatch` | download破損、proxy/cache、release assetとmanifestの組み合わせ違い | 同じreleaseのassetと`checksums.txt`を再取得し、`sha256sum ASSET_NAME`または`shasum -a 256 ASSET_NAME`の結果をmanifestの対象行と比較してから`tnnl update`を再実行する |
| `candidate version mismatch` | archiveのbinary versionとlatest tagが一致しない | `uname -s`、`uname -m`、`tnnl version`を確認し、GitHub Releasesのtagと対象assetを照合する |
| executable directoryへのpermission failure | symlink解決後の`tnnl`親directoryに書き込み権限がない | `ls -l "$(command -v tnnl)"`でsymlinkを確認し、`tnnl update`のerrorに表示された解決済みpathを`PATH_FROM_ERROR`へ置き換えて`ls -ld "$(dirname "PATH_FROM_ERROR")"`を実行する。必要ならuser-ownedな`PATH`へ再配置する |
