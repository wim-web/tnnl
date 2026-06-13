---
name: renovate-automerge
description: このリポジトリの Renovate PR を調査し、repo固有ルールに従ってマージする時に使う。
---

# Renovate Automerge

この skill は、`wim-web/tnnl` で Renovate が作成した open PR を確認し、以下のルールに従ってマージまたは報告する。

## 対象PR

- 作成者が Renovate の open PR のみを対象にする。
- このリポジトリで観測した Renovate PR author: `app/renovate`, `renovate[bot]`
- commit author として観測した Renovate author: `renovate[bot]`
- base branch が `main` の PR のみを対象にする。
- Dependabot や人間が作成した PR は対象外。

## リポジトリ前提

- remote: `https://github.com/wim-web/tnnl.git`
- default branch: `main`
- Renovate 設定: `renovate.json` は `local>wim-web/renovate-config` を extend している。
- package manager / manifest / lockfile:
  - Go: `go.mod`, `go.sum`
  - aqua: `aqua.yaml`
  - Terraform provider lock: `terraform/.terraform.lock.hcl`
- CI:
  - PR では `.github/workflows/test.yml` の workflow `Test` が `go mod download` と `go test -v -race ./...` を実行する。
  - branch protection は観測時点で未設定。ただし、この skill では `Test / test` を必須 check として扱う。
- Release:
  - `.github/workflows/release.yml` は tag push で GoReleaser を実行する。
  - release workflow は `v[0-9]+.[0-9]+.[0-9]+` に一致する tag push で起動する。
  - `.goreleaser.yml` は `before.hooks` で `go mod tidy` を実行し、`tnnl` binary を darwin/linux 向けに build する。
  - 観測時点の最新 tag は `v0.6.2`。
- Infra:
  - `terraform/` に AWS RDS/ECS/network 関連の Terraform ファイルがある。
  - Dockerfile / docker-compose は観測していない。

## 必ず確認すること

- PR title/body
- changed files
- update type
- Renovate がPR本文に載せた release notes / changelog / compatibility notes
- upstream changelog / release notes / migration guide
- 破壊的変更、deprecated API、設定変更、peer dependency変更、runtime要件変更の有無
- 影響範囲: runtime dependency / dev dependency / build tool / CI / Docker / infra / deploy / database
- check status
- merge conflict の有無
- requested changes / 未解決の人間 review comment の有無

## マージしてよいもの

以下を全て満たす PR だけマージしてよい。

- PR author が `app/renovate` または `renovate[bot]`
- open PR
- draft ではない
- base branch が `main`
- merge state が `CLEAN`
- human review が未承認でもよいが、`CHANGES_REQUESTED`、requested changes、未解決の人間 review comment、未解決の人間 comment がない
- 過去の `Renovate automerge review` コメントは、現在の repo-local skill と最新調査で未マージ理由が解消済みなら、未解決の人間 comment として扱わない
- 必須 check `Test / test` が `SUCCESS`
- Renovate PR body と upstream changelog / release notes / migration guide を確認し、breaking changes、deprecated API、設定変更、peer dependency変更、runtime要件変更がない
- changed files、依存の用途、release notes、CI結果から、この repo への影響範囲が小さいと具体的に説明できる
- changed files が、以下の許可パターンのいずれかだけに収まる

許可パターン:

- `go.mod` と `go.sum` だけを変更する Go module の patch/minor update
  - direct / indirect dependency のどちらも、PR body と upstream notes を確認して低影響と判断できるなら許可する。
  - AWS SDK、Charmbracelet 系、Cobra など runtime dependency でも、API breaking、設定変更、runtime要件変更、deprecated API の利用がなく、`Test / test` が成功しているなら許可する。
  - `module` path、`go` directive、toolchain 指定を変える PR はこのパターンでは許可しない。
- `aqua.yaml` だけを変更する patch/minor update
  - `aquaproj/aqua-registry`、`aquaproj/aqua`、`spf13/cobra-cli`、`golang/go`、`hashicorp/terraform` の更新は、upstream notes を確認して低影響と判断できるなら許可する。
  - `golang/go` は patch update を許可する。race detector や cross compile 関連ファイルに差分があっても、release notes に既存コード、module resolution、GoReleaser build、対応 platform への明示的な breaking change、migration、known regression がない場合はマージしてよい。minor update は release notes を確認し、GoReleaser build や runtime 要件への影響が低いと説明できる場合だけ許可する。
  - `hashicorp/terraform` は CLI pin の patch/minor update だけ許可する。Terraform code、provider lock、state、provider behavior に影響する変更を伴う場合はマージしない。
- `.github/workflows/test.yml` または `.github/workflows/release.yml` だけを変更する GitHub Actions / CI / release tool の patch/minor update
  - action は commit SHA pin のままで、Renovate が付ける version comment も新しい version と対応していること。
  - workflow の trigger、permissions、secrets、release args、artifact publish 先を変える PR はマージしない。
  - GoReleaser action の patch/minor update は、action release notes に breaking changes や required input / permission changes がなければ許可する。
- `terraform/.terraform.lock.hcl` だけを変更する Terraform provider の patch/minor update
  - provider release notes を確認し、既存 `terraform/*.tf` の resource / data source に破壊的変更、deprecated attribute、state migration、権限追加がないと判断できる場合だけ許可する。
  - `.tf` ファイルを変更する PR はこのパターンでは許可しない。
- security update
  - patch/minor update で、上記いずれかの許可パターンに収まり、breaking changes や運用変更がないと確認できる場合は許可する。

## マージしてはいけないもの

以下のいずれかに該当する PR はマージせず、理由を報告する。

- PR author が `app/renovate` または `renovate[bot]` ではない
- draft PR
- base branch が `main` ではない
- merge conflict がある、または merge state が `CLEAN` ではない
- 必須 check `Test / test` が failed / pending / missing
- requested changes、未解決の人間 review comment、未解決の人間 comment がある
- major update
- `go.mod` の `module` path、`go` directive、toolchain 指定を変更する PR
- source code、test code、migration、Terraform code を変更する PR
- runtime dependency / framework / AWS SDK / CLI 挙動への影響が大きい、または低影響と説明できない PR
- Terraform code、AWS infra、state、provider behavior、RDS/ECS/network への影響が大きい、または低影響と説明できない PR
- `.goreleaser.yml`、release args、workflow trigger、permissions、secrets、artifact publish に関係する PR
- Docker image / Dockerfile / docker-compose に関係する PR
- database / migration / RDS に関係する PR
- changelog / release notes / migration guide を確認できず、影響範囲を判断できない PR
- breaking changes / deprecated API / peer dependency変更 / runtime要件変更 / 設定変更の可能性が残る PR
- この skill の「マージしてよいもの」に明記されていない PR

マージしない Renovate PR がある場合:

- 最終報告だけで済ませてはいけない。対象 PR に GitHub comment で未マージ理由を残す。
- コメントには、確認した upstream release notes / changelog / migration guide のどの内容がこの skill の禁止条件に該当したかを具体的に書く。
- `Test / test` が成功していてもマージしない場合は、check 成功だけでは許可条件を満たさない理由を書く。
- comment 投稿に失敗した場合は、マージせず、投稿に失敗したことと理由を最終報告に含める。

## 必須 check

- `Test / test`
  - GitHub 上の workflow name は `Test`
  - check run name は `test`
  - conclusion は `SUCCESS` でなければならない

branch protection は観測時点で未設定だが、この skill では `Test / test` を必須 check として扱う。check が存在しない、pending、cancelled、skipped、failure の場合はマージしない。

## マージ方法

- `gh pr merge <PR番号> --squash --delete-branch` を使う。
- merge commit や rebase merge は使わない。
- squash commit message は Renovate の PR title を基本にし、不要な手編集をしない。
- open な Renovate PR を全て確認してから、マージ可能と判断した PR を順にマージする。
- 1 件マージするたびに、open な Renovate PR を再取得し、merge state / check status / review・comment 状態を再評価する。
- GitHub の merge state が `UNKNOWN` の PR がある場合は、release に進まず、少し待って再取得する。最終判断時点で `UNKNOWN` のままなら、その PR は「判断できないため未マージ」として扱う。
- merge state が `DIRTY` / `BLOCKED` / `UNKNOWN` から `CLEAN` に再計算されることがあるため、マージ試行失敗直後に release へ進んではいけない。open PR を再取得し、マージ可能な PR が残っていないことを確認してから release に進む。
- マージした PR が 1 件以上あり、かつ release tag が必要な変更を含む場合は、最終再評価でマージ可能な PR が 0 件になった後にだけ release tag を作成して push する。
- この skill の 1 回の実行で作成してよい release tag は最大 1 個。release tag を push した後に追加で Renovate PR をマージしたり、追加 release tag を作ったりしてはいけない。

## リリース方法

マージした PR が 1 件以上あり、かつ `tnnl` CLI の配布バイナリに入る dependency や build version に影響する場合だけ、以下を実行する。

Renovate による `.github/workflows/release.yml` の tool version pin / action version pin だけの patch/minor update は、trigger、permissions、secrets、release args、artifact publish 先を変えない限り、次回 release 時に自然に使われる workflow 更新として扱い、この skill の実行中には release tag を作らない。

release tag が必要な変更:

- `go.mod` / `go.sum` の変更
- `go` の build version を変える `aqua.yaml` の変更
- `.goreleaser.yml` の変更
- source code / test code の変更。ただし Renovate automerge では原則マージしない。

release tag が不要な変更:

- `aqua.yaml` のうち `hashicorp/terraform` など、`tnnl` CLI の build / runtime / release artifact に入らないローカルツール pin だけの変更
- `.github/workflows/test.yml` または `.github/workflows/release.yml` のうち、Renovate による GitHub Actions / CI / release tool の patch/minor version pin 更新だけの変更
- `terraform/.terraform.lock.hcl` だけの変更
- Terraform provider / Terraform CLI / infra tool の更新で、Go binary の内容・GoReleaser build・release workflow に影響しないもの

release tag が不要な変更だけをマージした場合:

- release tag を作らない。
- `tnnl update` を実行しない。
- 最終報告に、release を作らなかった理由と local CLI を更新しなかった理由を書く。

- 全ての対象 Renovate PR を確認し、マージする PR とマージしない PR を判断し終えてから release に進む。
- マージ可能な PR を全て `squash` merge し終える。
- release tag 作成前に、open な Renovate PR を必ず再取得し、以下を確認する。
  - `mergeStateStatus: CLEAN` かつ `Test / test: SUCCESS` かつ他の許可条件を満たす PR が残っていない。
  - `mergeStateStatus: UNKNOWN` の PR は再取得して状態確定を待つ。待っても確定しない場合は、確認不能として未マージ理由に記録する。
  - 直前のマージで `main` が更新された後に、以前 `DIRTY` / `UNKNOWN` だった PR が `CLEAN` になっていない。
- マージ可能な PR が 1 件でも残っている場合は release tag を作らず、マージ手順に戻る。
- release tag は一連の Renovate automerge 作業の最後に 1 回だけ作る。依存 PR ごとに tag を分けてはいけない。
- `main` を最新化し、release tag が最新の `origin/main` を指すようにする。
- 既存 tag を確認し、最新の `vX.Y.Z` から次の patch version を作る。
  - 例: 最新 tag が `v0.6.2` なら次は `v0.6.3`。
  - Renovate automerge では major update をマージしないため、自動 release は原則 patch bump にする。
  - dependency update が minor でも、この repo の source code に feature 追加がない限り patch bump にする。
- `git tag -a <next-tag> -m "Release <next-tag>"` で annotated tag を作る。
- `git push origin <next-tag>` で tag を push し、`.github/workflows/release.yml` を起動する。
- GitHub Actions の `release` workflow が `SUCCESS` になるまで確認する。
- release workflow が failed / cancelled / timed out の場合は追加操作をせず、失敗した run URL と原因の要約を報告する。
- release workflow が `SUCCESS` になった後、ローカルの `tnnl` CLI を最新 release に更新する。
  - `tnnl update` を実行する。
  - 更新後に `tnnl version` を実行し、出力が作成した release tag と一致することを確認する。
  - `tnnl` が PATH にない、更新に失敗する、または version が作成 tag と一致しない場合は、追加 release tag を作らず、失敗内容を報告する。

release tag を作ってはいけない場合:

- マージした PR が 0 件
- マージした PR が `tnnl` CLI の配布バイナリや release pipeline に影響しない変更だけである
- この skill 実行中に既に release tag を作成または push している
- open な Renovate PR の最終再評価が終わっていない
- 許可条件を満たすマージ可能な Renovate PR が残っている
- `main` を最新化できない
- 最新 tag を判断できない
- 次に作る tag が既に存在する
- `git tag` または `git push origin <tag>` が失敗する
- GitHub Actions の状態を確認できない

## 報告

作業後に以下を日本語で報告する。

- マージした PR: PR 番号、title、merge method
- マージしなかったが対応が必要な PR: PR 番号、title、理由、未マージ理由コメントの URL
- 既に同等内容の未マージ理由コメントがある PR: comment skipped として扱い、既存コメント URL を報告
- マージしなかったが人間確認が必要な Renovate PR には、コメント投稿の有無に関係なく `renovate-needs-manual-review` を付ける。既に同等内容の comment がある場合も、重複コメントは投稿せず label は付ける。
- merge state の一時的な `UNKNOWN`、release tag push 後の同一 run 内追加マージ禁止、tooling の一時的な失敗など、次回 automation run で再評価すればよいだけの PR には `renovate-needs-manual-review` を付けない。既に付いている場合は外す。
- release: 作成した tag、release workflow の結果、run URL
- local CLI: `tnnl update` の結果、更新後の `tnnl version`
- 対象となる Renovate PR がなかった場合: 対象なしと報告
- check や review 状態が判断できなかった場合: マージせず、確認できなかった項目を報告

## PR コメント

- マージしなかった Renovate PR には、調査結果を PR 上に残す価値がある場合だけコメントする。
- コメントする場合は、確認した release notes / changelog、影響範囲、マージしなかった具体理由、人間に確認してほしい点を簡潔に含める。
- 既に同等内容の comment がある場合は、重複コメントを投稿せず comment skipped として扱い、既存コメント URL を報告する。
- マージしなかったが人間確認が必要な Renovate PR には、コメント投稿の有無に関係なく `renovate-needs-manual-review` を付ける。既に同等内容の comment がある場合も、重複コメントは投稿せず label は付ける。
- merge state の一時的な `UNKNOWN`、release tag push 後の同一 run 内追加マージ禁止、tooling の一時的な失敗など、次回 automation run で再評価すればよいだけの PR には `renovate-needs-manual-review` を付けない。既に付いている場合は外す。

## 禁止操作

- Renovate branch に commit や push をしない。
- PR を close しない。
- Renovate の rebase checkbox を操作しない。
- この skill に明記されていない条件の PR はマージしない。
- マージした PR がない場合は release tag を作らない。
- `vX.Y.Z` 形式ではない release tag を作らない。
