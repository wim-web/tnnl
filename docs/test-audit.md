# テスト価値監査・対応結果台帳

更新日: **2026-10-28（JST）**
基準commit: `396f7117ab39552e56f62180e2066786de119bca`を起点とする監査・修正 / Go `1.27.3 darwin/arm64`

**残っていたQ06〜Q13もすべて対応し、LV01〜LV13・Q01〜Q13の全件を完了した。** 最終の全体raceテストは393ケースPASS。今回追加した18種類の退行と、従来のM01〜M05をすべて検出した。

今回の強化中に、本番の不具合2件を再現して修正した。**go-builder等の通常ディレクトリをgo runと誤認する更新対象選択**と、**子孫が出力パイプを保持するとPlugin事前確認がキャンセル後も戻らない問題**である。

Goテストは初回170関数/383ケース → 前回整理後163/371 → **現行164関数/393ケース**。今回追加のTest関数は実path resolverの1件（T173）。他の7項目は既存テストとfixtureを強化した。

- [現行全件台帳](test-audit-ledger.md): 28ファイル・164関数、全ケース、対応状態、削除/統合の履歴。
- [全ケースCSV](test-audit-cases.csv): Go 393＋シェル9＋既存CI assertion 1＝403行。
- [検証証拠JSON](test-audit-evidence.json): 本番修正前の再現、修正後の検証、変異結果、初回失敗と再実行、source hash。

## 今回のQ06〜Q13対応

| ID | 対応・保証 | 主なテスト | 検証 |
| --- | --- | --- | --- |
| Q06 **完了** | 候補自身の実行pathを記録。候補scriptの$0からstageの親を観測し、current executableと同じ親directoryに配置されることを確認。TMPDIR不使用も維持。 | [TestUpdaterStagesCandidateBesideExecutable](../cmd/update/orchestration_test.go#L61) | 別directoryにstagingする変異を検出 |
| Q07 **完了** | 旧fileのbytes/mode/identityを保護。candidate open/copy失敗でcurrentをsnapshot比較対象へ追加。他人temp保護・自己temp cleanupも維持。 | [TestReplaceExecutableFailureCleansUpOnlyOwnTemp](../cmd/update/install_test.go#L272) | 内容破壊・mode変更・identity置換の3変異をopen/copy両caseで検出 |
| Q08 **完了** | 実resolverとPATH/symlinkを検証。子process8ケースと判定10ケースを追加。go-builder等をgo runと誤認する本番不具合も修正。 | [TestResolveTargetExecutablePath](../cmd/update/path_test.go#L14) | 旧実装の誤選択を再現して修正。PATH探索無効化・symlink解決除去の2変異を検出 |
| Q09 **完了** | nil境界を単独で到達。総崩れnil1caseを単一fieldのnil6caseへ置換。各々その前の条件をreadyに保つ。 | [TestEligibleTaskRequiresReadyTaskAndContainer](../internal/target/eligibility_test.go#L11) | 6つのnil読取を直接dereferenceへ変える変異を各caseで検出（panicを含む） |
| Q10 **完了** | 公開Preflightを実processへ移行。実PATH/argv/env、caller取消/期限、自身の3秒期限、子孫のpipe継承を検証。WaitDelay=200msで終了待ちの実不具合を修正。 | [TestPreflightDeadlineIsDiscoverable](../internal/session_manager/session_test.go#L273) | 修正前の終了待ちを再現。WaitDelay削除・自身の期限延長の2変異を検出 |
| Q11 **完了** | 出力失敗の2経路を確認。already-latest/updatedを分離し、更新後writer失敗でもcandidate内容・0755への置換が済んでいることを確認。 | [TestUpdaterReturnsOutputError](../cmd/update/orchestration_test.go#L193) | statusを置換前へ移す変異とwriter errorを無視する変異を検出 |
| Q12 **完了** | 有効IDでcleanupまで通過。元ctx/Value/Cause/Invocationの保証を維持し、正しいsession IDでcleanupが1回呼ばれるfixtureへ変更。 | [TestRemoteSessionPassesOriginalContextToPlugin](../pkg/command/session_test.go#L198) | 元ctxをWithoutCancelへ変える変異を検出 |
| Q13 **完了** | 独立watchdogで局所FAIL。期限欠落時に1秒でFAIL。ctxとwatchdogが両方readyならctx.Errを先に確認して偽陽性を防ぐ。 | [TestRemoteSessionCleanupTimeoutIsDiscoverable](../pkg/command/session_test.go#L161) | cleanup期限削除時に全体timeoutではなく独立watchdogでFAIL |

## 本番コードの修正

| 対象 | 修正前 | 修正後 |
| --- | --- | --- |
| [更新対象選択](../cmd/update/update.go#L435) | pathに`/go-build`が含まれるだけで、`go-builder`等もPATH上の別tnnlへ切り替わる | directory segmentが`go-build`または`go-build`＋数字の場合に限定。通常インストール、go run相当、PATH欠落、symlinkを実子processで確認 |
| [Plugin事前確認](../internal/session_manager/session.go#L86) | caller取消/期限後に子孫がstdout/stderrを保持すると、CombinedOutputの終了待ちが続く | 実exec.Cmdに`WaitDelay=200ms`。caller取消/期限とPreflight自身の3秒期限を実processで確認。3秒制限と終了I/O待ちの上限は別の境界 |

## 最終検証

| 検証 | 結果 |
| --- | --- |
| 全Goテスト | `go test -race -count=1 -timeout=3m -coverprofile=… -json ./...`：**393 leaf PASS、fail/skip 0**（444 test node） |
| go vet / gofmt / diff check | **PASS** |
| シェルverifier | **9ケースPASS** |
| 従来のM01〜M05 | 最終ソースの隔離コピーで再実行し、**5/5を意図したテストで検出** |
| Q06〜Q13の追加退行 | 対象テスト＋隔離overlayで**18/18を検出**。網羅的mutation scoreではない |
| statement coverage | 85.4%（前回84.9%、初回84.8%）。assertionの強さや全本番統合を示す指標ではない |

初回の全体実行とM01実行には、既存の候補version正常系で5秒timeoutが1件ずつ発生した。会話処理の中断・並行実行と重なったが、原因は断定していない。ソースを変更せず逐次再実行し、全体raceは全PASS、M01は期待した適格性テストだけのFAILになった。初回の失敗もJSONへ保存し、成功結果で上書きして隠していない。

実AWS/実Session Manager Pluginとの接続、Linux上の再実行、GitHub Actions自体の実行は対象外。CIの開発版version assertionは維持しており、初回監査でのローカル再現結果と今回の実linker版テストを区別している。

## 低価値部分・統合候補の対応

全13組を対応済みとした。固有の保護を持つ小さなテストは残し、ケース・assertion単位で整理している。LV11/LV12は今回の保証範囲の判断も記録した。

| ID | 対応結果 | 残した保証・判断 | 主な移植先 |
| --- | --- | --- | --- |
| LV01 **完了** | 重複2ケース削除 | container name/runtime IDの個別metadataケースを削除。空候補防御の代表とtarget適格性試験を維持。 | [TestResolveTargetRejectsEmptyResourcesBeforeChooser](../internal/view/resolve_test.go#L190) |
| LV02 **完了** | 本番形式へ置換 | 空task成功fakeをWait errorのcause/後続停止検証へ置換。不適格containerの契約違反防御を代表1件維持。 | [TestResolveTargetRejectsEmptyResourcesBeforeChooser](../internal/view/resolve_test.go#L190) |
| LV03 **完了** | 2関数を1 matrixへ統合 | 通常errorをcluster/task/containerの3段階で検証し、quit+error優先は代表1件。zero result・cause・後続停止を維持。 | [TestResolveTargetPropagatesChooserErrors](../internal/view/resolve_test.go#L378) |
| LV04 **完了** | 上位へ移植・helper重複削除 | label完全一致とfull ARN valueを上位選択試験へ移植。不正ARNは上位へ集約、blank group fallbackを独立維持。 | [TestResolveTargetSelectsExactReadyTaskARN](../internal/view/resolve_test.go#L56) |
| LV05 **完了** | 自己代入assertion削除 | item.Valueを自分のfixtureと比較するassertionのみ削除。FilterValueのlabel契約を維持し改名。 | [TestItemFilterValueUsesLabel](../internal/listview/simple_test.go#L11) |
| LV06 **完了** | factory3ケースを代表1件＋実CLI3親へ移行 | mainの実登録から親別help・既定出力名・JSON全field/defaultを確認。単にfactory確認を減らすだけにしない。 | [TestCLIBinaryContracts](../main_test.go#L16) |
| LV07 **完了** | help2組を統合・2関数削除 | flag自身のUsage assertionを描画help試験へ移植し、重複substringを削除。 | [TestPortforwardHelpDocumentsInputAndAutomaticLocalPort](../cmd/portforward/portforward_test.go#L162) |
| LV08 **完了** | 重複validator1関数削除 | abc/0の複合エラーはresolver試験を維持。Execの前後空白を直接渡すvalidator試験は維持。 | [TestResolvePortForwardReturnsZeroValueOnValidationError](../internal/input/resolve_test.go#L138) |
| LV09 **完了** | 2関数・4ケースを上位へ統合 | 正規化は候補versionの3実行ケース、asset URLはupdaterの実HTTP request pathで保証。 | [TestVerifyCandidateVersion](../cmd/update/install_test.go#L40) |
| LV10 **完了** | 重複assertionのみ削除 | Silence boolは実child error、出力相互比較は両入口の固定期待値に集約。-v定義・不要flag・writer errorは維持。 | [TestExecuteContextReturnsChildErrorSilently](../cmd/root_test.go#L65) |
| LV11 **完了** | 非標準値4ケース削除 | 通常release/module/pseudo版・dev・fallbackの15ケースを保証範囲とし、vv/V/linker(devel)の実装上の癖を固定しない。受容する本番実装は変更なし。 | [TestResolveVersion](../internal/buildinfo/version_test.go#L20) |
| LV12 **完了** | fake入力破壊1関数削除 | 現在のSDK経路をテスト契約とし、任意clientの入力append破壊耐性は独立保証しない。100/100/1分割と本番のslice copyは維持。 | [TestResolverEligibleTasksChunksDescribeRequestsAndPreservesResponseOrder](../internal/target/resolver_test.go#L372) |
| LV13 **完了** | 内部error木assertion削除 | Unwrap型・子error数だけ削除し、全failure文言とpartial返却禁止を維持。 | [TestResolverEligibleTasksReturnsJoinedDescribeFailures](../internal/target/resolver_test.go#L424) |

## 5種類の見逃しへの対応と変異結果

各変異は隔離コピーに1種類ずつ適用し、毎回 `go test -count=1 -json ./...` を実行した。元に戻してから次を適用。本番コードを作業ツリーで変える必要はなく、下記の誤動作は監査のために意図的に作ったもの。

| ID | 変異 | 強化した保証 | 検出結果 |
| --- | --- | --- | --- |
| Q01/M01 **完了** | 適格性filterを無効化 | 6 task中ready2件のみを独立した完全値の期待値で照合。 [TestResolverEligibleTasksFiltersUnreadyDescribeResults](../internal/target/resolver_test.go#L283) | FAIL / 期待した1件で検出 |
| Q02/M02 **完了** | force時のO_TRUNCを除去 | 両force試験を長い旧JSON→短いJSONに変更し、期待bytes/lengthとstrict decodeを確認。 [TestMakeInputFileRefusesExistingFileUnlessForced](../internal/input/func_test.go#L47) | FAIL / 期待した2件で検出 |
| Q03/M03 **完了** | 3CLIのcontextをWithoutCancelへ置換 | exec/portforward/remoteportforwardの3試験でValue・deadline・caller取消を確認。 [TestExecCommandPassesExecuteContextToRunner](../cmd/exec/exec_test.go#L62) | FAIL / 期待した3件で検出 |
| Q04/M04 **完了** | 2つのSSM document定数を誤値へ変更 | 2 document名を本番定数から独立したliteralで照合。 [TestPortForwardHandlerSelectsSecondReplicaAndAllocatesLocalPort](../internal/handler/portforward_test.go#L48) | FAIL / 期待した2件で検出 |
| Q05/M05 **完了** | cmd.Versionをdevに固定 | 既知linker版の実バイナリをversion/--version/-vで起動。旧dev同士の比較を置換。 [TestCLIBinaryContracts](../main_test.go#L16) | FAIL / 期待した3件で検出 |


## 保証範囲と今後の更新

改善IDを付けた全項目は完了した。実AWS接続、非空UIのProgram全体、productionDependenciesの全組立て、非ゼロwaitのhandler→本番Sleepなど、ID外で列挙した統合境界は今回の完了範囲に含めない。

context/Cause・constructor cleanup・実socket・独立JSON契約など固有の保護を持つ既存テストは維持。公開Preflight、実PATH/symlink resolver、実template登録は今回までに本番の呼出経路を通す保証へ強化済み。

テストを変更した際は対応ID、現行sourceリンク、CSVケース、source hash、対応する退行検出結果を同時更新する。

## マージ前検証

最新main `0bc466fed663dbeca2e953c0bebcd75cb1f95443` の依存関係を取り込み、race付き393ケース（fail/skip 0）、go vet、gofmtを再確認した。シェルverifierの9ケースもPASS。監査当初の基準commitと今回の検証ベースを証拠JSONで区別している。
