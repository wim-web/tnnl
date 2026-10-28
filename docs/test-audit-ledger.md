# 全テスト監査・対応台帳

更新日: 2026-10-28（JST）。基準commit: `396f7117ab39552e56f62180e2066786de119bca`を起点とする監査・修正。

[対応概要](test-audit.md) / [現行全ケースCSV](test-audit-cases.csv) / [全検証証拠](test-audit-evidence.json)

**LV01〜LV13・Q01〜Q13はすべて対応済み。** 現行Go 28ファイル・164関数・393 leaf。初回監査は26ファイル・170関数・383 leaf、前回対応後は27ファイル・163関数・371 leaf。旧IDを維持し、今回追加の実path resolver試験はT173。削除9関数・改名3件の履歴も保持した。

全393 leafはrace付きでPASS、skip 0。途中の初回全体実行には既存候補version確認の5秒timeoutが1件あった。原因は断定せず、逐次再実行の全PASSとともに対応概要・証拠へ記録している。

ソースリンク/CSV lineは現行位置。引継ぎ根拠の行番号は差分に沿って更新し、消えた位置は「初回監査位置」と明記した。通常テストが通るだけでなく、追加18退行と従来5変異をそれぞれ検出した。nil安全性の6変異は対象caseでのpanicによる検出を含む。

## 対応IDと保護の移植先

| ID | 状態・対応 | 残した保証・判断 | 主なテスト |
| --- | --- | --- | --- |
| LV01 | 完了: 重複2ケース削除 | container name/runtime IDの個別metadataケースを削除。空候補防御の代表とtarget適格性試験を維持。 | [TestResolveTargetRejectsEmptyResourcesBeforeChooser](../internal/view/resolve_test.go#L190) |
| LV02 | 完了: 本番形式へ置換 | 空task成功fakeをWait errorのcause/後続停止検証へ置換。不適格containerの契約違反防御を代表1件維持。 | [TestResolveTargetRejectsEmptyResourcesBeforeChooser](../internal/view/resolve_test.go#L190) |
| LV03 | 完了: 2関数を1 matrixへ統合 | 通常errorをcluster/task/containerの3段階で検証し、quit+error優先は代表1件。zero result・cause・後続停止を維持。 | [TestResolveTargetPropagatesChooserErrors](../internal/view/resolve_test.go#L378) |
| LV04 | 完了: 上位へ移植・helper重複削除 | label完全一致とfull ARN valueを上位選択試験へ移植。不正ARNは上位へ集約、blank group fallbackを独立維持。 | [TestResolveTargetSelectsExactReadyTaskARN](../internal/view/resolve_test.go#L56) |
| LV05 | 完了: 自己代入assertion削除 | item.Valueを自分のfixtureと比較するassertionのみ削除。FilterValueのlabel契約を維持し改名。 | [TestItemFilterValueUsesLabel](../internal/listview/simple_test.go#L11) |
| LV06 | 完了: factory3ケースを代表1件＋実CLI3親へ移行 | mainの実登録から親別help・既定出力名・JSON全field/defaultを確認。単にfactory確認を減らすだけにしない。 | [TestCLIBinaryContracts](../main_test.go#L16) |
| LV07 | 完了: help2組を統合・2関数削除 | flag自身のUsage assertionを描画help試験へ移植し、重複substringを削除。 | [TestPortforwardHelpDocumentsInputAndAutomaticLocalPort](../cmd/portforward/portforward_test.go#L162) |
| LV08 | 完了: 重複validator1関数削除 | abc/0の複合エラーはresolver試験を維持。Execの前後空白を直接渡すvalidator試験は維持。 | [TestResolvePortForwardReturnsZeroValueOnValidationError](../internal/input/resolve_test.go#L138) |
| LV09 | 完了: 2関数・4ケースを上位へ統合 | 正規化は候補versionの3実行ケース、asset URLはupdaterの実HTTP request pathで保証。 | [TestVerifyCandidateVersion](../cmd/update/install_test.go#L40) |
| LV10 | 完了: 重複assertionのみ削除 | Silence boolは実child error、出力相互比較は両入口の固定期待値に集約。-v定義・不要flag・writer errorは維持。 | [TestExecuteContextReturnsChildErrorSilently](../cmd/root_test.go#L65) |
| LV11 | 完了: 非標準値4ケース削除 | 通常release/module/pseudo版・dev・fallbackの15ケースを保証範囲とし、vv/V/linker(devel)の実装上の癖を固定しない。受容する本番実装は変更なし。 | [TestResolveVersion](../internal/buildinfo/version_test.go#L20) |
| LV12 | 完了: fake入力破壊1関数削除 | 現在のSDK経路をテスト契約とし、任意clientの入力append破壊耐性は独立保証しない。100/100/1分割と本番のslice copyは維持。 | [TestResolverEligibleTasksChunksDescribeRequestsAndPreservesResponseOrder](../internal/target/resolver_test.go#L372) |
| LV13 | 完了: 内部error木assertion削除 | Unwrap型・子error数だけ削除し、全failure文言とpartial返却禁止を維持。 | [TestResolverEligibleTasksReturnsJoinedDescribeFailures](../internal/target/resolver_test.go#L424) |
| Q01/M01 | 完了・変異検出 | 6 task中ready2件のみを独立した完全値の期待値で照合。 | [TestResolverEligibleTasksFiltersUnreadyDescribeResults](../internal/target/resolver_test.go#L283) |
| Q02/M02 | 完了・変異検出 | 両force試験を長い旧JSON→短いJSONに変更し、期待bytes/lengthとstrict decodeを確認。 | [TestMakeInputFileRefusesExistingFileUnlessForced](../internal/input/func_test.go#L47) |
| Q03/M03 | 完了・変異検出 | exec/portforward/remoteportforwardの3試験でValue・deadline・caller取消を確認。 | [TestExecCommandPassesExecuteContextToRunner](../cmd/exec/exec_test.go#L62) |
| Q04/M04 | 完了・変異検出 | 2 document名を本番定数から独立したliteralで照合。 | [TestPortForwardHandlerSelectsSecondReplicaAndAllocatesLocalPort](../internal/handler/portforward_test.go#L48) |
| Q05/M05 | 完了・変異検出 | 既知linker版の実バイナリをversion/--version/-vで起動。旧dev同士の比較を置換。 | [TestCLIBinaryContracts](../main_test.go#L16) |
| Q06 | 完了: 候補自身の実行pathを記録 | 候補scriptの$0からstageの親を観測し、current executableと同じ親directoryに配置されることを確認。TMPDIR不使用も維持。 | [TestUpdaterStagesCandidateBesideExecutable](../cmd/update/orchestration_test.go#L61) |
| Q07 | 完了: 旧fileのbytes/mode/identityを保護 | candidate open/copy失敗でcurrentをsnapshot比較対象へ追加。他人temp保護・自己temp cleanupも維持。 | [TestReplaceExecutableFailureCleansUpOnlyOwnTemp](../cmd/update/install_test.go#L272) |
| Q08 | 完了: 実resolverとPATH/symlinkを検証 | 子process8ケースと判定10ケースを追加。go-builder等をgo runと誤認する本番不具合も修正。 | [TestResolveTargetExecutablePath](../cmd/update/path_test.go#L14) |
| Q09 | 完了: nil境界を単独で到達 | 総崩れnil1caseを単一fieldのnil6caseへ置換。各々その前の条件をreadyに保つ。 | [TestEligibleTaskRequiresReadyTaskAndContainer](../internal/target/eligibility_test.go#L11) |
| Q10 | 完了: 公開Preflightを実processへ移行 | 実PATH/argv/env、caller取消/期限、自身の3秒期限、子孫のpipe継承を検証。WaitDelay=200msで終了待ちの実不具合を修正。 | [TestPreflightDeadlineIsDiscoverable](../internal/session_manager/session_test.go#L273) |
| Q11 | 完了: 出力失敗の2経路を確認 | already-latest/updatedを分離し、更新後writer失敗でもcandidate内容・0755への置換が済んでいることを確認。 | [TestUpdaterReturnsOutputError](../cmd/update/orchestration_test.go#L193) |
| Q12 | 完了: 有効IDでcleanupまで通過 | 元ctx/Value/Cause/Invocationの保証を維持し、正しいsession IDでcleanupが1回呼ばれるfixtureへ変更。 | [TestRemoteSessionPassesOriginalContextToPlugin](../pkg/command/session_test.go#L198) |
| Q13 | 完了: 独立watchdogで局所FAIL | 期限欠落時に1秒でFAIL。ctxとwatchdogが両方readyならctx.Errを先に確認して偽陽性を防ぐ。 | [TestRemoteSessionCleanupTimeoutIsDiscoverable](../pkg/command/session_test.go#L161) |

## ファイル別網羅性

| ファイル | Test関数 | leafケース |
| --- | ---: | ---: |
| [cmd/exec/exec_test.go](../cmd/exec/exec_test.go) | 5 | 5 |
| [cmd/inputfile/make_test.go](../cmd/inputfile/make_test.go) | 5 | 5 |
| [cmd/portforward/portforward_test.go](../cmd/portforward/portforward_test.go) | 5 | 9 |
| [cmd/remoteportforward/remoteportforward_test.go](../cmd/remoteportforward/remoteportforward_test.go) | 5 | 10 |
| [cmd/root_test.go](../cmd/root_test.go) | 6 | 8 |
| [cmd/update/install_test.go](../cmd/update/install_test.go) | 4 | 17 |
| [cmd/update/orchestration_test.go](../cmd/update/orchestration_test.go) | 13 | 14 |
| [cmd/update/path_test.go](../cmd/update/path_test.go) | 1 | 8 |
| [cmd/update/release_test.go](../cmd/update/release_test.go) | 2 | 12 |
| [cmd/update/update_test.go](../cmd/update/update_test.go) | 14 | 33 |
| [internal/buildinfo/version_test.go](../internal/buildinfo/version_test.go) | 2 | 16 |
| [internal/handler/exec_test.go](../internal/handler/exec_test.go) | 5 | 6 |
| [internal/handler/portforward_test.go](../internal/handler/portforward_test.go) | 7 | 10 |
| [internal/input/func_test.go](../internal/input/func_test.go) | 3 | 4 |
| [internal/input/resolve_test.go](../internal/input/resolve_test.go) | 12 | 12 |
| [internal/input/validate_test.go](../internal/input/validate_test.go) | 4 | 20 |
| [internal/listview/simple_test.go](../internal/listview/simple_test.go) | 3 | 3 |
| [internal/session_manager/session_test.go](../internal/session_manager/session_test.go) | 14 | 26 |
| [internal/target/eligibility_test.go](../internal/target/eligibility_test.go) | 2 | 19 |
| [internal/target/identifier_test.go](../internal/target/identifier_test.go) | 2 | 48 |
| [internal/target/resolver_test.go](../internal/target/resolver_test.go) | 11 | 18 |
| [internal/target/wait_test.go](../internal/target/wait_test.go) | 1 | 13 |
| [internal/view/resolve_test.go](../internal/view/resolve_test.go) | 11 | 22 |
| [main_test.go](../main_test.go) | 1 | 6 |
| [pkg/command/exec_test.go](../pkg/command/exec_test.go) | 7 | 26 |
| [pkg/command/portforward_test.go](../pkg/command/portforward_test.go) | 8 | 12 |
| [pkg/command/session_test.go](../pkg/command/session_test.go) | 7 | 7 |
| [pkg/port/available_test.go](../pkg/port/available_test.go) | 4 | 4 |

## 現行Goテスト全件

### T001 — TestExecCommandExplicitFlagsOverrideFile

- ソース: [cmd/exec/exec_test.go:17](../cmd/exec/exec_test.go#L17)
- 判定: **維持** / 状態: 維持
- 守る契約: --commandと明示--wait 0がfile値を上書き
- 本番経路: mainのblank import→RootCmd登録→newExecCommand.RunE→input.ResolveExec→handler runner（テストはrunnerをfake化）
- 重複・対応先: TestResolveExecPrecedence（下位value mergeのみ）
- 根拠・限界: 実Cobra parsingとFlags().Changedを通る。pointerを直接与えるresolverテストにないCLIアダプター保証。
- 対応: 維持。未指定/commandのみ/waitのみでfileが失われないケースは追加候補。
- ケース別注意: サブテストなし
- 実行: 1 leaf / 全件PASS。

### T002 — TestExecCommandInvalidInputDoesNotInvokeRunner

- ソース: [cmd/exec/exec_test.go:39](../cmd/exec/exec_test.go#L39)
- 判定: **維持** / 状態: 維持
- 守る契約: fileの空白command/負waitを拒否しrunner呼出0回
- 本番経路: mainのblank import→RootCmd登録→newExecCommand.RunE→input.ResolveExec→handler runner（テストはrunnerをfake化）
- 重複・対応先: TestValidateExecRejectsNegativeWaitAndBlankCommand（validationのみ）
- 根拠・限界: 同じエラーだが本テストのみAWSを開始するrunnerを呼ばない副作用境界を保証。
- 対応: 維持
- ケース別注意: サブテストなし
- 実行: 1 leaf / 全件PASS。

### T003 — TestExecCommandPassesExecuteContextToRunner

- ソース: [cmd/exec/exec_test.go:62](../cmd/exec/exec_test.go#L62)
- 判定: **維持** / 状態: 対応済み
- 守る契約: Cobra ExecuteContextで受けたcontextのValue、deadline、caller取消をrunnerまで保つ
- 本番経路: newExecCommand → Cobra ExecuteContext → runner
- 重複・対応先: なし
- 根拠・限界: deadline付きcaller contextをrunnerから取得し、deadline一致とcancel後のcontext.Canceledを確認。WithoutCancelはdeadline確認で失敗。
- 対応: Q03/M03強化済み
- ケース別注意: サブテストなし
- 実行: 1 leaf / 全件PASS。

### T004 — TestExecCommandInputFileHelpNamesParent

- ソース: [cmd/exec/exec_test.go:93](../cmd/exec/exec_test.go#L93)
- 判定: **維持** / 状態: 維持
- 守る契約: Short非空、input-file flag存在、flag説明に正しい生成コマンド
- 本番経路: mainのblank import→RootCmd登録→newExecCommand.RunE→input.ResolveExec→handler runner（テストはrunnerをfake化）
- 重複・対応先: なし
- 根拠・限界: ユーザーが読むflag固有のusageを保護。別helpテストはwait/優先順位のみなので同じassertではない。
- 対応: 維持。実MakeInputFileCmd登録を保証するテストとは分ける。
- ケース別注意: サブテストなし
- 実行: 1 leaf / 全件PASS。

### T005 — TestExecHelpDocumentsWaitAndInputPrecedence

- ソース: [cmd/exec/exec_test.go:107](../cmd/exec/exec_test.go#L107)
- 判定: **維持** / 状態: 維持
- 守る契約: 描画されたhelpがwait=0/positive wait/入力優先順位を案内
- 本番経路: mainのblank import→RootCmd登録→newExecCommand.RunE→input.ResolveExec→handler runner（テストはrunnerをfake化）
- 重複・対応先: なし
- 根拠・限界: 重要な待機タイミングと優先順位の利用者向け契約。ただし英語文字列変更には壊れやすい。
- 対応: 維持
- ケース別注意: サブテストなし
- 実行: 1 leaf / 全件PASS。

### T006 — TestNewCreatesTemplateAtOutputAndReportsPath

- ソース: [cmd/inputfile/make_test.go:17](../cmd/inputfile/make_test.go#L17)
- 判定: **維持** / 状態: 維持
- 守る契約: --output指定先へ渡したskeletonを保存しstdoutへ作成path報告
- 本番経路: main→各親のMakeInputFileCmd（本番）→inputfile.New.RunE→MakeInputFile
- 重複・対応先: なし
- 根拠・限界: 実ファイルとCobra/out writerの接続を確認。skeleton/defaultPathはテスト独自指定で、本番の各親の引数までは確認しない。
- 対応: 維持
- ケース別注意: サブテストなし
- 実行: 1 leaf / 全件PASS。

### T007 — TestNewUsesParentSpecificDescription

- ソース: [cmd/inputfile/make_test.go:46](../cmd/inputfile/make_test.go#L46)
- 判定: **維持** / 状態: 対応済み
- 守る契約: factoryのUseと親名を含むShort説明
- 本番経路: New(parent, output, skeleton) → cobra.Command metadata
- 重複・対応先: 各CommandInputFileHelpNamesParent（利用案内は別箇所）
- 根拠・限界: 同一処理の3親名caseをexec1件へ縮約。実3登録はmain binary試験へ移す。
- 対応: LV06対応完了。factory代表1件を維持し、親別の実登録/説明/生成schemaはTestCLIBinaryContractsが保証。
- ケース別注意: exec代表1件。実CLIの3親はT171で検証。
- 実行: 1 leaf / 全件PASS。

### T008 — TestNewRefusesExistingFileAndPreservesContents

- ソース: [cmd/inputfile/make_test.go:56](../cmd/inputfile/make_test.go#L56)
- 判定: **維持** / 状態: 維持
- 守る契約: force未指定でfs.ErrExist、既存bytes不変、成功stdoutなし
- 本番経路: main→各親のMakeInputFileCmd（本番）→inputfile.New.RunE→MakeInputFile
- 重複・対応先: TestMakeInputFileRefusesExistingFileUnlessForced
- 根拠・限界: 下位既存保護に加えCobra default force=falseの配線と成功表示を抑制する追加価値がある。
- 対応: 維持
- ケース別注意: サブテストなし
- 実行: 1 leaf / 全件PASS。

### T009 — TestNewForceReplacesExistingFile

- ソース: [cmd/inputfile/make_test.go:85](../cmd/inputfile/make_test.go#L85)
- 判定: **維持** / 状態: 対応済み
- 守る契約: --forceで長い旧fileを短いportforwardテンプレートへ完全置換し、生成pathを報告
- 本番経路: New → Cobra Execute(--force) → input.MakeInputFile
- 重複・対応先: TestMakeInputFileRefusesExistingFileUnlessForced
- 根拠・限界: 旧fileは有効な長いJSONで新99 bytesより長いことを明示。独立literal期待bytes/lengthとReadInputFileのstrict decode、期待input値、stdoutを確認。
- 対応: Q02/M02強化済み
- ケース別注意: サブテストなし
- 実行: 1 leaf / 全件PASS。

### T010 — TestNewReturnsOutputWriteError

- ソース: [cmd/inputfile/make_test.go:124](../cmd/inputfile/make_test.go#L124)
- 判定: **維持** / 状態: 維持
- 守る契約: 生成成功後のstdout障害をerrors.Is可能なまま返す
- 本番経路: main→各親のMakeInputFileCmd（本番）→inputfile.New.RunE→MakeInputFile
- 重複・対応先: なし
- 根拠・限界: writer差替えは実cmd.OutOrStdoutへの出力経路で、成功扱いしてしまうエラー無視を防ぐ。
- 対応: 維持
- ケース別注意: サブテストなし
- 実行: 1 leaf / 全件PASS。

### T011 — TestPortforwardCommandOverridesOnlyExplicitFlags

- ソース: [cmd/portforward/portforward_test.go:17](../cmd/portforward/portforward_test.go#L17)
- 判定: **維持** / 状態: 維持
- 守る契約: 明示したフラグだけfile値へ上書きし未指定値を維持、runner1回
- 本番経路: mainのblank import→RootCmd登録→newPortforwardCommand.RunE→input.ResolvePortForward→handler runner（テストはrunnerをfake化）
- 重複・対応先: なし
- 根拠・限界: Flags().Changedと実フラグparseをテーブルで区別。resolverへのpointer直渡しでは発見できない配線不良を防ぐ。
- 対応: 維持
- ケース別注意: no overrides: file全値; target only: targetだけ変更; local only: localだけ変更
- 実行: 3 leaf / 全件PASS。

| 実行サブケース | 個別判定 |
| --- | --- |
| `local_only` | 維持 |
| `no_overrides` | 維持 |
| `target_only` | 維持 |

### T012 — TestPortforwardCommandPreservesExplicitEmptyLocalPort

- ソース: [cmd/portforward/portforward_test.go:82](../cmd/portforward/portforward_test.go#L82)
- 判定: **維持** / 状態: 維持
- 守る契約: 明示--local-port ""はfile local 8080を空へ戻しtarget80を保持
- 本番経路: mainのblank import→RootCmd登録→newPortforwardCommand.RunE→input.ResolvePortForward→handler runner（テストはrunnerをfake化）
- 重複・対応先: TestResolvePortForwardAppliesExplicitOverridesAndNormalizes（pointer/空白値）
- 根拠・限界: CobraでChanged=trueの空文字を維持する独自境界。
- 対応: 維持
- ケース別注意: サブテストなし
- 実行: 1 leaf / 全件PASS。

### T013 — TestPortforwardCommandPassesExecuteContextToRunner

- ソース: [cmd/portforward/portforward_test.go:99](../cmd/portforward/portforward_test.go#L99)
- 判定: **維持** / 状態: 対応済み
- 守る契約: Cobra ExecuteContextで受けたcontextのValue、deadline、caller取消をrunnerまで保つ
- 本番経路: newPortforwardCommand → Cobra ExecuteContext → runner
- 重複・対応先: なし
- 根拠・限界: deadline付きcaller contextをrunnerから取得し、deadline一致とcancel後のcontext.Canceledを確認。WithoutCancelはdeadline確認で失敗。
- 対応: Q03/M03強化済み
- ケース別注意: サブテストなし
- 実行: 1 leaf / 全件PASS。

### T014 — TestPortforwardCommandInvalidExplicitFlagDoesNotInvokeRunner

- ソース: [cmd/portforward/portforward_test.go:130](../cmd/portforward/portforward_test.go#L130)
- 判定: **維持** / 状態: 維持
- 守る契約: fileの正常値を明示不正値で上書きした後にvalidationを行いrunner0回
- 本番経路: mainのblank import→RootCmd登録→newPortforwardCommand.RunE→input.ResolvePortForward→handler runner（テストはrunnerをfake化）
- 重複・対応先: なし
- 根拠・限界: 空値/不正値を都合よく未指定扱いしfileへ戻すバグを防ぐ。
- 対応: 維持
- ケース別注意: empty target: required; invalid target: decimal; invalid local: range。全件runner0回。
- 実行: 3 leaf / 全件PASS。

| 実行サブケース | 個別判定 |
| --- | --- |
| `empty_target` | 維持 |
| `invalid_local` | 維持 |
| `invalid_target` | 維持 |

### T016 — TestPortforwardHelpDocumentsInputAndAutomaticLocalPort

- ソース: [cmd/portforward/portforward_test.go:162](../cmd/portforward/portforward_test.go#L162)
- 判定: **維持** / 状態: 対応済み
- 守る契約: input-file flag自身の親名説明、描画helpの自動local port/優先順位案内
- 本番経路: newPortforwardCommand → flag.Usage / Cobra Help
- 重複・対応先: 旧InputFileHelpNamesParentのflag Usage検査を本関数へ統合済み
- 根拠・限界: 旧InputFileHelpNamesParentを統合し固有flag.Usageを維持。描画help側で同じgenerator文字列を重ねて比較する箇所のみ削除。
- 対応: LV07統合済み
- ケース別注意: サブテストなし
- 実行: 1 leaf / 全件PASS。

### T017 — TestRemotePortforwardCommandOverridesOnlyExplicitFlags

- ソース: [cmd/remoteportforward/remoteportforward_test.go:17](../cmd/remoteportforward/remoteportforward_test.go#L17)
- 判定: **維持** / 状態: 維持
- 守る契約: 明示したフラグだけfile値へ上書きし未指定値を維持、runner1回
- 本番経路: mainのblank import→RootCmd登録→newRemotePortforwardCommand.RunE→input.ResolveRemotePortForward→handler runner（テストはrunnerをfake化）
- 重複・対応先: なし
- 根拠・限界: Flags().Changedと実フラグparseをテーブルで区別。resolverへのpointer直渡しでは発見できない配線不良を防ぐ。
- 対応: 維持
- ケース別注意: no overrides: file全値; remote only: remoteだけ変更; local only and empty: localだけ空へ; host only: hostだけ変更
- 実行: 4 leaf / 全件PASS。

| 実行サブケース | 個別判定 |
| --- | --- |
| `host_only` | 維持 |
| `local_only_and_empty` | 維持 |
| `no_overrides` | 維持 |
| `remote_only` | 維持 |

### T018 — TestRemotePortforwardCommandAllExplicitFlagsOverrideFile

- ソース: [cmd/remoteportforward/remoteportforward_test.go:96](../cmd/remoteportforward/remoteportforward_test.go#L96)
- 判定: **維持** / 状態: 維持
- 守る契約: 3つの明示flagsを同時にfileへ上書き
- 本番経路: mainのblank import→RootCmd登録→newRemotePortforwardCommand.RunE→input.ResolveRemotePortForward→handler runner（テストはrunnerをfake化）
- 重複・対応先: TestRemotePortforwardCommandOverridesOnlyExplicitFlags; TestResolveRemotePortForwardAppliesExplicitOverridesAndNormalizes
- 根拠・限界: 単一flagケースに重なるがifをelse-ifへ誤変更するとこの同時指定だけ失敗するので削除しない。
- 対応: 維持
- ケース別注意: サブテストなし
- 実行: 1 leaf / 全件PASS。

### T019 — TestRemotePortforwardCommandPassesExecuteContextToRunner

- ソース: [cmd/remoteportforward/remoteportforward_test.go:127](../cmd/remoteportforward/remoteportforward_test.go#L127)
- 判定: **維持** / 状態: 対応済み
- 守る契約: Cobra ExecuteContextで受けたcontextのValue、deadline、caller取消をrunnerまで保つ
- 本番経路: newRemotePortforwardCommand → Cobra ExecuteContext → runner
- 重複・対応先: なし
- 根拠・限界: deadline付きcaller contextをrunnerから取得し、deadline一致とcancel後のcontext.Canceledを確認。WithoutCancelはdeadline確認で失敗。
- 対応: Q03/M03強化済み
- ケース別注意: サブテストなし
- 実行: 1 leaf / 全件PASS。

### T020 — TestRemotePortforwardCommandInvalidRequiredValueDoesNotInvokeRunner

- ソース: [cmd/remoteportforward/remoteportforward_test.go:158](../cmd/remoteportforward/remoteportforward_test.go#L158)
- 判定: **維持** / 状態: 維持
- 守る契約: fileの正常値に対する明示empty remote/不正remote/empty hostを拒否しrunner0回
- 本番経路: mainのblank import→RootCmd登録→newRemotePortforwardCommand.RunE→input.ResolveRemotePortForward→handler runner（テストはrunnerをfake化）
- 重複・対応先: なし
- 根拠・限界: required fieldのChanged判定と解決後validationを実際に接続。
- 対応: 維持
- ケース別注意: empty remote; invalid remote; empty host。local invalidは本CLIでは未試験。
- 実行: 3 leaf / 全件PASS。

| 実行サブケース | 個別判定 |
| --- | --- |
| `empty_host` | 維持 |
| `empty_remote` | 維持 |
| `invalid_remote` | 維持 |

### T022 — TestRemotePortforwardHelpDocumentsRemoteHostAndAutomaticLocalPort

- ソース: [cmd/remoteportforward/remoteportforward_test.go:194](../cmd/remoteportforward/remoteportforward_test.go#L194)
- 判定: **維持** / 状態: 対応済み
- 守る契約: input-file flag自身の親名説明、描画helpのremote host/自動local port/優先順位案内
- 本番経路: newRemotePortforwardCommand → flag.Usage / Cobra Help
- 重複・対応先: 旧InputFileHelpNamesParentのflag Usage検査を本関数へ統合済み
- 根拠・限界: 旧InputFileHelpNamesParentを統合し固有flag.Usageを維持。描画help側で同じgenerator文字列を重ねて比較する箇所のみ削除。
- 対応: LV07統合済み
- ケース別注意: サブテストなし
- 実行: 1 leaf / 全件PASS。

### T023 — TestExecuteContextPropagatesCancellation

- ソース: [cmd/root_test.go:15](../cmd/root_test.go#L15)
- 判定: **維持** / 状態: 維持
- 守る契約: 合成child commandへcaller cancellationが届きExecuteContextがcontext.Canceledで戻る
- 本番経路: main → cmd.ExecuteContext → Cobra child RunE
- 重複・対応先: なし
- 根拠・限界: 実RootCmdへchildを登録して実行。mainのsignal.NotifyContext、実各commandの登録までは検証しない。1秒待ちは失敗時のhang防止。
- 対応: 維持
- ケース別注意:
- 実行: 1 leaf / 全件PASS。

### T024 — TestExecuteContextReturnsChildErrorSilently

- ソース: [cmd/root_test.go:65](../cmd/root_test.go#L65)
- 判定: **維持** / 状態: 維持
- 守る契約: child errorのcauseを保ち、CobraがエラーとUsageを重複出力しない
- 本番経路: main → cmd.ExecuteContext → RootCmd.SilenceErrors/SilenceUsage
- 重複・対応先: なし
- 根拠・限界: errors.Isと実stdout/stderrによる振る舞いassertion。main自身のstderr/exit codeは別契約。
- 対応: 維持
- ケース別注意:
- 実行: 1 leaf / 全件PASS。

### T025 — TestRootCommandMetadataAndFlags

- ソース: [cmd/root_test.go:93](../cmd/root_test.go#L93)
- 判定: **維持** / 状態: 対応済み
- 守る契約: rootの説明、--version/-vの定義、不要profile/region/cluster/service flag不存在
- 本番経路: RootCmd → Cobra help/flag parse
- 重複・対応先: TestExecuteContextReturnsChildErrorSilently / TestVersionCommandsWriteIdenticalOutput / TestRootHelpDocumentsAWSSetup
- 根拠・限界: Silence boolの直接assertは実child errorテストに統合。固有の説明/flag定義は維持。
- 対応: LV10対応完了。維持。
- ケース別注意: 単一ケース。
- 実行: 1 leaf / 全件PASS。

### T026 — TestRootHelpDocumentsAWSSetup

- ソース: [cmd/root_test.go:132](../cmd/root_test.go#L132)
- 判定: **維持** / 状態: 維持
- 守る契約: AWS設定chain、環境変数、aws-vault例、plugin/PATH要件をhelpに示す
- 本番経路: RootCmd.Help → Cobra help rendering
- 重複・対応先: なし
- 根拠・限界: ユーザーのセットアップに必要な説明契約。文字列変更時の保守は必要だが単なる見栄えのsnapshotではない。
- 対応: 維持
- ケース別注意:
- 実行: 1 leaf / 全件PASS。

### T028 — TestVersionCommandsWriteIdenticalOutput

- ソース: [cmd/root_test.go:145](../cmd/root_test.go#L145)
- 判定: **維持** / 状態: 対応済み
- 守る契約: version/--versionが設定した版を改行付きstdoutへ出しstderrが空
- 本番経路: RootCmd.ExecuteContext → writeVersion
- 重複・対応先: なし
- 根拠・限界: 各caseが同じ具体値をassertするので最後の出力相互比較を削除。2入口の保護は維持。
- 対応: LV10対応完了。維持。
- ケース別注意: subcommand / flag の2case。
- 実行: 2 leaf / 全件PASS。

| 実行サブケース | 個別判定 |
| --- | --- |
| `flag` | 維持 |
| `subcommand` | 維持 |

### T029 — TestVersionCommandsReturnWriteErrors

- ソース: [cmd/root_test.go:173](../cmd/root_test.go#L173)
- 判定: **維持** / 状態: 維持
- 守る契約: versionと--version両入口がwrite失敗のcauseと操作文脈を返す
- 本番経路: RootCmd.ExecuteContext → writeVersion → OutOrStdout.Write
- 重複・対応先: なし
- 根拠・限界: 正常出力テストでは守れない失敗伝播。2入口が共通helperを使い続けることも確認。
- 対応: 維持
- ケース別注意: subcommand / flag の両経路を維持。
- 実行: 2 leaf / 全件PASS。

| 実行サブケース | 個別判定 |
| --- | --- |
| `flag` | 維持 |
| `subcommand` | 維持 |

### T030 — TestVerifyCandidateVersion

- ソース: [cmd/update/install_test.go:40](../cmd/update/install_test.go#L40)
- 判定: **維持** / 状態: 維持
- 守る契約: 候補を version 引数で起動、v/空白正規化、一致/不一致、空出力、非ゼロexit、取消/期限/pipe保持、欠損pathを処理
- 本番経路: main の update import → init → UpdateCmd → productionUpdater().run → updater.run → verifyCandidateVersion
- 重複・対応先: TestVerifyCandidateVersion（旧名から移管済み） の3正常caseは本テストに包含。TestReadBinaryVersionBoundsInheritedStdoutPipeCleanupは別probe実装。
- 根拠・限界: install_test.go:40-189 が11subtestで実子processとerror chainを保護。TestMain:28-37 は再実行したtest binaryを親/子helperに分岐し、child stdout保持を再現するfixture。
- 対応: 維持。実リリースbinaryの代わりにshell/TestMain helperだが、exec/argv/context/pipe境界は実動作。
- ケース別注意: 11件: unprefixed→prefixed、prefixed→unprefixed、surrounding whitespace、mismatch canonical message、empty、whitespace-only、nonzero exit (*exec.ExitError)、parent cancellation、50ms deadline (<2s)、inherited stdout (<500ms)、missing path (os.ErrNotExist)。
- 実行: 11 leaf / 全件PASS。

| 実行サブケース | 個別判定 |
| --- | --- |
| `bounds_inherited_stdout_pipe_cleanup` | 維持 |
| `honors_short_parent_deadline_promptly` | 維持 |
| `matches_prefixed_candidate_to_unprefixed_release` | 維持 |
| `matches_unprefixed_candidate_to_prefixed_release` | 維持 |
| `normalizes_surrounding_whitespace` | 維持 |
| `preserves_parent_cancellation` | 維持 |
| `rejects_empty_output` | 維持 |
| `rejects_mismatched_version_with_canonical_values` | 維持 |
| `rejects_whitespace_only_output` | 維持 |
| `wraps_missing_candidate_path_error` | 維持 |
| `wraps_nonzero_process_error_with_operation` | 維持 |

### T031 — TestReplaceExecutablePreservesLegacySentinel

- ソース: [cmd/update/install_test.go:192](../cmd/update/install_test.go#L192)
- 判定: **維持** / 状態: 維持
- 守る契約: 成功更新時に既存 .tnnl.new のinode/権限/内容を壊さない
- 本番経路: main の update import → init → UpdateCmd → productionUpdater().run → replaceExecutable
- 重複・対応先: TestReplaceExecutableSuccess とsetupのみ重複
- 根拠・限界: install_test.go:192-231 はlegacy固定一時名への回帰を捕捉。成功テストへ併合可能だが保護は独自で、失敗時sentinel保存だけでは成功分岐を保証しない。
- 対応: 維持、または成功テストにsentinel assertionを移して1fixtureに統合。assertion自体は残す。
- ケース別注意: 単一ケース
- 実行: 1 leaf / 全件PASS。

### T032 — TestReplaceExecutableSuccess

- ソース: [cmd/update/install_test.go:234](../cmd/update/install_test.go#L234)
- 判定: **維持** / 状態: 維持
- 守る契約: 候補内容へ置換、権限755、自己一時ファイル残留なし
- 本番経路: main の update import → init → UpdateCmd → productionUpdater().run → replaceExecutable
- 重複・対応先: TestUpdaterInstallsOnlyAfterVerification
- 根拠・限界: install_test.go:234-269 は実filesystemで内容/permission/cleanupを直接assert。上位成功テストはtemp残留を見ない。
- 対応: 維持。lower-level cleanupの固有保護がある。
- ケース別注意: 単一ケース
- 実行: 1 leaf / 全件PASS。

### T033 — TestReplaceExecutableFailureCleansUpOnlyOwnTemp

- ソース: [cmd/update/install_test.go:272](../cmd/update/install_test.go#L272)
- 判定: **維持** / 状態: 対応済み
- 守る契約: candidate open/copy失敗時に旧current通常fileのbytes・mode・identityが保存される。全失敗ケースで他人のtemp/sentinelを保存し、自分のtempだけをcleanupする。
- 本番経路: updater.run → replaceExecutable → candidate open/copy / replacement temp creation / rename / deferred cleanup
- 重複・対応先: なし
- 根拠・限界: candidate open/copyの2ケースでcurrentPathの事前snapshotをpreservedFilesへ登録し、既存assertionでos.SameFile・Mode・内容を独立比較。異常targetを作るcreate temp/renameは通常current保存と区別。内容破壊・mode変更・identity置換の各変異を両ケースが検出した。
- 対応: Q07対応完了。維持。
- ケース別注意: 4 leaf: candidate open / candidate copy は通常currentの3保証を追加。create temp / rename は他人のfile保護と自己temp cleanupの既存保証を維持。
- 実行: 4 leaf / 全件PASS。

| 実行サブケース | 個別判定 |
| --- | --- |
| `candidate_copy` | 維持 |
| `candidate_open` | 維持 |
| `create_temp` | 維持 |
| `rename` | 維持 |

### T034 — TestUpdaterInstallsOnlyAfterVerification

- ソース: [cmd/update/orchestration_test.go:27](../cmd/update/orchestration_test.go#L27)
- 判定: **維持** / 状態: 維持
- 守る契約: HTTP最新解決→archive/checksums取得→展開→候補起動→実ファイル更新の成功、内容/実行権限/出力/URL順序
- 本番経路: main の update import → init → UpdateCmd → productionUpdater().run → updater.run の全成功経路
- 重複・対応先: TestReplaceExecutableSuccess は局所ファイル境界、TestUpdaterInstallsOnlyAfterVerification（旧名から移管済み） は通常URL結合
- 根拠・限界: orchestration_test.go:31-57 がHTTP・gzip/tar・shell subprocess・ファイルを実際に通す。名前の only after verification はこの成功例単独では証明せず、失敗系と組み合わせて成立。
- 対応: 維持。名前は successful update の保護を表すと正確。実GoReleaser binary/productionUpdater初期化はfixtureで代替。
- ケース別注意: 単一ケース
- 実行: 1 leaf / 全件PASS。

### T035 — TestUpdaterStagesCandidateBesideExecutable

- ソース: [cmd/update/orchestration_test.go:61](../cmd/update/orchestration_test.go#L61)
- 判定: **維持** / 状態: 対応済み
- 守る契約: 候補version process自身が観測した実行pathから、stage directoryが更新対象exeの親directory直下にあることを確認。無効TMPDIRでも更新でき、最後にcandidate bytesがinstalledであることも保持。
- 本番経路: updater.run → os.MkdirTemp → download/checksum/extract → verifyCandidateVersion実process → replaceExecutable
- 重複・対応先: TestUpdaterInstallsOnlyAfterVerification（成功/内容）
- 根拠・限界: 期待pathを本番MkdirTempから作らず、候補processの$0と元exeの親を独立比較。別の有効な祖先directoryへstagingする変異を検出済み。
- 対応: Q06対応済み。隣接stagingを直接観測する回帰テストとして維持。
- ケース別注意: 単独ケース。fixture候補scriptにpath記録を追加しarchive/checksumを再構成。TMPDIR不使用の保証も残す。
- 実行: 1 leaf / 全件PASS。

### T036 — TestUpdaterLeavesExecutableUntouchedOnChecksumMismatch

- ソース: [cmd/update/orchestration_test.go:93](../cmd/update/orchestration_test.go#L93)
- 判定: **維持** / 状態: 維持
- 守る契約: 正しい形式のarchiveでもdigest不一致なら更新せず元内容/権限を保存
- 本番経路: main の update import → init → UpdateCmd → productionUpdater().run → updater.run → verifyFileSHA256
- 重複・対応先: TestVerifyFileSHA256/rejects one byte difference、TestUpdaterDoesNotExtractBeforeChecksumVerification
- 根拠・限界: orchestration_test.go:95-101 は検証gateを外す回帰で実行ファイルが置換され失敗する。下位hash検証だけでは配線欠落を守れない。
- 対応: 維持。後者は不正gzipで順序を別途守るため同一ではない。
- ケース別注意: 単一ケース
- 実行: 1 leaf / 全件PASS。

### T037 — TestUpdaterDoesNotExtractBeforeChecksumVerification

- ソース: [cmd/update/orchestration_test.go:104](../cmd/update/orchestration_test.go#L104)
- 判定: **維持** / 状態: 維持
- 守る契約: 不正gzipよりchecksum mismatchが先に返り元実行ファイル保存
- 本番経路: main の update import → init → UpdateCmd → productionUpdater().run → updater.run → checksum → extractBinaryFromArchive
- 重複・対応先: なし
- 根拠・限界: orchestration_test.go:106-116 で二重不良fixtureを与えエラー順序まで検証。単なるchecksum mismatch正常archiveケースにない順序の保護。
- 対応: 維持
- ケース別注意: 単一ケース
- 実行: 1 leaf / 全件PASS。

### T038 — TestUpdaterLeavesExecutableUntouchedOnCandidateVersionMismatch

- ソース: [cmd/update/orchestration_test.go:119](../cmd/update/orchestration_test.go#L119)
- 判定: **維持** / 状態: 維持
- 守る契約: digest正常でも候補versionがreleaseと違う場合に元内容/権限を保存
- 本番経路: main の update import → init → UpdateCmd → productionUpdater().run → updater.run → verifyCandidateVersion → replaceExecutable
- 重複・対応先: なし
- 根拠・限界: orchestration_test.go:119-126 は検証gate省略を検知。helperの文字列比較だけでは本番で呼ばれることを守れない。
- 対応: 維持
- ケース別注意: 単一ケース
- 実行: 1 leaf / 全件PASS。

### T039 — TestUpdaterPropagatesCallerCancellation

- ソース: [cmd/update/orchestration_test.go:129](../cmd/update/orchestration_test.go#L129)
- 判定: **維持** / 状態: 維持
- 守る契約: update開始前取消でcontext.Canceledを呼出元へ返し既存実行ファイルを保存
- 本番経路: main の update import → init → UpdateCmd → productionUpdater().run → updater.run → fetchLatestRelease
- 重複・対応先: TestFetchLatestReleasePreservesCallerCancellation
- 根拠・限界: orchestration_test.go:131-138 はrunからHTTPへのcontext配線を守る。downloadや候補probe中の取消は保証しない。
- 対応: 維持。境界をまたぐcontext伝播として合理的な重複。
- ケース別注意: 単一ケース
- 実行: 1 leaf / 全件PASS。

### T040 — TestUpdaterCancellationStopsCurrentVersionProbe

- ソース: [cmd/update/orchestration_test.go:141](../cmd/update/orchestration_test.go#L141)
- 判定: **維持** / 状態: 維持
- 守る契約: version probe中の取消がrunを終了しassetを一切取得せず元ファイル保存
- 本番経路: main の update import → init → UpdateCmd → productionUpdater().run → updater.run → currentVersion
- 重複・対応先: なし
- 根拠・限界: orchestration_test.go:157-174 で起動後取消、帰還、asset countゼロ、内容/権限を検査。下位取消だけではrunが続行しないことを保証できない。
- 対応: 維持
- ケース別注意: 単一ケース
- 実行: 1 leaf / 全件PASS。

### T041 — TestUpdaterAlreadyLatestMakesNoAssetRequests

- ソース: [cmd/update/orchestration_test.go:177](../cmd/update/orchestration_test.go#L177)
- 判定: **維持** / 状態: 維持
- 守る契約: 最新版一致なら更新せずassetを取得せずstatusだけ返す
- 本番経路: main の update import → init → UpdateCmd → productionUpdater().run → updater.run current==latest 分岐
- 重複・対応先: なし
- 根拠・限界: orchestration_test.go:181-190 が副作用の否定まで検査。成功更新テストとは別の経路。
- 対応: 維持
- ケース別注意: 単一ケース
- 実行: 1 leaf / 全件PASS。

### T042 — TestUpdaterReturnsOutputError

- ソース: [cmd/update/orchestration_test.go:193](../cmd/update/orchestration_test.go#L193)
- 判定: **維持** / 状態: 対応済み
- 守る契約: already-latest/updatedの両方でwriter errorのcauseとwrite update status文脈を返す。前者は旧file不変、後者はcandidate bytesと0755が既に反映済みであることを確認。
- 本番経路: updater.run → currentVersion/HTTP/検証/置換 → status fmt.Fprintf
- 重複・対応先: なし
- 根拠・限界: 更新後のwriter失敗を実HTTP/archive/process/file経路で検証。statusを置換前へ移動する変異と、更新後writer errorを無視する変異の両方を検出済み。
- 対応: Q11対応済み。成功statusではなく更新副作用と返却errorの組合せを維持。
- ケース別注意: already latest / updated の2subtest。
- 実行: 2 leaf / 全件PASS。

| 実行サブケース | 個別判定 |
| --- | --- |
| `already_latest` | 維持 |
| `updated` | 維持 |

### T043 — TestUpdaterAssetRedirectsUseOriginalClientPolicy

- ソース: [cmd/update/orchestration_test.go:232](../cmd/update/orchestration_test.go#L232)
- 判定: **維持** / 状態: 維持
- 守る契約: latest取得の独自redirect停止が後続archive/manifest redirect policyに漏れない
- 本番経路: main の update import → init → UpdateCmd → productionUpdater().run → fetchLatestRelease → downloadFile×2
- 重複・対応先: なし
- 根拠・限界: orchestration_test.go:232-248 は実httptest redirectを2回通す。unitのclient不変性だけでは後続が同じclientを利用する配線を保証しない。
- 対応: 維持
- ケース別注意: 単一ケース
- 実行: 1 leaf / 全件PASS。

### T044 — TestUpdaterWrapsExecutableResolutionError

- ソース: [cmd/update/orchestration_test.go:252](../cmd/update/orchestration_test.go#L252)
- 判定: **維持** / 状態: 維持
- 守る契約: 更新対象pathの解決エラーに操作contextを付け原因保持
- 本番経路: main の update import → init → UpdateCmd → productionUpdater().run → updater.run executablePath
- 重複・対応先: なし
- 根拠・限界: orchestration_test.go:254-258 は初手停止とerror chainを守る。nil clientのfixtureなので後続へ進めば望むerrorにならずfailする。実resolver自体の正しさは対象外。
- 対応: 維持
- ケース別注意: 単一ケース
- 実行: 1 leaf / 全件PASS。

### T045 — TestNewUpdateCommandUsesContextAndOutput

- ソース: [cmd/update/orchestration_test.go:262](../cmd/update/orchestration_test.go#L262)
- 判定: **維持** / 状態: 維持
- 守る契約: Cobra ExecuteContextのcontext値と指定出力writerをrunnerへ渡す
- 本番経路: main の update import → init → UpdateCmd → productionUpdater().run → newUpdateCommand.RunE
- 重複・対応先: なし
- 根拠・限界: orchestration_test.go:266-280 はrunner内contextとwriter内容を実行後に照合。本番run本体とCobra境界を分けた必要な配線テスト。
- 対応: 維持
- ケース別注意: 単一ケース
- 実行: 1 leaf / 全件PASS。

### T046 — TestNewUpdateCommandReturnsRunnerError

- ソース: [cmd/update/orchestration_test.go:283](../cmd/update/orchestration_test.go#L283)
- 判定: **維持** / 状態: 維持
- 守る契約: Cobra RunEがrunnerのsentinel errorを保持
- 本番経路: main の update import → init → UpdateCmd → productionUpdater().run → newUpdateCommand.RunE
- 重複・対応先: なし
- 根拠・限界: orchestration_test.go:283-288 は短いがRunEでerr握り潰す回帰を捕捉。標準Cobra自体を試すだけでなく独自closureを通る。
- 対応: 維持
- ケース別注意: 単一ケース
- 実行: 1 leaf / 全件PASS。

### T173 — TestResolveTargetExecutablePath

- ソース: [cmd/update/path_test.go:14](../cmd/update/path_test.go#L14)
- 判定: **維持** / 状態: 対応済み
- 守る契約: 実os.Executable/LookPath/EvalSymlinksによる更新対象選択。通常自身、起動symlink、別basename、custom go-run directory、system temp、PATH symlink、PATH欠落fallback、go-builder誤認防止。
- 本番経路: helper子process → resolveTargetExecutablePath → os.Executable → resolveSymlinkPath → shouldUsePathBinary → exec.LookPath → resolveSymlinkPath
- 重複・対応先: なし
- 根拠・限界: 注入なしで実resolverを呼ぶ。test実行ファイルを専用fixture名で起動し、PATHとTMPDIRを独立fixtureに閉じ込め、期待値はfixture構造から直接指定。本番パス更新やAWS接続は行わない。
- 対応: Q08対応完了。8実processケースを維持。
- ケース別注意: Darwin/Linuxの8ケース。helperは同一Test関数をenv/-test.runで再起動し結果fileを返す。10秒context＋WaitDelay、hardlink不可時はcopyへfallback。
- 実行: 8 leaf / 全件PASS。

| 実行サブケース | 個別判定 |
| --- | --- |
| `PATH_symlink_resolves_to_target` | 維持 |
| `custom_go_run_directory_uses_PATH` | 維持 |
| `installed_binary_keeps_itself` | 維持 |
| `launched_symlink_resolves_to_installed_binary` | 維持 |
| `missing_PATH_binary_falls_back_to_running_executable` | 維持 |
| `other_executable_name_uses_PATH` | 維持 |
| `similar_go_builder_directory_keeps_itself` | 維持 |
| `system_temporary_directory_uses_PATH` | 維持 |

### T047 — TestChecksumForAsset

- ソース: [cmd/update/release_test.go:12](../cmd/update/release_test.go#L12)
- 判定: **維持** / 状態: 維持
- 守る契約: manifestから完全一致assetの唯一のSHA256を選び曖昧/不正形式を拒否
- 本番経路: main の update import → init → UpdateCmd → productionUpdater().run → updater.run → checksumForAsset
- 重複・対応先: TestUpdaterLeavesExecutableUntouchedOnChecksumMismatch はdigest検証でありmanifest構文を代替しない
- 根拠・限界: release_test.go:13-117 は9種類の入力契約。prefix一致とmissingは同じエラー分岐だが誤ったsuffix matchingだけを前者が狙うので有用。
- 対応: 維持。scanner長大行のerrorは未検証。
- ケース別注意: 9件: exact filename、suffix-only拒否、missing拒否、duplicate拒否、4 fields拒否、digest長不正、nonhex、blank lines許容、GNU * marker許容。
- 実行: 9 leaf / 全件PASS。

| 実行サブケース | 個別判定 |
| --- | --- |
| `accepts_GNU_binary_marker_for_exact_filename` | 維持 |
| `allows_blank_lines` | 維持 |
| `does_not_match_filename_that_merely_ends_with_asset_name` | 維持 |
| `rejects_duplicate_exact_entries` | 維持 |
| `rejects_malformed_nonblank_line` | 維持 |
| `rejects_matching_digest_with_wrong_length` | 維持 |
| `rejects_matching_nonhex_digest` | 維持 |
| `rejects_missing_exact_entry` | 維持 |
| `selects_exact_filename` | 維持 |

### T048 — TestVerifyFileSHA256

- ソース: [cmd/update/release_test.go:120](../cmd/update/release_test.go#L120)
- 判定: **維持** / 状態: 維持
- 守る契約: 実ファイルの一致/1byte差分不一致/ファイル欠損の操作付きerror
- 本番経路: main の update import → init → UpdateCmd → productionUpdater().run → updater.run → verifyFileSHA256
- 重複・対応先: TestUpdaterLeavesExecutableUntouchedOnChecksumMismatch
- 根拠・限界: release_test.go:121-149 は実ファイルhash計算を検査。orchestrationは配線と無更新、単体は小入力とエラー局所化を担当。
- 対応: 維持。read途中errorは未検証。
- ケース別注意: 3件: matching file、one byte difference、missing fileのpath入りerror。
- 実行: 3 leaf / 全件PASS。

| 実行サブケース | 個別判定 |
| --- | --- |
| `accepts_matching_file` | 維持 |
| `rejects_one_byte_difference` | 維持 |
| `reports_open_error_with_path` | 維持 |

### T049 — TestUpdateHelpDocumentsVerificationAndReplacement

- ソース: [cmd/update/update_test.go:25](../cmd/update/update_test.go#L25)
- 判定: **維持** / 状態: 維持
- 守る契約: help に checksum / candidate version / same-directory atomic replacement / write permission の4文言を掲載
- 本番経路: main の update import → init → UpdateCmd → productionUpdater().run → newUpdateCommand.Help
- 重複・対応先: 実動作は TestUpdaterInstallsOnlyAfterVerification、checksum/version mismatch と staging の各テスト
- 根拠・限界: 検証・置換・非対応条件の説明がhelpに残ることを守る。動作の証明ではないが、説明契約として独立価値がある。文字列固定だけを理由に低価値とは判定しない。
- 対応: 説明契約として維持。挙動保証はorchestrationテストで担う。
- ケース別注意: 単一ケース
- 実行: 1 leaf / 全件PASS。

### T050 — TestFetchLatestReleaseUsesInjectedClientWithoutAuthorization

- ソース: [cmd/update/update_test.go:52](../cmd/update/update_test.go#L52)
- 判定: **維持** / 状態: 維持
- 守る契約: 指定 client/URL を利用し Authorization を付けず、最新リダイレクトから tag と query/fragment を除いた配布 base URL を取得
- 本番経路: main の update import → init → UpdateCmd → productionUpdater().run → fetchLatestRelease
- 重複・対応先: TestUpdaterInstallsOnlyAfterVerification はリクエスト配線の上位保証
- 根拠・限界: update_test.go:55-80 で transport、URL、header、取得値を検証する。実HTTP接続は差し替えるが client の注入契約を直接守る。
- 対応: 維持。環境由来の認証追加も禁止する契約なら、その環境を明示したケースにする。
- ケース別注意: 単一ケース
- 実行: 1 leaf / 全件PASS。

### T051 — TestFetchLatestReleaseDoesNotMutateInjectedClient

- ソース: [cmd/update/update_test.go:84](../cmd/update/update_test.go#L84)
- 判定: **維持** / 状態: 維持
- 守る契約: latest プローブだけ redirect 追跡を止め、共有clientの元 CheckRedirect を保存
- 本番経路: main の update import → init → UpdateCmd → productionUpdater().run → fetchLatestRelease → client の shallow copy
- 重複・対応先: TestUpdaterAssetRedirectsUseOriginalClientPolicy
- 根拠・限界: update_test.go:101-111 で元policy未実行と元policyが存続することを検査。download が同じclientを使う本番経路に対応。
- 対応: 維持。上位テストは後続asset通信、こちらはclient不変性を局所的に診断する。
- ケース別注意: 単一ケース
- 実行: 1 leaf / 全件PASS。

### T052 — TestFetchLatestReleaseRejectsUserinfoBeforeRequest

- ソース: [cmd/update/update_test.go:115](../cmd/update/update_test.go#L115)
- 判定: **維持** / 状態: 維持
- 守る契約: URL userinfo をHTTP送信前に拒否し transport を呼ばない
- 本番経路: main の update import → init → UpdateCmd → productionUpdater().run → fetchLatestRelease → parsePublicHTTPURL
- 重複・対応先: なし
- 根拠・限界: update_test.go:116-123 が送信前拒否まで確認。URL parserの拒否テストだけでは fetch 経路のvalidation抜けを捕捉できない。
- 対応: 維持
- ケース別注意: 単一ケース
- 実行: 1 leaf / 全件PASS。

### T053 — TestFetchLatestReleasePreservesCallerCancellation

- ソース: [cmd/update/update_test.go:127](../cmd/update/update_test.go#L127)
- 判定: **維持** / 状態: 維持
- 守る契約: 取消済みcontextで latest 取得が errors.Is(context.Canceled) を保つ
- 本番経路: main の update import → init → UpdateCmd → productionUpdater().run → fetchLatestRelease
- 重複・対応先: TestUpdaterPropagatesCallerCancellation
- 根拠・限界: update_test.go:127-134 は実http.Clientの取消経路。上位テストと重なるが単独APIのcontext wrapを局所化できる。
- 対応: 維持。取消前のみで、通信中取消は別の未検証範囲。
- ケース別注意: 単一ケース
- 実行: 1 leaf / 全件PASS。

### T054 — TestReleaseFromLatestLocation

- ソース: [cmd/update/update_test.go:137](../cmd/update/update_test.go#L137)
- 判定: **維持** / 状態: 維持
- 守る契約: escaped slash/percent を二重decodeせず tag/配布URLを生成し、query/fragment破棄と大文字scheme正規化
- 本番経路: main の update import → init → UpdateCmd → productionUpdater().run → fetchLatestRelease → releaseFromLatestLocation
- 重複・対応先: なし
- 根拠・限界: update_test.go:138-154 は特殊パスの具体的期待値。通常のfixture URLでは通らないエンコードの分岐を守る。
- 対応: 維持
- ケース別注意: 同一関数内の2入力: proxy%2Ftenant + v1%2Frc%252F1、および HTTPS 大文字scheme。
- 実行: 1 leaf / 全件PASS。

### T055 — TestReleaseFromLatestLocationRejectsUnsafeOrAmbiguousURL

- ソース: [cmd/update/update_test.go:158](../cmd/update/update_test.go#L158)
- 判定: **維持** / 状態: 維持
- 守る契約: 絶対HTTP(S) URL・host・userinfoなし・release tag 1セグメントという入力契約を拒否例で保護
- 本番経路: main の update import → init → UpdateCmd → productionUpdater().run → fetchLatestRelease → releaseFromLatestLocation → parsePublicHTTPURL
- 重複・対応先: なし
- 根拠・限界: update_test.go:159-175 は全11拒否ケース。非nil errorのみだが、目的は不正URLの受理を防ぐことで各ケースが異なる入力境界。
- 対応: 維持
- ケース別注意: 11件: 空、相対、scheme-relative、FTP、host欠落、userinfo、tag marker欠落、空tag、余計なsegment、encoded marker、invalid percent escape。
- 実行: 11 leaf / 全件PASS。

| 実行サブケース | 個別判定 |
| --- | --- |
| `#00` | 維持 |
| `//example.test/releases/tag/v1.2.3` | 維持 |
| `/releases/tag/v1.2.3` | 維持 |
| `ftp://example.test/releases/tag/v1.2.3` | 維持 |
| `https://example.test/%2Freleases%2Ftag%2Fv1.2.3` | 維持 |
| `https://example.test/releases/latest` | 維持 |
| `https://example.test/releases/tag/` | 維持 |
| `https://example.test/releases/tag/%zz` | 維持 |
| `https://example.test/releases/tag/v1.2.3/extra` | 維持 |
| `https://user:password@example.test/releases/tag/v1.2.3` | 維持 |
| `https:/releases/tag/v1.2.3` | 維持 |

### T058 — TestShouldUsePathBinary

- ソース: [cmd/update/update_test.go:180](../cmd/update/update_test.go#L180)
- 判定: **維持** / 状態: 対応済み
- 守る契約: 自身維持・別basename・system temp・TempDir外go-build＋数字・exact go-buildを区別し、temp隣接prefix/go-builder/go-build-tools/go-build123extra/my-go-build123を誤認しない。
- 本番経路: resolveTargetExecutablePath → shouldUsePathBinary → PATH探索の要否
- 重複・対応先: なし
- 根拠・限界: TMPDIRを独立fixtureへ設定してgo-build分岐をsystem temp条件から分離。旧実装では類似prefix3caseがFAILし、segment判定修正後にPASS。
- 対応: Q08対応完了。現行10caseを維持。
- ケース別注意: 10ケース。新規7ケースにより重なるtemp条件だけでは通らない境界を確認。
- 実行: 10 leaf / 全件PASS。

| 実行サブケース | 個別判定 |
| --- | --- |
| `exact_go_build_directory` | 維持 |
| `go_build_must_start_a_directory_name` | 維持 |
| `go_build_numeric_prefix_needs_a_boundary` | 維持 |
| `go_build_outside_system_temp` | 維持 |
| `go_build_tools_is_an_installed_directory` | 維持 |
| `go_builder_is_an_installed_directory` | 維持 |
| `normal_installed_path` | 維持 |
| `not_tnnl_binary_name` | 維持 |
| `similar_temporary_directory_prefix` | 維持 |
| `temp_go_run_path` | 維持 |

### T059 — TestCurrentVersion_FromBinary

- ソース: [cmd/update/update_test.go:253](../cmd/update/update_test.go#L253)
- 判定: **維持** / 状態: 維持
- 守る契約: 対象実行ファイルを version 引数で起動し埋込値より優先、v prefix除去
- 本番経路: main の update import → init → UpdateCmd → productionUpdater().run → currentVersion → readBinaryVersion
- 重複・対応先: TestUpdaterInstallsOnlyAfterVerification
- 根拠・限界: update_test.go:256 のshellは version 引数以外に失敗するため実 subprocess 呼び出しも検査。helper単体でなく呼び出し契約を守る。
- 対応: 維持。shell fixture と実リリースbinary の差はあるが subprocess/引数/出力変換は本番と同じ。
- ケース別注意: 単一ケース
- 実行: 1 leaf / 全件PASS。

### T060 — TestCurrentVersion_FallbackToEmbedded

- ソース: [cmd/update/update_test.go:270](../cmd/update/update_test.go#L270)
- 判定: **維持** / 状態: 維持
- 守る契約: 実行不能時に埋込 buildinfo.Current の正規化値へフォールバック
- 本番経路: main の update import → init → UpdateCmd → productionUpdater().run → currentVersion
- 重複・対応先: TestVerifyCandidateVersion（旧名から移管済み） / TestVerifyCandidateVersion は正規化のみ
- 根拠・限界: update_test.go:271-276 が唯一のfallback保証。期待値にも normalizeVersion を使うため変換自体は独立ではないが、別テストが変換を保証しており主目的はfallback。
- 対応: 維持。空出力/非ゼロexit時のfallbackは未実行だが同じerr分岐。
- ケース別注意: 単一ケース
- 実行: 1 leaf / 全件PASS。

### T061 — TestCurrentVersionPreservesCallerCancellation

- ソース: [cmd/update/update_test.go:280](../cmd/update/update_test.go#L280)
- 判定: **維持** / 状態: 維持
- 守る契約: 実行中version probeの取消を速やかに返し、embedded fallbackで握り潰さない
- 本番経路: main の update import → init → UpdateCmd → productionUpdater().run → currentVersion → readBinaryVersion
- 重複・対応先: TestUpdaterCancellationStopsCurrentVersionProbe
- 根拠・限界: update_test.go:296 で子起動同期後に取消し、errors.Isと2秒以内帰還を検査。取消済みcontextだけのテストとは異なる。
- 対応: 維持。上位の取消配線と下位のfallback抑止は別の責務。
- ケース別注意: 単一ケース
- 実行: 1 leaf / 全件PASS。

### T062 — TestReadBinaryVersionBoundsInheritedStdoutPipeCleanup

- ソース: [cmd/update/update_test.go:312](../cmd/update/update_test.go#L312)
- 判定: **維持** / 状態: 維持
- 守る契約: 子孫がstdout pipeを保持しても installed binary probe が期限切れ後に短時間で終了
- 本番経路: main の update import → init → UpdateCmd → productionUpdater().run → currentVersion → readBinaryVersion
- 重複・対応先: TestVerifyCandidateVersion/bounds inherited stdout pipe cleanup
- 根拠・限界: update_test.go:312-345 がTestMain helperの子孫pipeとmanual contextを利用し WaitDelay 欠落を検出。verifyCandidateVersion は別のexec.Command生成なので重複に見えても両方必要。
- 対応: 維持。500ms上限は200ms WaitDelayと1秒子寿命を区別するが負荷時の余裕に注意。
- ケース別注意: 単一ケース
- 実行: 1 leaf / 全件PASS。

### T063 — TestDownloadFileUsesInjectedClientWithoutAuthorization

- ソース: [cmd/update/update_test.go:348](../cmd/update/update_test.go#L348)
- 判定: **維持** / 状態: 維持
- 守る契約: 注入clientで認証headerを追加せずbodyを実ファイルへ保存
- 本番経路: main の update import → init → UpdateCmd → productionUpdater().run → downloadFile
- 重複・対応先: TestUpdaterInstallsOnlyAfterVerification（同じno-auth/content経路）
- 根拠・限界: update_test.go:350-369 がheaderと保存内容を照合。上位fixtureもno-authを拒否するが個別download APIの回帰を局所化する。
- 対応: 維持。URL/method/User-Agent/通信途中errorの保証はこのテストにはない。
- ケース別注意: 単一ケース
- 実行: 1 leaf / 全件PASS。

### T064 — TestDownloadFilePreservesCallerCancellation

- ソース: [cmd/update/update_test.go:373](../cmd/update/update_test.go#L373)
- 判定: **維持** / 状態: 維持
- 守る契約: 取消済みdownloadでcontext.Canceledを保ち宛先ファイルを作らない
- 本番経路: main の update import → init → UpdateCmd → productionUpdater().run → downloadFile
- 重複・対応先: なし
- 根拠・限界: update_test.go:378-383 は cancellation と無副作用の両方。上位取消はlatest段階で止まるためdownloadの取消を代替しない。
- 対応: 維持
- ケース別注意: 単一ケース
- 実行: 1 leaf / 全件PASS。

### T065 — TestCurrentCanonicalizesLinkerVersion

- ソース: [internal/buildinfo/version_test.go:8](../internal/buildinfo/version_test.go#L8)
- 判定: **維持** / 状態: 維持
- 守る契約: Current公開入口がlinkerVersionの空白と先頭vを正規化して返す
- 本番経路: cmd.Version / updater fallback → buildinfo.Current → resolveVersion
- 重複・対応先: なし
- 根拠・限界: private helperのtableと異なり公開入口の配線を守る。linker変数をテスト内で変更する単体試験であり、実ldflagsのシンボル名からCLI出力までの保証はT171 TestCLIBinaryContractsが補完済み。
- 対応: 維持
- ケース別注意:
- 実行: 1 leaf / 全件PASS。

### T066 — TestResolveVersion

- ソース: [internal/buildinfo/version_test.go:20](../internal/buildinfo/version_test.go#L20)
- 判定: **維持** / 状態: 対応済み
- 守る契約: 通常linker版優先、module/pseudo版のfallback、local VCS/dev扱い、buildinfo不在、空白/先頭v正規化の15case
- 本番経路: buildinfo.Current → resolveVersion
- 重複・対応先: TestVerifyCandidateVersion（旧名から移管済み）(別用途の同型正規化、一律統合は不要)
- 根拠・限界: 特殊値4caseを整理し、通常配布版とmodule/dev経路の15caseを維持。実ldflagsからの配線はTestCLIBinaryContractsが補完。
- 対応: LV11対応完了。非標準versionの現在動作自体は変更なし。
- ケース別注意: 全15case維持。
- 実行: 15 leaf / 全件PASS。

| 実行サブケース | 個別判定 |
| --- | --- |
| `dev_linker_falls_back_to_module` | 維持 |
| `dev_module_version_returns_dev` | 維持 |
| `devel_module_version_returns_dev` | 維持 |
| `empty_linker_falls_back_to_module` | 維持 |
| `empty_module_version_returns_dev` | 維持 |
| `linker_version_takes_precedence` | 維持 |
| `linker_version_wins_over_local_VCS_build` | 維持 |
| `local_VCS_pseudo_version_returns_dev` | 維持 |
| `module_proxy_pseudo_version_is_preserved` | 維持 |
| `nil_build_info_returns_dev` | 維持 |
| `unavailable_build_info_returns_dev` | 維持 |
| `v_only_linker_falls_back_to_module` | 維持 |
| `v_only_module_version_returns_dev` | 維持 |
| `whitespace_linker_falls_back_to_module` | 維持 |
| `whitespace_module_version_returns_dev` | 維持 |

### T067 — TestExecHandlerPreflightFailureStopsBeforeAWS

- ソース: [internal/handler/exec_test.go:147](../internal/handler/exec_test.go#L147)
- 判定: **維持** / 状態: 維持
- 守る契約: plugin preflight が失敗したら、原因を保持して返し、config読込・ECS/SSM生成・選択・ポート確保へ進まない。
- 本番経路: cmd/exec.ExecCmd → handler.ExecHandler → execHandler → view.ResolveTarget / command.StartExecSession → RemoteSession.Run（試験は execHandler から依存を注入）
- 重複・対応先: TestPortForwardHandlerPreflightFailureStopsBeforeAWS（別実装経路、必要な対）
- 根拠・限界: 147-174: 後続依存を t.Fatal にして副作用抑止を検証。portforwardにも似た試験があるが、本番 exec.go:19 と portforward.go:47 は別の順序制御であり片側削除は片側の回帰を逃す。
- 対応: 維持する。
- ケース別注意: 単一ケース。
- 実行: 1 leaf / 全件PASS。

### T068 — TestExecHandlerSelectsExactSecondReplicaAndRunsInOrder

- ソース: [internal/handler/exec_test.go:177](../internal/handler/exec_test.go#L177)
- 判定: **維持** / 状態: 維持
- 守る契約: 同一serviceの2 replicaから選んだ完全task ARNを ExecuteCommandへ渡し、返却taskのruntime IDを再取得してpluginへ渡す。preflightからpluginまでの順序、元context、service/cluster/container/command/region、成功時にterminateしないこと。
- 本番経路: cmd/exec.ExecCmd → handler.ExecHandler → execHandler → view.ResolveTarget / command.StartExecSession → RemoteSession.Run（試験は execHandler から依存を注入）
- 重複・対応先: TestStartExecSessionUsesAuthoritativeResponseAndFreshRuntimeID; TestResolveTargetSelectsExactReadyTaskARN（異なる境界）
- 根拠・限界: 177-215: 実view/resolver/command/sessionを通す境界統合試験であり、各unitだけでは保証できない選択から実行までの配線を守る。fixtureが Wait=0 のみ、factoryへ渡すcfgとpreflight/loadConfigのcontext同一性は未検証。
- 対応: 維持。別途 Wait>0 の秒→duration伝達と factoryへのcfg伝達を補う。
- ケース別注意: 単一ケース。
- 実行: 1 leaf / 全件PASS。

### T069 — TestExecHandlerPluginCancellationCleansUpWithDetachedContext

- ソース: [internal/handler/exec_test.go:219](../internal/handler/exec_test.go#L219)
- 判定: **維持** / 状態: 維持
- 守る契約: plugin実行中に親contextがcancelされても生きた期限付きcontextで正確なsessionを1度終了し、context valueとplugin/cancel/cleanupの全原因を保持する。
- 本番経路: cmd/exec.ExecCmd → handler.ExecHandler → execHandler → view.ResolveTarget / command.StartExecSession → RemoteSession.Run（試験は execHandler から依存を注入）
- 重複・対応先: TestRemoteSessionCleanupUsesLiveIndependentContext; TestRemoteSessionJoinsCleanupError; TestStartExecSessionTerminatesExactSessionWhenPluginFails（後者の統合先）
- 根拠・限界: 219-264: handlerから実RemoteSession.RunとSSM adapterまで通し、呼出順序も確認。unitのcontext試験と重なるが handlerがエラーを落とす/cleanup前に返る回帰を防ぐ。
- 対応: 維持する。
- ケース別注意: 単一ケース。
- 実行: 1 leaf / 全件PASS。

### T070 — TestExecHandlerViewCancellationCreatesNoRemoteSession

- ソース: [internal/handler/exec_test.go:267](../internal/handler/exec_test.go#L267)
- 判定: **維持** / 状態: 維持
- 守る契約: 選択UIのquitを正常終了として扱い、ExecuteCommand/StartSession/TerminateSession/pluginを起動しない。
- 本番経路: cmd/exec.ExecCmd → handler.ExecHandler → execHandler → view.ResolveTarget / command.StartExecSession → RemoteSession.Run（試験は execHandler から依存を注入）
- 重複・対応先: TestResolveTargetUserCancellation（異なる境界）
- 根拠・限界: 267-285: viewのquit単体試験はquitを返す契約、こちらはhandlerがquitを尊重してremote副作用を出さない契約。必要な境界の重複。
- 対応: 維持する。
- ケース別注意: 単一ケース。
- 実行: 1 leaf / 全件PASS。

### T071 — TestExecHandlerPreservesConfigAndAPIErrors

- ソース: [internal/handler/exec_test.go:289](../internal/handler/exec_test.go#L289)
- 判定: **維持** / 状態: 維持
- 守る契約: config読込失敗は原因と操作文脈を保持してclient生成を止める。resolver API(ListTasks)失敗は原因を保持しExecuteCommandへ進まない。
- 本番経路: cmd/exec.ExecCmd → handler.ExecHandler → execHandler → view.ResolveTarget / command.StartExecSession → RemoteSession.Run（試験は execHandler から依存を注入）
- 重複・対応先: なし
- 根拠・限界: 289-327: configは後続factoryを禁止、resolver APIはexecuteCalls=0を検証。単なるerror wrapper追認ではなく遠隔副作用の遮断を守る。
- 対応: 維持する。
- ケース別注意: config / resolver API の2サブケース。後者はplugin/SSM生成を直接禁止しないが実戻り値とExecuteCommand未呼出を確認。
- 実行: 2 leaf / 全件PASS。

| 実行サブケース | 個別判定 |
| --- | --- |
| `config` | 維持 |
| `resolver_API` | 維持 |

### T072 — TestPortForwardHandlerPreflightFailureStopsBeforeAWS

- ソース: [internal/handler/portforward_test.go:18](../internal/handler/portforward_test.go#L18)
- 判定: **維持** / 状態: 維持
- 守る契約: preflight失敗時に原因を保持し、config・client生成・選択・local port確保より前に停止する。
- 本番経路: cmd/(remote)portforward → handler.(Remote)PortforwardHandler → (remote)portForwardHandler → portforwardHandler → view.ResolveTarget → StartPortForwardSession → RemoteSession.Run（試験は依存を注入）
- 重複・対応先: TestExecHandlerPreflightFailureStopsBeforeAWS（別実装経路、必要な対）
- 根拠・限界: 18-45: exec側と別本番関数の順序を守る。両者を1つのtableにまとめることは可能だがケース削除は不適切。
- 対応: 維持する。
- ケース別注意: 単一ケース。
- 実行: 1 leaf / 全件PASS。

### T073 — TestPortForwardHandlerSelectsSecondReplicaAndAllocatesLocalPort

- ソース: [internal/handler/portforward_test.go:48](../internal/handler/portforward_test.go#L48)
- 判定: **維持** / 状態: 対応済み
- 守る契約: 2番目replicaの正確なSSM target、local port割当1回、AWS-StartPortForwardingSessionというdocument literal、parameters、caller contextとpluginまでの順序
- 本番経路: portForwardHandler → view.ResolveTarget/Resolver → port allocation → command.StartPortForwardSession → RemoteSession.Run
- 重複・対応先: TestStartPortForwardSessionUsesCallerContextAndExactInput（API adapter境界）、TestRemotePortForwardHandlerBuildsExactParameters（同じoracle弱点）
- 根拠・限界: document期待値を本番定数から独立させ、定数誤変更を検出可能にした。他の選択/parameter/副作用境界も既存どおり保護。
- 対応: Q04/M04対応済み。維持。
- ケース別注意: 単独ケース。独立literalはAWS-StartPortForwardingSession。
- 実行: 1 leaf / 全件PASS。

### T074 — TestPortForwardHandlerExplicitPortSkipsAllocation

- ソース: [internal/handler/portforward_test.go:98](../internal/handler/portforward_test.go#L98)
- 判定: **維持** / 状態: 維持
- 守る契約: 明示local portがあればallocatorを呼ばず、値6000をそのままSSMパラメータへ送る。
- 本番経路: cmd/(remote)portforward → handler.(Remote)PortforwardHandler → (remote)portForwardHandler → portforwardHandler → view.ResolveTarget → StartPortForwardSession → RemoteSession.Run（試験は依存を注入）
- 重複・対応先: なし
- 根拠・限界: 98-114: 自動確保試験とは逆分岐。不要なsocket確保やユーザー値上書きを捕捉する独立保護。
- 対応: 維持する。
- ケース別注意: 単一ケース。
- 実行: 1 leaf / 全件PASS。

### T075 — TestPortForwardHandlerAllocationFailureDoesNotStartSession

- ソース: [internal/handler/portforward_test.go:117](../internal/handler/portforward_test.go#L117)
- 判定: **維持** / 状態: 維持
- 守る契約: allocator errorの原因保持、0/-1/65536の異常portを拒否し、どの失敗でもSSM StartSessionを開始しない。
- 本番経路: cmd/(remote)portforward → handler.(Remote)PortforwardHandler → (remote)portForwardHandler → portforwardHandler → view.ResolveTarget → StartPortForwardSession → RemoteSession.Run（試験は依存を注入）
- 重複・対応先: なし
- 根拠・限界: 117-154: 範囲外値は現在のproduction port.AvailablePortでは通常返らないが、net.Listenerやallocator境界の防御。全体は実際に起こるlisten/close errorと副作用停止を守るため維持。
- 対応: 維持する。
- ケース別注意: allocator error / zero port / negative port / port above maximum の4ケース。負数と上限超えは防御契約であり低頻度を理由に削除しない。
- 実行: 4 leaf / 全件PASS。

| 実行サブケース | 個別判定 |
| --- | --- |
| `allocator_error` | 維持 |
| `negative_port` | 維持 |
| `port_above_maximum` | 維持 |
| `zero_port` | 維持 |

### T076 — TestRemotePortForwardHandlerBuildsExactParameters

- ソース: [internal/handler/portforward_test.go:158](../internal/handler/portforward_test.go#L158)
- 判定: **維持** / 状態: 対応済み
- 守る契約: remote forwardがhost/remote port/local portを正しく渡し、AWS-StartPortForwardingSessionToRemoteHostというdocument literalを選ぶ
- 本番経路: remotePortForwardHandler → portforwardHandler → command.StartPortForwardSession
- 重複・対応先: TestPortForwardHandlerSelectsSecondReplicaAndAllocatesLocalPort（共通部分は同一handler、remote wrapper契約は独立）
- 根拠・限界: document期待値を本番定数から独立させ、定数誤変更を検出可能にした。明示port時のallocator禁止とparameter一致を維持。
- 対応: Q04/M04対応済み。維持。
- ケース別注意: 単独ケース。独立literalはAWS-StartPortForwardingSessionToRemoteHost。
- 実行: 1 leaf / 全件PASS。

### T077 — TestPortForwardHandlerPluginFailureTerminatesAfterPlugin

- ソース: [internal/handler/portforward_test.go:191](../internal/handler/portforward_test.go#L191)
- 判定: **維持** / 状態: 維持
- 守る契約: portforward経路でplugin失敗原因を返し、plugin実行後に作成したsession IDを1回terminateする。
- 本番経路: cmd/(remote)portforward → handler.(Remote)PortforwardHandler → (remote)portForwardHandler → portforwardHandler → view.ResolveTarget → StartPortForwardSession → RemoteSession.Run（試験は依存を注入）
- 重複・対応先: TestStartPortForwardSessionTerminatesExactSessionWhenPluginFails（統合元）; TestRemoteSessionTerminatesWhenPluginFails（純粋な契約）
- 根拠・限界: 191-210: portforward handler→constructor→Runの一貫した終了処理を守る。session単体だけではSSM callbackの誤配線を防げない。
- 対応: 維持。constructor側重複試験を統合する場合、ここへ元context/valueとinvocation全体のassertionを移す。
- ケース別注意: 単一ケース。
- 実行: 1 leaf / 全件PASS。

### T078 — TestPortForwardHandlerViewCancellationDoesNotAllocateOrStart

- ソース: [internal/handler/portforward_test.go:214](../internal/handler/portforward_test.go#L214)
- 判定: **維持** / 状態: 維持
- 守る契約: UI quit時は正常終了し、local port確保・SSM StartSession・plugin実行をしない。
- 本番経路: cmd/(remote)portforward → handler.(Remote)PortforwardHandler → (remote)portForwardHandler → portforwardHandler → view.ResolveTarget → StartPortForwardSession → RemoteSession.Run（試験は依存を注入）
- 重複・対応先: TestExecHandlerViewCancellationCreatesNoRemoteSession（別handler、副作用が異なる）
- 根拠・限界: 214-234: exec側と異なりport確保という副作用を持つ本番分岐を守る。view側のquit戻り値試験だけで代替できない。
- 対応: 維持する。
- ケース別注意: 単一ケース。
- 実行: 1 leaf / 全件PASS。

### T079 — TestReadInputFileStrict

- ソース: [internal/input/func_test.go:12](../internal/input/func_test.go#L12)
- 判定: **維持** / 状態: 維持
- 守る契約: 未知フィールド拒否と複数JSON文書拒否
- 本番経路: main→登録されたCLI.RunE→Resolve*→ReadInputFile/Validate*
- 重複・対応先: なし
- 根拠・限界: ReadInputFileのDisallowUnknownFieldsと2回目DecodeによるEOF確認を実ファイルで検証。encoding/json既定値では保証されないアプリの厳格読込契約。
- 対応: 維持
- ケース別注意: unknown field: typo commnadを拒否。two JSON documents: 連続2文書を拒否。どちらも異なる防御。
- 実行: 2 leaf / 全件PASS。

| 実行サブケース | 個別判定 |
| --- | --- |
| `two_JSON_documents` | 維持 |
| `unknown_field` | 維持 |

### T080 — TestMakeInputFileRefusesExistingFileUnlessForced

- ソース: [internal/input/func_test.go:47](../internal/input/func_test.go#L47)
- 判定: **維持** / 状態: 対応済み
- 守る契約: 非forceは既存bytesを保存しfs.ErrExist、forceは長い旧fileを短いJSONへ切り詰めてstrict decode可能な期待内容へ置換
- 本番経路: make-input-file → input.MakeInputFile / ReadInputFile
- 重複・対応先: TestNewRefusesExistingFileAndPreservesContents; TestNewForceReplacesExistingFile
- 根拠・限界: 旧366 bytesは新71 bytesより長いことも確認。期待bytes比較とstrict読み込みによりO_TRUNC欠落の末尾残存を検出。
- 対応: Q02/M02強化済み
- ケース別注意: サブテストなし
- 実行: 1 leaf / 全件PASS。

### T081 — TestInputFileUnicodeRoundTrip

- ソース: [internal/input/func_test.go:89](../internal/input/func_test.go#L89)
- 判定: **維持** / 状態: 維持
- 守る契約: 生成ファイルの0600権限、2-space整形・終端改行、Unicode入出力の無損失
- 本番経路: CLI make-input-file→MakeInputFile; CLI入力→ReadInputFile
- 重複・対応先: なし
- 根拠・限界: 独自document型なので本番ExecInput等のschemaは検査しないが、0600と末尾改行はこの関数群で唯一の保護。Unicode自体は標準JSONへ依存している。
- 対応: 削除しない。将来統合時は本番入力型へ移し、権限/改行のassertを残す。
- ケース別注意: サブテストなし
- 実行: 1 leaf / 全件PASS。

### T082 — TestResolveExecPrecedence

- ソース: [internal/input/resolve_test.go:11](../internal/input/resolve_test.go#L11)
- 判定: **維持** / 状態: 維持
- 守る契約: file値を明示commandとwait=0で上書きし、clusterをtrim
- 本番経路: exec.RunE→ResolveExec→ReadInputFile→normalizeExec→ValidateExec
- 重複・対応先: TestExecCommandExplicitFlagsOverrideFile（CLI側はChanged判定を追加検証）
- 根拠・限界: 値0のpointer overrideをnilの未指定と区別。CLIと完全重複ではなく、このテストにのみcluster trimのassertがある。
- 対応: 維持
- ケース別注意: サブテストなし
- 実行: 1 leaf / 全件PASS。

### T083 — TestResolveExecUsesDefaultWithoutFileOrOverride

- ソース: [internal/input/resolve_test.go:31](../internal/input/resolve_test.go#L31)
- 判定: **維持** / 状態: 維持
- 守る契約: 入力なしでcommand=sh、wait=0、ECS指定なし
- 本番経路: exec.RunE→ResolveExec→ReadInputFile→normalizeExec→ValidateExec
- 重複・対応先: なし
- 根拠・限界: Resolverがデフォルトの出所であり、CLI contextテストはrunner入力を比較しないため独立した保証。
- 対応: 維持
- ケース別注意: サブテストなし
- 実行: 1 leaf / 全件PASS。

### T084 — TestResolveExecUsesNormalizedFileValuesWithoutOverrides

- ソース: [internal/input/resolve_test.go:43](../internal/input/resolve_test.go#L43)
- 判定: **維持** / 状態: 維持
- 守る契約: override未指定でfileのcommand/waitを維持しcluster/service/commandをtrim
- 本番経路: exec.RunE→ResolveExec→ReadInputFile→normalizeExec→ValidateExec
- 重複・対応先: なし
- 根拠・限界: CLI execテストにないfileだけの分岐と正規化を具体値で検証。
- 対応: 維持
- ケース別注意: サブテストなし
- 実行: 1 leaf / 全件PASS。

### T085 — TestResolveExecPropagatesStrictInputFileError

- ソース: [internal/input/resolve_test.go:61](../internal/input/resolve_test.go#L61)
- 判定: **維持** / 状態: 維持
- 守る契約: 厳格読込エラーの操作名/未知フィールドを返し、入力はゼロ値
- 本番経路: exec.RunE→ResolveExec→ReadInputFile→normalizeExec→ValidateExec
- 重複・対応先: TestReadInputFileStrict/unknown field（下位parserのみ）
- 根拠・限界: 未知フィールド拒否は重なるがresolverがerrorを飲み込まないこととzero value返却を追加検証。
- 対応: 維持
- ケース別注意: サブテストなし
- 実行: 1 leaf / 全件PASS。

### T086 — TestResolvePortForwardAppliesExplicitOverridesAndNormalizes

- ソース: [internal/input/resolve_test.go:78](../internal/input/resolve_test.go#L78)
- 判定: **維持** / 状態: 維持
- 守る契約: target overrideのtrim、local空白override→空文字、cluster/service trim
- 本番経路: portforward.RunE→ResolvePortForward→ReadInputFile→normalizeECS/TrimSpace→ValidatePortForward
- 重複・対応先: TestPortforwardCommandPreservesExplicitEmptyLocalPort（CLI Changed判定）
- 根拠・限界: file localを空に戻す契約とtrimはCLIでの明示空文字とは異なる入力条件。
- 対応: 維持
- ケース別注意: サブテストなし
- 実行: 1 leaf / 全件PASS。

### T087 — TestResolvePortForwardUsesFileValuesAndPreservesDefaultLocalPort

- ソース: [internal/input/resolve_test.go:106](../internal/input/resolve_test.go#L106)
- 判定: **維持** / 状態: 維持
- 守る契約: file target/cluster/serviceをtrimし、local未指定は空文字
- 本番経路: portforward.RunE→ResolvePortForward→ReadInputFile→normalizeECS/TrimSpace→ValidatePortForward
- 重複・対応先: なし
- 根拠・限界: 自動ポート選択への入力契約。handlerのallocatorを直接試すテストとは接続点が異なる。
- 対応: 維持
- ケース別注意: サブテストなし
- 実行: 1 leaf / 全件PASS。

### T088 — TestResolvePortForwardDefaultFailsRequiredTargetValidation

- ソース: [internal/input/resolve_test.go:128](../internal/input/resolve_test.go#L128)
- 判定: **維持** / 状態: 維持
- 守る契約: 入力なしではtarget必須エラーとゼロ値を返す
- 本番経路: portforward.RunE→ResolvePortForward→ReadInputFile→normalizeECS/TrimSpace→ValidatePortForward
- 重複・対応先: TestValidatePortForward/missing target（下位validationのみ）
- 根拠・限界: default空入力がvalidationを経由すること、エラー時値の返却を保護。
- 対応: 維持
- ケース別注意: サブテストなし
- 実行: 1 leaf / 全件PASS。

### T089 — TestResolvePortForwardReturnsZeroValueOnValidationError

- ソース: [internal/input/resolve_test.go:138](../internal/input/resolve_test.go#L138)
- 判定: **維持** / 状態: 維持
- 守る契約: target abc/local 0の2つの検証エラーを集約しゼロ値を返す
- 本番経路: portforward.RunE→ResolvePortForward→ReadInputFile→normalizeECS/TrimSpace→ValidatePortForward
- 重複・対応先: TestResolvePortForwardReturnsZeroValueOnValidationError（旧名から移管済み）
- 根拠・限界: 下位validatorテストと同じ入力/エラー断片に加えresolverのゼロ値返却を保証。統合するならこの上位テストを残せる。
- 対応: 維持。LV08の下位重複テストを本関数へ統合済み。
- ケース別注意: サブテストなし
- 実行: 1 leaf / 全件PASS。

### T090 — TestResolveRemotePortForwardAppliesExplicitOverridesAndNormalizes

- ソース: [internal/input/resolve_test.go:159](../internal/input/resolve_test.go#L159)
- 判定: **維持** / 状態: 維持
- 守る契約: remote/local/hostの明示上書きと全文字列trim、空localへの戻し
- 本番経路: remoteportforward.RunE→ResolveRemotePortForward→ReadInputFile→normalizeECS/TrimSpace→ValidateRemotePortForward
- 重複・対応先: TestRemotePortforwardCommandAllExplicitFlagsOverrideFile（CLIフラグ配線）
- 根拠・限界: 本テストはTrimSpaceと空local、CLIは複数Flags().Changedの同時成立をそれぞれ保護。
- 対応: 維持
- ケース別注意: サブテストなし
- 実行: 1 leaf / 全件PASS。

### T091 — TestResolveRemotePortForwardUsesNormalizedFileValues

- ソース: [internal/input/resolve_test.go:191](../internal/input/resolve_test.go#L191)
- 判定: **維持** / 状態: 維持
- 守る契約: overrideなしでfile remote/local/host/cluster/serviceをtrimして採用
- 本番経路: remoteportforward.RunE→ResolveRemotePortForward→ReadInputFile→normalizeECS/TrimSpace→ValidateRemotePortForward
- 重複・対応先: なし
- 根拠・限界: fileのみの分岐と全フィールド正規化。
- 対応: 維持
- ケース別注意: サブテストなし
- 実行: 1 leaf / 全件PASS。

### T092 — TestResolveRemotePortForwardDefaultJoinsRequiredValidationErrors

- ソース: [internal/input/resolve_test.go:216](../internal/input/resolve_test.go#L216)
- 判定: **維持** / 状態: 維持
- 守る契約: 入力なしはremote port/host両必須エラーとゼロ値
- 本番経路: remoteportforward.RunE→ResolveRemotePortForward→ReadInputFile→normalizeECS/TrimSpace→ValidateRemotePortForward
- 重複・対応先: なし
- 根拠・限界: remoteのrequired=true配線とhostチェックを同時保護。ValidateRemotePortForwardにremote空の専用ケースはなく独自価値がある。
- 対応: 維持
- ケース別注意: サブテストなし
- 実行: 1 leaf / 全件PASS。

### T093 — TestResolveRemotePortForwardReturnsZeroValueOnValidationError

- ソース: [internal/input/resolve_test.go:231](../internal/input/resolve_test.go#L231)
- 判定: **維持** / 状態: 維持
- 守る契約: remote abc/local 0/空白hostの3エラーとゼロ値
- 本番経路: remoteportforward.RunE→ResolveRemotePortForward→ReadInputFile→normalizeECS/TrimSpace→ValidateRemotePortForward
- 重複・対応先: TestValidateRemotePortForward/all invalid（空白hostと空hostの差あり）
- 根拠・限界: 集約は重なるがresolverのtrim/validation/zero valueを一体で保護。
- 対応: 維持
- ケース別注意: サブテストなし
- 実行: 1 leaf / 全件PASS。

### T094 — TestValidatePortForward

- ソース: [internal/input/validate_test.go:8](../internal/input/validate_test.go#L8)
- 判定: **維持** / 状態: 維持
- 守る契約: ASCII数字のみ、必須target、範囲1..65535、local省略、複数の不正表現
- 本番経路: portforward.RunE→ResolvePortForward→ReadInputFile→normalizeECS/TrimSpace→ValidatePortForward
- 重複・対応先: なし
- 根拠・限界: 範囲下限/上限、ParseUint overflow、optional localは異なる契約。ほぼ同じ分岐でも安価なboundary例を機械的に削る根拠は弱い。
- 対応: 維持
- ケース別注意: 14件: missing target; non-decimal target; target above range; target below range; negative-sign target; positive-sign target; malformed signed target; space-padded target; fractional target; digit-only overflow; local below range; non-decimal local; lower bound with default local; upper bounds。space-padded targetは本番Resolveが先にtrimするためCLIでの拒否を意味しない。malformed signed target (--1) はnegative-sign (-1)と先頭文字判定が同じで限界効用は小さいが低コスト。
- 実行: 14 leaf / 全件PASS。

| 実行サブケース | 個別判定 |
| --- | --- |
| `digit-only_overflow` | 維持 |
| `fractional_target` | 維持 |
| `local_below_range` | 維持 |
| `lower_bound_with_default_local` | 維持 |
| `malformed_signed_target` | 維持 |
| `missing_target` | 維持 |
| `negative-sign_target` | 維持 |
| `non-decimal_local` | 維持 |
| `non-decimal_target` | 維持 |
| `positive-sign_target` | 維持 |
| `space-padded_target` | 維持 |
| `target_above_range` | 維持 |
| `target_below_range` | 維持 |
| `upper_bounds` | 維持 |

### T096 — TestValidateExecRejectsNegativeWaitAndBlankCommand

- ソース: [internal/input/validate_test.go:100](../internal/input/validate_test.go#L100)
- 判定: **維持** / 状態: 維持
- 守る契約: 空白command/負waitの両エラーを集約
- 本番経路: exec.RunE→ResolveExec→ReadInputFile→normalizeExec→ValidateExec
- 重複・対応先: TestExecCommandInvalidInputDoesNotInvokeRunner（CLI→resolverの同入力）
- 根拠・限界: 単独validatorの集約保証。CLIに同じ2断片があるため重なりはあるが、エラー診断の局所化とコストは良好。
- 対応: 維持
- ケース別注意: サブテストなし
- 実行: 1 leaf / 全件PASS。

### T097 — TestValidateExecAcceptsCommandAndNonNegativeWait

- ソース: [internal/input/validate_test.go:112](../internal/input/validate_test.go#L112)
- 判定: **維持** / 状態: 維持
- 守る契約: 非空commandとwait=0を受理
- 本番経路: exec.RunE→ResolveExec→ReadInputFile→normalizeExec→ValidateExec
- 重複・対応先: ResolveExecの正常系と一部重なるが、前後空白の直接入力はここでのみ確認
- 根拠・限界: Cmd=" sh "を直接ValidateExecへ渡し受理を確認。resolverは先にTrimSpaceするため同じ入力をvalidatorへ渡さず、この直接入力契約を代替しない。
- 対応: 維持。将来validatorをtable化する場合は、この前後空白の正常ケースを移植してから統合。
- ケース別注意: サブテストなし
- 実行: 1 leaf / 全件PASS。

### T098 — TestValidateRemotePortForward

- ソース: [internal/input/validate_test.go:118](../internal/input/validate_test.go#L118)
- 判定: **維持** / 状態: 維持
- 守る契約: remote/localの上下限と省略、空白host拒否、remote/local/host全エラー集約
- 本番経路: remoteportforward.RunE→ResolveRemotePortForward→ReadInputFile→normalizeECS/TrimSpace→ValidateRemotePortForward
- 重複・対応先: TestValidatePortForwardの数値境界; TestResolveRemotePortForwardReturnsZeroValueOnValidationErrorの集約
- 根拠・限界: 数値parserは共通でもremoteフィールドとラベルの配線が別。全体を重複扱いすべきでない。
- 対応: 維持
- ケース別注意: 4件: valid lower bounds with default local; valid upper bounds; blank host; all invalid。all invalidはresolverの3断片確認にほぼ包含されるためケース単位統合余地あり。
- 実行: 4 leaf / 全件PASS。

| 実行サブケース | 個別判定 |
| --- | --- |
| `all_invalid` | 維持 |
| `blank_host` | 維持 |
| `valid_lower_bounds_with_default_local` | 維持 |
| `valid_upper_bounds` | 維持 |

### T099 — TestItemFilterValueUsesLabel

- ソース: [internal/listview/simple_test.go:11](../internal/listview/simple_test.go#L11)
- 判定: **維持** / 状態: 対応済み
- 守る契約: item.FilterValueが識別用Valueではなく表示Labelをフィルタ対象として返す
- 本番経路: RenderOptions → list item → FilterValue
- 重複・対応先: 既存重複はLV01〜LV05で統合済み
- 根拠・限界: テスト自身のstruct代入を確認していたValue保持assertionだけを削除。FilterValue呼出のassertionは維持。
- 対応: 対応済み。保護を維持。
- ケース別注意: 単一ケース。FilterValueのLabelを確認。
- 実行: 1 leaf / 全件PASS。

### T100 — TestRenderOptionsReturnsNoItemsError

- ソース: [internal/listview/simple_test.go:23](../internal/listview/simple_test.go#L23)
- 判定: **維持** / 状態: 維持
- 守る契約: 空optionsならTUIを起動せず*NoItemsErrorを返しTitleを保持する。
- 本番経路: RenderOptions(empty)。現行唯一の本番呼び出しは handler.productionDependencies.choose → ResolveTarget.chooseOption で、chooseOptionが先に空候補を拒否する。
- 重複・対応先: TestResolveTargetRejectsEmptyResourcesBeforeChooser（同じユーザー状況だがguardは別）
- 根拠・限界: 現CLIはchooseOptionで空候補を先に弾くため本番の主要経路ではないが、RenderOptionsの明示的な空入力契約を安価に守る唯一の直接テスト。削除効果は小さい。非空UI経路の代替証拠にはならない。
- 対応: 防御契約として維持。非空UI・取消・フィルタの保証を補完。
- ケース別注意: サブテストなし。戻りvalue/quitとError()文面はassertせず、error型とTitleだけ。
- 実行: 1 leaf / 全件PASS。

### T101 — TestModelEnterSelectsValueWhenLabelsAreDuplicated

- ソース: [internal/listview/simple_test.go:40](../internal/listview/simple_test.go#L40)
- 判定: **維持** / 状態: 維持
- 守る契約: 表示labelが同じでも選択中の2番目itemのvalueをEnterで返し、tea.Quitを発行する。
- 本番経路: RenderOptions → tea.Program → model.Update(Enter)
- 重複・対応先: TestResolveTargetSelectsExactReadyTaskARN は chooser をfakeするため本テストとは実行層が別
- 根拠・限界: simple_test.go:46-63で実bubbles modelと実KeyPressMsgを使いvalueとQuitMsgを検証。モデルを手組みするためRenderOptionsのitems変換・Program起動・戻り値回収は保証しないが、Enter選択の中心契約は有効。service ARN fixtureもUIがopaque stringとして扱う目的では妥当。
- 対応: 維持。非空RenderOptionsの配線を通す検証とq/Ctrl+C/フィルタ状態の契約を別途補う。
- ケース別注意: サブテストなし。重複label＋2番目をSelect(1)したEnter操作。
- 実行: 1 leaf / 全件PASS。

### T102 — TestInvocationArgumentsMatchAWSCLIContract

- ソース: [internal/session_manager/session_test.go:126](../internal/session_manager/session_test.go#L126)
- 判定: **維持** / 状態: 維持
- 守る契約: plugin起動argvの6要素・順序・Region/StartSession/profile/endpoint、およびsession JSONのSessionId/StreamUrl/TokenValueとrequest JSONのTargetを独立の期待値で確認。
- 本番経路: cmd exec/portforward/remoteportforward → handler → RemoteSession.Run → Runner.Run → Invocation.arguments（session.go:28,141）
- 重複・対応先: TestRunnerRunPassesSixArgumentsAndAssignsStreams（伝送先が異なる補完関係）
- 根拠・限界: session_test.go:141-167 は値をJSON decodeして独立のmapと比較するため、JSONキー・引数位置の変更を検知する。Runner試験の期待値はarguments()自身なので、本試験を消すと共通変換バグを見逃す。
- 対応: 維持。外部AWS pluginを実行している試験ではなく、リポジトリが期待する起動契約の固定試験として扱う。
- ケース別注意: 単独ケース。
- 実行: 1 leaf / 全件PASS。

### T103 — TestInvocationArgumentsAllowEmptyProfileAndEndpoint

- ソース: [internal/session_manager/session_test.go:171](../internal/session_manager/session_test.go#L171)
- 判定: **維持** / 状態: 維持
- 守る契約: profile/endpoint未指定でも起動引数の空文字スロットを保持し、検証エラーにしない。
- 本番経路: cmd exec/portforward/remoteportforward → handler → RemoteSession.Run → Runner.Run → Invocation.arguments（session.go:28,141）
- 重複・対応先: TestPreflightSuccessUsesExactVersionCommandAndEnvironment/unset values stay empty（環境→RunnerとRunner→argvの境界が異なる）
- 根拠・限界: session_test.go:172-177 はarguments()側の任意項目契約を守る。preflight試験はRunnerのフィールドまでであり、argv生成で空値を拒否する退行は本試験だけが直接捕捉する。
- 対応: 維持。
- ケース別注意: 単独ケース。
- 実行: 1 leaf / 全件PASS。

### T104 — TestInvocationArgumentsRejectBlankRequiredFields

- ソース: [internal/session_manager/session_test.go:181](../internal/session_manager/session_test.go#L181)
- 判定: **維持** / 状態: 維持
- 守る契約: session ID・stream URL・token・Region・Targetの各空文字/空白値を拒否し、argvをnilにする。
- 本番経路: cmd exec/portforward/remoteportforward → handler → RemoteSession.Run → Runner.Run → Invocation.arguments（session.go:28,141）
- 重複・対応先: TestRunnerValidationErrorDoesNotStartProcess（検証網羅と副作用禁止の補完）
- 根拠・限界: session_test.go:187-196の10ケースは独立した必須項目を変異させ、160-165でエラーとnil argvを確認。入力の10方向をRunner subprocessで重く繰り返さない分担は妥当。
- 対応: 維持。
- ケース別注意: 10ケース: empty session ID / whitespace session ID / empty stream URL / whitespace stream URL / empty token / whitespace token / empty region / whitespace region / empty target / whitespace target。全て維持。
- 実行: 10 leaf / 全件PASS。

| 実行サブケース | 個別判定 |
| --- | --- |
| `empty_region` | 維持 |
| `empty_session_ID` | 維持 |
| `empty_stream_URL` | 維持 |
| `empty_target` | 維持 |
| `empty_token` | 維持 |
| `whitespace_region` | 維持 |
| `whitespace_session_ID` | 維持 |
| `whitespace_stream_URL` | 維持 |
| `whitespace_target` | 維持 |
| `whitespace_token` | 維持 |

### T105 — TestPreflightReportsMissingPlugin

- ソース: [internal/session_manager/session_test.go:215](../internal/session_manager/session_test.go#L215)
- 判定: **維持** / 状態: 維持
- 守る契約: 固定コマンド名を探索し、plugin未検出なら起動処理を呼ばず、元エラーとインストール案内を返す。
- 本番経路: 各handler → productionDependencies.preflight（dependencies.go:47）→ Preflight（session.go:84）→ preflight（session.go:94）
- 重複・対応先: なし
- 根拠・限界: session_test.go:218-224 は検索名と後続呼出禁止、185-189はerrors.Isと案内を確認。OS探索そのものはfakeだがエラー整形・短絡の責務を検証している。
- 対応: 維持。
- ケース別注意: 単独ケース。
- 実行: 1 leaf / 全件PASS。

### T106 — TestPreflightReportsVersionFailureWithOutput

- ソース: [internal/session_manager/session_test.go:237](../internal/session_manager/session_test.go#L237)
- 判定: **維持** / 状態: 維持
- 守る契約: --version失敗の原因をerrors.Isで保持し、ユーザーにコマンドとプロセス出力を提示する。
- 本番経路: 各handler → productionDependencies.preflight（dependencies.go:47）→ Preflight（session.go:84）→ preflight（session.go:94）
- 重複・対応先: なし
- 根拠・限界: session_test.go:243のsentinel/出力を205-210で独立に照合。Runnerの実プロセスエラー試験とはpreflight時診断出力を含む点が異なる。
- 対応: 維持。
- ケース別注意: 単独ケース。
- 実行: 1 leaf / 全件PASS。

### T107 — TestPreflightRejectsEmptyVersionOutput

- ソース: [internal/session_manager/session_test.go:258](../internal/session_manager/session_test.go#L258)
- 判定: **維持** / 状態: 維持
- 守る契約: --versionが成功終了でも空白だけの出力ならplugin準備成功にしない。
- 本番経路: 各handler → productionDependencies.preflight（dependencies.go:47）→ Preflight（session.go:84）→ preflight（session.go:94）
- 重複・対応先: なし
- 根拠・限界: session.go:126-127の独立分岐を空白/改行/タブで確認。単なるerror有無ではなくempty output文脈も確認している。
- 対応: 維持。
- ケース別注意: 単独ケース。
- 実行: 1 leaf / 全件PASS。

### T108 — TestPreflightDeadlineIsDiscoverable

- ソース: [internal/session_manager/session_test.go:273](../internal/session_manager/session_test.go#L273)
- 判定: **維持** / 状態: 対応済み
- 守る契約: public Preflightが実plugin helperの起動後、親の実deadlineまたは本番3秒timeoutをcontext.DeadlineExceededとして返し、stdout/stderrを継承する子孫が生きていても終了する。process ExitErrorと操作文脈も保持。
- 本番経路: Preflight → exec.LookPath → exec.CommandContext(--version) → CombinedOutput → context timeout / WaitDelay
- 重複・対応先: TestPreflightParentCancellationIsDiscoverable（DeadlineExceeded/Canceledの異なる原因なので統合しても両ケースを保持）
- 根拠・限界: fakeの協調終了を使わず、helperの子孫が出力pipeを継承したmarkerで同期。親deadline caseは期限到達後1秒、内部timeout caseは起動同期後4秒を独立上限とし、終了待ちの欠落を局所FAILにする。
- 対応: Q10対応完了。WaitDelay=0、preflight timeout=30秒の各変異を検出。
- ケース別注意: caller deadline / preflight timeout の2case。子孫はrelease fileで終了し、独立10秒上限とcleanup待ちを備える。
- 実行: 2 leaf / 全件PASS。

| 実行サブケース | 個別判定 |
| --- | --- |
| `caller_deadline` | 維持 |
| `preflight_timeout` | 維持 |

### T109 — TestPreflightParentCancellationIsDiscoverable

- ソース: [internal/session_manager/session_test.go:286](../internal/session_manager/session_test.go#L286)
- 判定: **維持** / 状態: 対応済み
- 守る契約: public Preflightの実process起動と子孫pipe継承後のcaller cancelがcontext.Canceledとして戻り、実ExitErrorと操作文脈を保持して1秒以内に終了する。
- 本番経路: Preflight → exec.LookPath → exec.CommandContext(--version) → CombinedOutput → caller cancel / WaitDelay
- 重複・対応先: TestPreflightDeadlineIsDiscoverable（エラー原因が異なる必要な補完）
- 根拠・限界: 修正前はpipe-holder終了までCombinedOutputが戻らず1秒上限でFAIL。子孫を存続させたまま呼出が戻ることを確認し、最後にrelease fileでcleanup。
- 対応: Q10対応完了。public factoryのWaitDelay=200msで終了を制限。
- ケース別注意: サブケースなし。markerで実起動とpipe継承を同期してからcancel。
- 実行: 1 leaf / 全件PASS。

### T110 — TestPreflightSuccessUsesExactVersionCommandAndEnvironment

- ソース: [internal/session_manager/session_test.go:292](../internal/session_manager/session_test.go#L292)
- 判定: **維持** / 状態: 対応済み
- 守る契約: 本番名session-manager-pluginを専用PATHから発見し、実processへ--versionとAWS環境変数を渡す。primary/fallback/emptyの選択結果を返却Runnerへ保持。
- 本番経路: Preflight → exec.LookPath → exec.CommandContext(--version) → CombinedOutput → Runner construction
- 重複・対応先: TestInvocationArgumentsAllowEmptyProfileAndEndpoint（設定入力とargv出力の異なる境界）
- 根拠・限界: private dependenciesのfakeを廃止。test executableを独立literal名でsymlinkし、helperが保存した実argv/envと返却Runnerを独立期待値と比較。PATHはfixture専用なので実インストール済みpluginを呼ばない。
- 対応: Q10対応完了。既存3caseを実process経路へ移植。
- ケース別注意: service-specific values win / fallback values are used / unset values stay empty。helperのrace exit sleepのみ無効化し、終了遅延の無関係な揺れを除外。
- 実行: 3 leaf / 全件PASS。

| 実行サブケース | 個別判定 |
| --- | --- |
| `fallback_values_are_used` | 維持 |
| `service-specific_values_win` | 維持 |
| `unset_values_stay_empty` | 維持 |

### T111 — TestRunnerValidationErrorDoesNotStartProcess

- ソース: [internal/session_manager/session_test.go:360](../internal/session_manager/session_test.go#L360)
- 判定: **維持** / 状態: 維持
- 守る契約: 必須tokenが空白ならRunner.Runが実子processを起動せずエラーを返す。
- 本番経路: 各handler → RemoteSession.Run → Runner.Run（session.go:141）→ exec.CommandContext と cmd.Run
- 重複・対応先: TestInvocationArgumentsRejectBlankRequiredFields（検証と起動副作用禁止を別々に保護）
- 根拠・限界: session_test.go:365-376のstartedファイル不在が実exec境界の副作用禁止を確認。単なるarguments()の単体テストでは起動順序の退行を防げない。
- 対応: 維持。
- ケース別注意: 単独ケース。
- 実行: 1 leaf / 全件PASS。

### T112 — TestRunnerRunPassesSixArgumentsAndAssignsStreams

- ソース: [internal/session_manager/session_test.go:380](../internal/session_manager/session_test.go#L380)
- 判定: **維持** / 状態: 維持
- 守る契約: 実子processに6 argvを引き渡し、os.Stdin/Stdout/Stderrをそれぞれ接続する。stdinの実読取とstdout/stderrの実出力を確認。
- 本番経路: 各handler → RemoteSession.Run → Runner.Run（session.go:141）→ exec.CommandContext と cmd.Run
- 重複・対応先: TestInvocationArgumentsMatchAWSCLIContract（エンコード契約とexec境界の補完）
- 根拠・限界: session_test.go:404-420はchildが保存したargvとarguments()を再計算した値を比較するため、エンコード仕様の独立oracleではない。ただし82行の独立JSON契約試験が補完し、407-410は実stdioを検証するので全体は高価値。
- 対応: 維持。expectedを本番arguments()から作る制約を明記し、独立契約試験とセットで保持。将来その試験を削るならここへ独立expectedを移す。
- ケース別注意: 単独ケース。
- 実行: 1 leaf / 全件PASS。

### T113 — TestRunnerRunWrapsProcessError

- ソース: [internal/session_manager/session_test.go:430](../internal/session_manager/session_test.go#L430)
- 判定: **維持** / 状態: 維持
- 守る契約: 実helperの非zero終了をrun session-manager-plugin文脈付きで返し、exec.ExitError型をerrors.Asで保持する。
- 本番経路: 各handler → RemoteSession.Run → Runner.Run（session.go:141）→ exec.CommandContext と cmd.Run
- 重複・対応先: TestRunnerRunPreservesCancellation/Deadlineの共通assertWrappedProcessError（通常の非zeroとcontextによるkillは別経路）
- 根拠・限界: session_test.go:435-443は実exit 17のprocessを通す。キャンセル試験で省略するとcontextなしの通常exitエラー経路が未検証になる。exit code自体はassertしないが、目的の型/文脈保護には足りる。
- 対応: 維持。
- ケース別注意: 単独ケース。
- 実行: 1 leaf / 全件PASS。

### T114 — TestRunnerRunPreservesCancellation

- ソース: [internal/session_manager/session_test.go:447](../internal/session_manager/session_test.go#L447)
- 判定: **維持** / 状態: 維持
- 守る契約: 子processが実際に起動した後にcancelし、Runnerが戻ってCanceledとexec.ExitErrorを両方保持する。
- 本番経路: 各handler → RemoteSession.Run → Runner.Run（session.go:141）→ exec.CommandContext と cmd.Run
- 重複・対応先: TestRunnerRunPreservesDeadline（共通経路だが異なるcontext.Errの保存）
- 根拠・限界: session_test.go:490-512の起動ファイル同期後にcancelするため「起動前cancelだけ」の空振りを避けている。5秒の待ち上限もあり、実exec.CommandContextによるkillまで確認。
- 対応: 維持。
- ケース別注意: 単独ケース。
- 実行: 1 leaf / 全件PASS。

### T115 — TestRunnerRunPreservesDeadline

- ソース: [internal/session_manager/session_test.go:463](../internal/session_manager/session_test.go#L463)
- 判定: **維持** / 状態: 維持
- 守る契約: 起動済みprocessにDoneを通知し、DeadlineExceededとexec.ExitErrorを両方保持する。
- 本番経路: 各handler → RemoteSession.Run → Runner.Run（session.go:141）→ exec.CommandContext と cmd.Run
- 重複・対応先: TestRunnerRunPreservesCancellation（同じkill処理、異なるdeadlineエラーidentity）
- 根拠・限界: manualDeadlineContext（session_test.go:550-578）は時刻由来のdeadlineではなくDone/Errを手動制御する。実process終了とDeadlineExceededの透過は検証でき、flakyな壁時計競争を避ける意図も妥当。実WithTimeoutのdeadline設定/伝播まで守ると解釈しない。
- 対応: 維持。実期限到達の保証ではなくDeadlineExceededの保存とprocess終了の試験として扱う。
- ケース別注意: 単独ケース。
- 実行: 1 leaf / 全件PASS。

### T116 — TestEligibleTaskRequiresReadyTaskAndContainer

- ソース: [internal/target/eligibility_test.go:11](../internal/target/eligibility_test.go#L11)
- 判定: **維持** / 状態: 対応済み
- 守る契約: taskのRUNNING/Exec有効/ARN、containerのname/RUNNING/runtime、ExecuteCommandAgentのRUNNINGを要求する。Task.LastStatus/TaskArn、Container.Name/LastStatus/RuntimeId、ManagedAgent.LastStatusが各々単独nilでもpanicせず不適格とする。
- 本番経路: Resolver.EligibleTasks → IsEligibleTask → EligibleContainers → hasRunningExecuteCommandAgent
- 重複・対応先: TestEligibleContainersReturnsOnlyReadyContainers（container条件の一部）
- 根拠・限界: 18case。各nil caseは他の全fieldをreadyに保つので先行条件による短絡がなく、6つの安全なpointer読取を直接保護する。各aws.ToStringをunsafe dereferenceにする6変異を全て対応caseで検出。
- 対応: Q09対応完了。維持。
- ケース別注意: 既存12case維持。旧nil pointers 1caseをnil task status / nil task ARN / nil container name / nil container status / nil runtime ID / nil execute command agent statusの6caseに置換。
- 実行: 18 leaf / 全件PASS。

| 実行サブケース | 個別判定 |
| --- | --- |
| `container_status_padded_with_whitespace` | 維持 |
| `container_stopped` | 維持 |
| `execute_command_agent_pending` | 維持 |
| `execute_command_agent_status_padded_with_whitespace` | 維持 |
| `execute_command_disabled` | 維持 |
| `fully_ready_task` | 維持 |
| `missing_container_name` | 維持 |
| `missing_execute_command_agent` | 維持 |
| `missing_runtime_ID` | 維持 |
| `missing_task_ARN` | 維持 |
| `nil_container_name` | 維持 |
| `nil_container_status` | 維持 |
| `nil_execute_command_agent_status` | 維持 |
| `nil_runtime_ID` | 維持 |
| `nil_task_ARN` | 維持 |
| `nil_task_status` | 維持 |
| `task_pending` | 維持 |
| `task_status_padded_with_whitespace` | 維持 |

### T117 — TestEligibleContainersReturnsOnlyReadyContainers

- ソース: [internal/target/eligibility_test.go:141](../internal/target/eligibility_test.go#L141)
- 判定: **維持** / 状態: 対応済み
- 守る契約: name欠損、STOPPED、runtime欠損、agent不在/未readyが混じる応答からready containerのみを値ごと保持して返す。
- 本番経路: Resolver/ResolveTarget → EligibleContainers
- 重複・対応先: TestEligibleTaskRequiresReadyTaskAndContainer（predicate）
- 根拠・限界: mixed responseのフィルタ契約を維持。Name=nilですぐ短絡する総崩れnilPointers fixtureはmissingNameと重複するので削除。各nil安全性はTestEligibleTaskRequiresReadyTaskAndContainerの6caseで直接保護。
- 対応: Q09対応完了。維持。
- ケース別注意: 単一mixed response case。ready1件＋不適格5件。
- 実行: 1 leaf / 全件PASS。

### T118 — TestClusterName

- ソース: [internal/target/identifier_test.go:8](../internal/target/identifier_test.go#L8)
- 判定: **維持** / 状態: 維持
- 守る契約: 短名/前後空白/通常ARN/GovCloud ARNを正規化し、不正なresource形状、service/partition/region/account ID、空白を拒否。
- 本番経路: handler.Exec/PortForward/RemotePortForward → view.ResolveTarget → target.Resolver → ClusterName; pkg/command.StartExecSession → ClusterName
- 重複・対応先: TestTaskID（共通parseECSARNの拒否条件）
- 根拠・限界: 短名とARNの両方を本番入口で受ける。両公開関数で共有parserの配線を確認するため、共通拒否ケースを単純な重複とはしない。
- 対応: 維持
- ケース別注意: 成功5件: short name/trimmed short name/ECS ARN/trimmed ECS ARN/GovCloud ARN。拒否15件: empty/whitespace/slash in short name/malformed ARN/non-ECS ARN/empty partition/empty region/empty account/short account/non-numeric account/wrong resource type/missing path/extra path/empty final segment/whitespace around resource segment。失敗時は元入力（空ならempty説明）を保持。
- 実行: 20 leaf / 全件PASS。

| 実行サブケース | 個別判定 |
| --- | --- |
| `accepts_short_names_and_ECS_ARNs/ECS_ARN` | 維持 |
| `accepts_short_names_and_ECS_ARNs/GovCloud_ARN` | 維持 |
| `accepts_short_names_and_ECS_ARNs/short_name` | 維持 |
| `accepts_short_names_and_ECS_ARNs/trimmed_ECS_ARN` | 維持 |
| `accepts_short_names_and_ECS_ARNs/trimmed_short_name` | 維持 |
| `rejects_malformed_identifiers/empty` | 維持 |
| `rejects_malformed_identifiers/empty_ARN_account` | 維持 |
| `rejects_malformed_identifiers/empty_ARN_final_segment` | 維持 |
| `rejects_malformed_identifiers/empty_ARN_partition` | 維持 |
| `rejects_malformed_identifiers/empty_ARN_region` | 維持 |
| `rejects_malformed_identifiers/extra_ARN_path_component` | 維持 |
| `rejects_malformed_identifiers/malformed_ARN` | 維持 |
| `rejects_malformed_identifiers/missing_ARN_path_component` | 維持 |
| `rejects_malformed_identifiers/non-ECS_ARN` | 維持 |
| `rejects_malformed_identifiers/non-numeric_ARN_account` | 維持 |
| `rejects_malformed_identifiers/short_ARN_account` | 維持 |
| `rejects_malformed_identifiers/slash_in_short_name` | 維持 |
| `rejects_malformed_identifiers/whitespace` | 維持 |
| `rejects_malformed_identifiers/whitespace_around_ARN_resource_segment` | 維持 |
| `rejects_malformed_identifiers/wrong_ARN_resource_type` | 維持 |

### T119 — TestTaskID

- ソース: [internal/target/identifier_test.go:86](../internal/target/identifier_test.go#L86)
- 判定: **維持** / 状態: 維持
- 守る契約: task/<id>・task/<cluster>/<id>・短長ARN・China ARNからID抽出。無prefix、誤resource、空segment、余分segment、ARN metadata不正を拒否。
- 本番経路: handler.Exec/PortForward/RemotePortForward → view.ResolveTarget → target.Resolver → TaskID; pkg/command.StartExecSession → TaskID
- 重複・対応先: TestClusterName（共通parseECSARNの拒否条件）
- 根拠・限界: TaskIDはSSM target文字列とtask choice identityに影響するため独立した境界テストが必要。
- 対応: 維持
- ケース別注意: 成功6件: short resource/long resource/short ARN/long ARN/trimmed long ARN/China ARN。拒否22件: empty/whitespace/missing resource prefix/missing resource path/wrong resource type/empty short final/empty long final/missing long middle/extra resource component/cluster segment whitespace/task ID whitespace/malformed ARN/non-ECS ARN/empty partition/empty region/empty account/short account/non-numeric account/wrong ARN resource type/missing ARN path/extra ARN path/empty ARN final。
- 実行: 28 leaf / 全件PASS。

| 実行サブケース | 個別判定 |
| --- | --- |
| `accepts_short_and_long_ECS_task_identifiers/China_ARN` | 維持 |
| `accepts_short_and_long_ECS_task_identifiers/long_ARN` | 維持 |
| `accepts_short_and_long_ECS_task_identifiers/long_resource` | 維持 |
| `accepts_short_and_long_ECS_task_identifiers/short_ARN` | 維持 |
| `accepts_short_and_long_ECS_task_identifiers/short_resource` | 維持 |
| `accepts_short_and_long_ECS_task_identifiers/trimmed_long_ARN` | 維持 |
| `rejects_malformed_identifiers/empty` | 維持 |
| `rejects_malformed_identifiers/empty_ARN_account` | 維持 |
| `rejects_malformed_identifiers/empty_ARN_final_segment` | 維持 |
| `rejects_malformed_identifiers/empty_ARN_partition` | 維持 |
| `rejects_malformed_identifiers/empty_ARN_region` | 維持 |
| `rejects_malformed_identifiers/empty_long_final_segment` | 維持 |
| `rejects_malformed_identifiers/empty_short_final_segment` | 維持 |
| `rejects_malformed_identifiers/extra_ARN_path_component` | 維持 |
| `rejects_malformed_identifiers/extra_resource_path_component` | 維持 |
| `rejects_malformed_identifiers/malformed_ARN` | 維持 |
| `rejects_malformed_identifiers/missing_ARN_path_component` | 維持 |
| `rejects_malformed_identifiers/missing_long_middle_segment` | 維持 |
| `rejects_malformed_identifiers/missing_resource_path` | 維持 |
| `rejects_malformed_identifiers/missing_resource_prefix` | 維持 |
| `rejects_malformed_identifiers/non-ECS_ARN` | 維持 |
| `rejects_malformed_identifiers/non-numeric_ARN_account` | 維持 |
| `rejects_malformed_identifiers/short_ARN_account` | 維持 |
| `rejects_malformed_identifiers/whitespace` | 維持 |
| `rejects_malformed_identifiers/whitespace_around_cluster_segment` | 維持 |
| `rejects_malformed_identifiers/whitespace_around_task_ID_segment` | 維持 |
| `rejects_malformed_identifiers/wrong_ARN_resource_type` | 維持 |
| `rejects_malformed_identifiers/wrong_resource_type` | 維持 |

### T120 — TestResolverClustersPaginatesInOrder

- ソース: [internal/target/resolver_test.go:133](../internal/target/resolver_test.go#L133)
- 判定: **維持** / 状態: 維持
- 守る契約: ListClusters全ページを応答順に連結し、初回nil token→次token、context markerを各呼び出しへ渡す。
- 本番経路: handler.Exec/PortForward/RemotePortForward → view.ResolveTarget → target.Resolver.Clusters
- 重複・対応先: なし
- 根拠・限界: 152-164行で順序・回数・token・context伝搬を独立に検証。本番クラスタ選択の候補欠落を防ぐ。
- 対応: 維持
- ケース別注意:
- 実行: 1 leaf / 全件PASS。

### T121 — TestResolverClustersRejectsRepeatedPaginationTokens

- ソース: [internal/target/resolver_test.go:167](../internal/target/resolver_test.go#L167)
- 判定: **維持** / 状態: 維持
- 守る契約: 同一tokenの反復とA→B→Aの循環で有限回のAPI呼び出し後に原因token付きerror。
- 本番経路: handler.Exec/PortForward/RemotePortForward → view.ResolveTarget → target.Resolver.Clusters
- 重複・対応先: TestResolverEligibleTasksRejectsRepeatedPaginationTokens（別APIの同種ループ）
- 根拠・限界: 2種類は直前tokenだけ比較する不完全実装と無検査の両方を捕まえる。fake回数上限のエラーを成功条件にせず、resolver自身の検知か確認。
- 対応: 維持
- ケース別注意: same token repeats (2 calls) / tokens form a cycle (3 calls)。ListTasksとは別ループなので維持。
- 実行: 2 leaf / 全件PASS。

| 実行サブケース | 個別判定 |
| --- | --- |
| `same_token_repeats` | 維持 |
| `tokens_form_a_cycle` | 維持 |

### T122 — TestResolverEligibleTasksPaginatesAndOrderedDeduplicates

- ソース: [internal/target/resolver_test.go:213](../internal/target/resolver_test.go#L213)
- 判定: **維持** / 状態: 維持
- 守る契約: ListTasksページ跨ぎの重複ARNを初出順に除き、Cluster・RUNNING・service有無・tokenを正しく送り、DescribeTasksへ重複なしで渡す。
- 本番経路: handler.Exec/PortForward/RemotePortForward → view.ResolveTarget → target.Resolver.EligibleTasks → taskARNs/describeTasks
- 重複・対応先: なし
- 根拠・限界: 243-278行は取得結果に加え送信inputを確認。service無しでnil送信する契約も含む。ready判定自体は対象外であり、全Describe応答がreadyという限界あり。
- 対応: 維持
- ケース別注意: with service / without service の2件。どちらも同じpaging fixtureだがservice parameterのoptional配線が異なるため維持。
- 実行: 2 leaf / 全件PASS。

| 実行サブケース | 個別判定 |
| --- | --- |
| `with_service` | 維持 |
| `without_service` | 維持 |

### T172 — TestResolverEligibleTasksFiltersUnreadyDescribeResults

- ソース: [internal/target/resolver_test.go:283](../internal/target/resolver_test.go#L283)
- 判定: **維持** / 状態: 対応済み
- 守る契約: DescribeTasks応答の停止task・exec無効task・agent未ready task・runtime欠損taskを除外し、2件のready taskのみ元の順序と値で返す。
- 本番経路: handler → Resolver.EligibleTasks → DescribeTasks → IsEligibleTask
- 重複・対応先: なし
- 根拠・限界: 独立した明示期待値を使い、全ready fixtureで見逃していたfilter呼出欠落を検出。
- 対応: 維持。M01のfilter除去変異を局所テストで検出済み。
- ケース別注意: 単一のmixed response。6 task中2 ready。
- 実行: 1 leaf / 全件PASS。

### T123 — TestResolverEligibleTasksRejectsRepeatedPaginationTokens

- ソース: [internal/target/resolver_test.go:319](../internal/target/resolver_test.go#L319)
- 判定: **維持** / 状態: 維持
- 守る契約: ListTasksの即時反復・token循環を検知し、operation/cluster/tokenを示す。
- 本番経路: handler.Exec/PortForward/RemotePortForward → view.ResolveTarget → target.Resolver.EligibleTasks → taskARNs
- 重複・対応先: TestResolverClustersRejectsRepeatedPaginationTokens（別APIの同種ループ）
- 根拠・限界: 別実装のpaginationと固有cluster診断を検証するため削減しない。
- 対応: 維持
- ケース別注意: same token repeats (2 calls) / tokens form a cycle (3 calls)。
- 実行: 2 leaf / 全件PASS。

| 実行サブケース | 個別判定 |
| --- | --- |
| `same_token_repeats` | 維持 |
| `tokens_form_a_cycle` | 維持 |

### T124 — TestResolverEligibleTasksChunksDescribeRequestsAndPreservesResponseOrder

- ソース: [internal/target/resolver_test.go:372](../internal/target/resolver_test.go#L372)
- 判定: **維持** / 状態: 維持
- 守る契約: 201件を100/100/1件へ分割し、各batchの正しいARN・Cluster/contextを渡し、List順ではなくDescribe応答順を保持する。
- 本番経路: handler.Exec/PortForward/RemotePortForward → view.ResolveTarget → target.Resolver.EligibleTasks → describeTasks
- 重複・対応先: TestResolverEligibleTasksChunksDescribeRequestsAndPreservesResponseOrder（旧名から移管済み）（通常の3分割）
- 根拠・限界: 実サービスの上限境界、末尾欠落、入力誤配線、応答順の改変を検出する。逆順応答により入力をそのまま期待値にした追認テストを回避。
- 対応: 維持
- ケース別注意:
- 実行: 1 leaf / 全件PASS。

### T126 — TestResolverEligibleTasksReturnsJoinedDescribeFailures

- ソース: [internal/target/resolver_test.go:424](../internal/target/resolver_test.go#L424)
- 判定: **維持** / 状態: 対応済み
- 守る契約: DescribeTasksの複数failureを全件文脈付きで返し、部分的に成功したtaskを返さない。
- 本番経路: Resolver.EligibleTasks → describeTasks → describeFailuresError
- 重複・対応先: なし
- 根拠・限界: 外部観測可能なerror全文とpartial返却禁止を維持し、Unwrap内部構造の固定だけを削除。
- 対応: LV13対応完了。維持。
- ケース別注意: 2 failureの文言とpartial結果なしを検証。
- 実行: 1 leaf / 全件PASS。

### T127 — TestResolverEligibleTasksFormatsBlankDescribeFailureFields

- ソース: [internal/target/resolver_test.go:465](../internal/target/resolver_test.go#L465)
- 判定: **維持** / 状態: 維持
- 守る契約: nil ARN/detail、空白ReasonのFailureでもpanicせず意味のあるplaceholderで診断する。
- 本番経路: handler.Exec/PortForward/RemotePortForward → view.ResolveTarget → target.Resolver.EligibleTasks → describeFailuresError
- 重複・対応先: TestResolverEligibleTasksReturnsJoinedDescribeFailures（error formattingの入口）
- 根拠・限界: 通常Failure内容テストにないnil/blankの外部応答をカバー。エラー表示のみでも問題解決のための契約として妥当。
- 対応: 維持。fixture共通化するなら通常Failureテーブルへ統合できるが、ケース自体は残す。
- ケース別注意:
- 実行: 1 leaf / 全件PASS。

### T128 — TestResolverEligibleTasksSkipsDescribeForEmptyTaskPages

- ソース: [internal/target/resolver_test.go:486](../internal/target/resolver_test.go#L486)
- 判定: **維持** / 状態: 維持
- 守る契約: 空ListTasksページでも次tokenを読み切り、全ページ空ならDescribeTasksを呼ばず空結果を返す。
- 本番経路: handler.Exec/PortForward/RemotePortForward → view.ResolveTarget → target.Resolver.EligibleTasks
- 重複・対応先: なし
- 根拠・限界: 503-507行で2ページ/Describeゼロを確認し、空inputの不要API callを防ぐ。Waitの空応答ケースとはpaginationとDescribe抑止の観測点が異なる。
- 対応: 維持
- ケース別注意:
- 実行: 1 leaf / 全件PASS。

### T129 — TestResolverAPIErrorsPreserveCauseAndContext

- ソース: [internal/target/resolver_test.go:511](../internal/target/resolver_test.go#L511)
- 判定: **維持** / 状態: 維持
- 守る契約: ListClusters/ListTasks/DescribeTasks各エラーをerrors.Isで発見でき、operationと該当clusterを診断文に残す。
- 本番経路: handler.Exec/PortForward/RemotePortForward → view.ResolveTarget → target.Resolver.Clusters/.EligibleTasks
- 重複・対応先: なし
- 根拠・限界: 3つの別return箇所を検証。名前のContextは診断文の文脈でありcontext.Contextの取消確認ではない（取消はwait側）。
- 対応: 維持
- ケース別注意: list clusters / list tasks / describe tasks の3ケース。
- 実行: 3 leaf / 全件PASS。

| 実行サブケース | 個別判定 |
| --- | --- |
| `describe_tasks` | 維持 |
| `list_clusters` | 維持 |
| `list_tasks` | 維持 |

### T130 — TestResolverRejectsNilAPIOutputs

- ソース: [internal/target/resolver_test.go:540](../internal/target/resolver_test.go#L540)
- 判定: **維持** / 状態: 維持
- 守る契約: 3APIが(nil,nil)を返してもnil dereferenceではなくoperation/cluster/nil responseを示す。
- 本番経路: handler.Exec/PortForward/RemotePortForward → view.ResolveTarget → target.Resolver.Clusters/.EligibleTasks
- 重複・対応先: なし
- 根拠・限界: nil応答は注入境界の防御。captureResolverPanicはpanicもerrorにするが、必要なnil response文言とoperation検証があるため通常nil dereferenceを誤合格にしない。SDKが生成する通常応答の検証ではないと区別。
- 対応: 維持
- ケース別注意: list clusters / list tasks / describe tasks の3ケース。
- 実行: 3 leaf / 全件PASS。

| 実行サブケース | 個別判定 |
| --- | --- |
| `describe_tasks` | 維持 |
| `list_clusters` | 維持 |
| `list_tasks` | 維持 |

### T131 — TestWaitForEligibleTasks

- ソース: [internal/target/wait_test.go:85](../internal/target/wait_test.go#L85)
- 判定: **維持** / 状態: 維持
- 守る契約: 13ケースで0待機の一回取得、retry後のfresh task、2秒pollと最終残時間、API失敗、負待機、親取消、API成功後の期限競合、clockとcontextの期限を保護。
- 本番経路: handler.Exec/PortForward/RemotePortForward → view.ResolveTarget → target.Resolver.WaitForEligibleTasks
- 重複・対応先: timeout names cluster service duration and readiness と sleep reaching the deadline prevents another lookup は期限到達時lookup抑止の一部が重なる
- 根拠・限界: 異なるタイミングの取消/期限を分離しており単純重複ではない。fake clockは時間依存ロジックを安定して観測できる。ただしproduction RealClock.Sleep自体は通らない。
- 対応: 維持。実Clockの取消観測を補う場合は小さな境界テストで十分。
- ケース別注意: zero ready: 呼出1回、元context/no deadline/no sleep。zero empty: 呼出1回と診断。pending then ready: 空→ready、fresh全task・2秒sleep。timeout: service/cluster/3sと2s+1s。API sentinel: 即return/no sleep。negative wait: API前拒否。already canceled: APIゼロ。cancel during sleep: cause伝搬。parent cancellation before ready response: 取消優先。child deadline before ready response:20msの実context期限優先。fake deadline before ready response:clock期限優先。sleep reaching deadline:追加APIなし。positive wait with fake epoch: API用deadlineは実時間のmaxWait。最後は注入clockのための境界契約で、一般利用者の機能とは分けて理解する。
- 実行: 13 leaf / 全件PASS。

| 実行サブケース | 個別判定 |
| --- | --- |
| `API_sentinel_returns_immediately_and_remains_discoverable` | 維持 |
| `already_canceled_parent_returns_context_canceled_without_lookup` | 維持 |
| `child_deadline_before_a_ready_API_response_wins_over_success` | 維持 |
| `fake_deadline_before_a_ready_API_response_wins_over_success` | 維持 |
| `negative_wait_is_rejected_before_an_API_lookup` | 維持 |
| `parent_cancellation_before_a_ready_API_response_wins_over_success` | 維持 |
| `parent_cancellation_during_sleep_remains_discoverable` | 維持 |
| `pending_then_ready_returns_the_fresh_final_tasks` | 維持 |
| `positive_wait_uses_a_real_API_timeout_independent_of_the_fake_clock_epoch` | 維持 |
| `sleep_reaching_the_deadline_prevents_another_lookup` | 維持 |
| `timeout_names_cluster_service_duration_and_readiness` | 維持 |
| `zero_wait_performs_one_empty_lookup_and_returns_details` | 維持 |
| `zero_wait_performs_one_ready_lookup_with_the_caller_context` | 維持 |

### T132 — TestResolveTargetSelectsExactReadyTaskARN

- ソース: [internal/view/resolve_test.go:56](../internal/view/resolve_test.go#L56)
- 判定: **維持** / 状態: 対応済み
- 守る契約: 同一service groupの複数taskがgroup＋short IDの正確なlabelとfull ARN valueで提示され、選択した2番目task/container/runtime/SSM targetを保持する
- 本番経路: handler → ResolveTarget → taskOptions → chooser → taskByARN
- 重複・対応先: 既存重複はLV01〜LV05で統合済み
- 根拠・限界: helper単体で確認していたlabel完全一致をここへ移植し、実選択経路で重複せず保護する。
- 対応: 対応済み。保護を維持。
- ケース別注意: 単一ケース。2件のoptionsを独立literal期待値と完全比較。
- 実行: 1 leaf / 全件PASS。

### T133 — TestResolveTargetWaitsAfterClusterResolution

- ソース: [internal/view/resolve_test.go:103](../internal/view/resolve_test.go#L103)
- 判定: **維持** / 状態: 維持
- 守る契約: cluster 選択の label は短名、value は full ARN。選択後にそのARN・service・wait durationを Resolver に渡し、ECSCluster と ClusterName を分けて返す。
- 本番経路: handler → ResolveTarget（cluster未指定）→ Resolver.Clusters → clusterOptions → choose → Resolver.WaitForEligibleTasks
- 重複・対応先: TestResolveTargetPreservesFullClusterARN は明示入力経路の別契約
- 根拠・限界: resolve_test.go:113-146 で選択肢、clusters→wait の順序、待機引数、結果を検証する。147-148 の clock!=nil は production clock の種類や機能までは証明しないが主要契約を損なうほどの弱点ではない。
- 対応: 維持。clock assertion の失敗文『production clock』は非nilだけを確認している旨に合わせる余地がある。
- ケース別注意: サブテストなし。staging/production の2 cluster のうち2番目を選択。
- 実行: 1 leaf / 全件PASS。

### T134 — TestResolveTargetRejectsUnofferedClusterChoice

- ソース: [internal/view/resolve_test.go:150](../internal/view/resolve_test.go#L150)
- 判定: **維持** / 状態: 維持
- 守る契約: chooser が返す有効な別ARN、または表示labelを候補のvalueと誤認しても、未提示clusterとして拒否し待機APIへ進まない。
- 本番経路: handler → ResolveTarget → hasOptionValue（resolve.go:57-59）
- 重複・対応先: なし。TaskByARN の identity テストとは resource と実装が別。
- 根拠・限界: resolve_test.go:181-184 は error・zero result・quit=false と wait 未実行まで確認。現在の正常 RenderOptions は提示valueしか返さないが、label/value配線の回帰を fail closed で検出する契約として価値がある。
- 対応: 維持。
- ケース別注意: 2件: valid unoffered ARN; valid label instead of option value。いずれも文字列構文の妥当性ではなく選択肢への所属を検証。
- 実行: 2 leaf / 全件PASS。

| 実行サブケース | 個別判定 |
| --- | --- |
| `valid_label_instead_of_option_value` | 維持 |
| `valid_unoffered_ARN` | 維持 |

### T135 — TestResolveTargetRejectsEmptyResourcesBeforeChooser

- ソース: [internal/view/resolve_test.go:190](../internal/view/resolve_test.go#L190)
- 判定: **維持** / 状態: 対応済み
- 守る契約: 空cluster、Waitの適格taskなしerror、注入resolverの不適格task成功返却に対し、zero resultとerrorを返してchooserを開始しない
- 本番経路: handler → ResolveTarget → Clusters / WaitForEligibleTasks → chooseOption
- 重複・対応先: 既存重複はLV01〜LV05で統合済み
- 根拠・限界: 本番Waitの成功空taskという偽fixtureをerror形式へ置換しcause/cluster文脈を確認。契約外成功値へのguardはempty-container代表1件。
- 対応: 対応済み。保護を維持。
- ケース別注意: 3cases: no clusters; wait reports no eligible tasks; resolver violates eligibility contract。最後のcaseのみ本番Resolver契約外の防御であると明記。
- 実行: 3 leaf / 全件PASS。

| 実行サブケース | 個別判定 |
| --- | --- |
| `no_clusters` | 維持 |
| `resolver_violates_eligibility_contract` | 維持 |
| `wait_reports_no_eligible_tasks` | 維持 |

### T136 — TestResolveTargetAutomaticSelection

- ソース: [internal/view/resolve_test.go:247](../internal/view/resolve_test.go#L247)
- 判定: **維持** / 状態: 維持
- 守る契約: 1 taskならtask UIを省略して複数containerから選べる。複数taskでも eligible container が1件だけならcontainer UIを省略し、非ready containerを選ばない。
- 本番経路: ResolveTarget → chooseOption(auto=true) → target.EligibleContainers → chooseOption(auto=true)
- 重複・対応先: TestResolveTargetSelectsExactReadyTaskARN は1container省略のみ部分重複
- 根拠・限界: resolve_test.go:257-273,292-308 は chooser のresource・呼出数・返されたidentityを比較し、非eligible container が先頭に存在する条件も確認する。task/containerで異なる選択段階を守る。
- 対応: 維持。
- ケース別注意: 2件: one task skips the task chooser; one eligible container skips the container chooser。後者はPENDING agentのcontainerを除外。
- 実行: 2 leaf / 全件PASS。

| 実行サブケース | 個別判定 |
| --- | --- |
| `one_eligible_container_skips_the_container_chooser` | 維持 |
| `one_task_skips_the_task_chooser` | 維持 |

### T137 — TestResolveTargetUserCancellation

- ソース: [internal/view/resolve_test.go:310](../internal/view/resolve_test.go#L310)
- 判定: **維持** / 状態: 維持
- 守る契約: cluster/task/container各段階でユーザー中断をエラー扱いせず、zero result・quit=trueで返し、次の選択やcluster後のwaitへ進まない。
- 本番経路: handler → ResolveTarget → choose（本番は listview.RenderOptions の quitting）
- 重複・対応先: TestExecHandlerViewCancellationCreatesNoRemoteSession; TestPortForwardHandlerViewCancellationDoesNotAllocateOrStart
- 根拠・限界: resolve_test.go:358-372 は各resourceで中断結果と呼出数を確認。handlerは中断後のAWS/session副作用停止を守るので単純重複ではない。ただしここはfake quitを注入し、実キー操作→quitを検証しない。
- 対応: 維持。別途 listview の q/Ctrl+C のテストが必要で、このテストを実UI中断の証拠とみなさない。
- ケース別注意: 3件: cluster chooser; task chooser; container chooser。
- 実行: 3 leaf / 全件PASS。

| 実行サブケース | 個別判定 |
| --- | --- |
| `cluster_chooser` | 維持 |
| `container_chooser` | 維持 |
| `task_chooser` | 維持 |

### T138 — TestResolveTargetPropagatesChooserErrors

- ソース: [internal/view/resolve_test.go:378](../internal/view/resolve_test.go#L378)
- 判定: **維持** / 状態: 対応済み
- 守る契約: cluster/task/container各chooserの通常errorでcauseとresource文脈を保持し、zero result・quit=false・後続停止。quitとerrorの同時返却もerror優先
- 本番経路: handler → ResolveTarget → chooseOption → Choose
- 重複・対応先: 既存重複はLV01〜LV05で統合済み
- 根拠・限界: 本番RenderOptions形式false+errorを全3段階で実行。競合形式の過剰matrixは代表1件へ統合。resolver呼出列とchooser1回も照合。
- 対応: 対応済み。保護を維持。
- ケース別注意: 4cases: cluster chooser; task chooser; container chooser; error wins over simultaneous quit。
- 実行: 4 leaf / 全件PASS。

| 実行サブケース | 個別判定 |
| --- | --- |
| `cluster_chooser` | 維持 |
| `container_chooser` | 維持 |
| `error_wins_over_simultaneous_quit` | 維持 |
| `task_chooser` | 維持 |

### T140 — TestResolveTargetPreservesFullClusterARN

- ソース: [internal/view/resolve_test.go:454](../internal/view/resolve_test.go#L454)
- 判定: **維持** / 状態: 維持
- 守る契約: 明示cluster ARNの周囲空白だけを除き、短名に置き換えず待機へ渡し、cluster一覧や不要なchooserを呼ばず、ECSClusterとClusterNameを分ける。
- 本番経路: ResolveTarget(inputClusterあり) → strings.TrimSpace → target.ClusterName → WaitForEligibleTasks
- 重複・対応先: TestResolveTargetWaitsAfterClusterResolution は選択入力側の部分重複
- 根拠・限界: resolve_test.go:463-477 は明示入力の正規化とcluster lookup省略を確認し、選択経路では捕まえない回帰を守る。
- 対応: 維持。
- ケース別注意: サブテストなし。前後空白付きfull ARN。
- 実行: 1 leaf / 全件PASS。

### T141 — TestResolveTargetRejectsMalformedMetadata

- ソース: [internal/view/resolve_test.go:481](../internal/view/resolve_test.go#L481)
- 判定: **維持** / 状態: 対応済み
- 守る契約: 入力cluster、列挙cluster、task ARNの不正metadataをchooserより前に拒否し、zero resultとresource文脈を返す
- 本番経路: handler → ResolveTarget → ClusterName / clusterOptions / taskOptions
- 重複・対応先: なし（旧helper不正ARNケースを統合済み）
- 根拠・限界: container metadata2caseは実際には空候補guardしか見ていなかったため別代表へ統合。残る3caseは各metadata検証を直接通る。
- 対応: 対応済み。保護を維持。
- ケース別注意: 3cases: input cluster; listed cluster; task ARN。
- 実行: 3 leaf / 全件PASS。

| 実行サブケース | 個別判定 |
| --- | --- |
| `input_cluster` | 維持 |
| `listed_cluster` | 維持 |
| `task_ARN` | 維持 |

### T143 — TestTaskOptionsUsesFallbackLabel

- ソース: [internal/view/resolve_test.go:537](../internal/view/resolve_test.go#L537)
- 判定: **維持** / 状態: 対応済み
- 守る契約: blank groupでもtask short IDにtask接頭辞を付けたlabelを返す
- 本番経路: ResolveTarget → taskOptions
- 重複・対応先: なし（blank groupの固有契約）
- 根拠・限界: 正常identityは上位成功test、不正ARNは上位metadata testへ移植済み。ここは固有のblank group fallbackだけを保持。
- 対応: 対応済み。保護を維持。
- ケース別注意: 単一ケース。旧blank group uses task labelを独立testへ変更。
- 実行: 1 leaf / 全件PASS。

### T144 — TestTaskByARNUsesOnlyExactFullARN

- ソース: [internal/view/resolve_test.go:548](../internal/view/resolve_test.go#L548)
- 判定: **維持** / 状態: 維持
- 守る契約: full ARN一致の2番目taskだけを返し、service group・短いtask ID・空白追加ARNで代用検索しない。拒否時は選択値とavailability文脈を保持。
- 本番経路: ResolveTarget → taskByARN（resolve.go:185-192）
- 重複・対応先: TestResolveTargetSelectsExactReadyTaskARN の正常full ARN検索のみ重複
- 根拠・限界: resolve_test.go:561-568 の3拒否値がidentity厳格性を独立に守る。上位正常系だけでは曖昧な短ID検索を許す実装へ変わったことを検出できない。
- 対応: 維持。将来上位経路へ移すなら未提示task choiceの3入力を保存する。
- ケース別注意: サブテストはないが1成功＋loop3拒否: service:payments、task-second、full ARN末尾空白。
- 実行: 1 leaf / 全件PASS。

### T171 — TestCLIBinaryContracts

- ソース: [main_test.go:16](../main_test.go#L16)
- 判定: **維持** / 状態: 対応済み
- 守る契約: ldflagsで埋めた1.2.3を実CLIのversion/--version/-vが返す。実登録されたexec/portforward/remoteportforwardのmake-input-fileで親別help、既定ファイル名、JSON全field/default値、成功出力を確認。
- 本番経路: go build -ldflags → main → init/RootCmd → versionまたはmake-input-file → MakeInputFile
- 重複・対応先: なし
- 根拠・限界: 本番helperや入力structから期待値を生成しない実バイナリの境界試験。AWS/pluginを起動せずversion初期化と本番登録を検証する。build/childは独立timeoutとWaitDelayで停止を限定。
- 対応: Q05/M05とLV06対応完了。既知linker版での実行を通常go testに含める。
- ケース別注意: 6 leaf: version 3入口、入力template 3親。各templateはhelpと生成の2processを実行。
- 実行: 6 leaf / 全件PASS。

| 実行サブケース | 個別判定 |
| --- | --- |
| `linker_version/long_flag` | 維持 |
| `linker_version/short_flag` | 維持 |
| `linker_version/subcommand` | 維持 |
| `registered_input_templates/exec` | 維持 |
| `registered_input_templates/portforward` | 維持 |
| `registered_input_templates/remoteportforward` | 維持 |

### T145 — TestStartExecSessionUsesAuthoritativeResponseAndFreshRuntimeID

- ソース: [pkg/command/exec_test.go:68](../pkg/command/exec_test.go#L68)
- 判定: **維持** / 状態: 維持
- 守る契約: ECSへ元入力/context/Interactive=trueを渡し、caller入力とは異なるExecuteCommand応答のcluster/task/containerを信頼してDescribeTasks。囮task/containerを除外しfresh runtime IDでplugin invocationを構成。session ID、5秒cleanup、SSM terminate adapterも検証。
- 本番経路: cmd/exec → handler.execHandler → StartExecSession → ECS ExecuteCommand / DescribeTasks → RemoteSession.Run
- 重複・対応先: TestExecHandlerSelectsExactSecondReplicaAndRunsInOrder（境界は重なるが独自反例あり）
- 根拠・限界: 68-153: caller値と応答値を意図的に変え、囮を置いた強いoracle。handler成功ケースはcaller=responseなのでこのunitの独自性がある。
- 対応: 維持する。
- ケース別注意: 単一ケース。
- 実行: 1 leaf / 全件PASS。

### T146 — TestStartExecSessionTerminatesExactSessionWhenPluginFails

- ソース: [pkg/command/exec_test.go:156](../pkg/command/exec_test.go#L156)
- 判定: **維持** / 状態: 維持
- 守る契約: StartExecSessionで作ったRemoteSessionをRunし、pluginへ元contextとinvocationを渡し、失敗原因を保持して正確なsessionを1度terminate、context valueを維持する。
- 本番経路: cmd/exec → handler.execHandler → StartExecSession → ECS ExecuteCommand / DescribeTasks → RemoteSession.Run
- 重複・対応先: TestExecHandlerPluginCancellationCleansUpWithDetachedContext; TestRemoteSessionTerminatesWhenPluginFails; TestStartExecSessionUsesAuthoritativeResponseAndFreshRuntimeID
- 根拠・限界: 156-190: handler cancellation試験と重なるが、constructor→Run→SSM callback配線をpackage内で局所検証し、元contextとInvocation全体の転送も確認。重複だけを理由に削るより責務を明確にして維持する価値がある。
- 対応: 維持を第一候補とする。handler/sessionと共通のfixtureやassertionを整理できる。統合するなら各固有assertionの移植とconstructor配線の保護を条件にする。
- ケース別注意: 単一ケース。
- 実行: 1 leaf / 全件PASS。

### T147 — TestStartExecSessionRejectsInvalidInputBeforeExecuteCommand

- ソース: [pkg/command/exec_test.go:193](../pkg/command/exec_test.go#L193)
- 判定: **維持** / 状態: 維持
- 守る契約: 空白cluster/task/container/command/regionを個別拒否し、原因フィールドを含むエラーを返し、ECS/SSM呼出しを一切しない。
- 本番経路: cmd/exec → handler.execHandler → StartExecSession → ECS ExecuteCommand / DescribeTasks → RemoteSession.Run
- 重複・対応先: なし
- 根拠・限界: 193-224: CLI入力validateと重なるcommandはあるが、regionはAWS config由来、targetはresolver由来で入力層では同じ契約を保証できない。StartExecSession公開境界の安全弁。
- 対応: 維持する。
- ケース別注意: cluster / task ARN / container / command / region の5ケース。各入力を一つだけ無効化。
- 実行: 5 leaf / 全件PASS。

| 実行サブケース | 個別判定 |
| --- | --- |
| `cluster` | 維持 |
| `command` | 維持 |
| `container` | 維持 |
| `region` | 維持 |
| `task_ARN` | 維持 |

### T148 — TestStartExecSessionWrapsExecuteCommandError

- ソース: [pkg/command/exec_test.go:228](../pkg/command/exec_test.go#L228)
- 判定: **維持** / 状態: 維持
- 守る契約: ExecuteCommand失敗のsentinelとcluster/task/container文脈を保持し、元contextを送り、DescribeTasksやTerminateSessionに進まない。
- 本番経路: cmd/exec → handler.execHandler → StartExecSession → ECS ExecuteCommand / DescribeTasks → RemoteSession.Run
- 重複・対応先: なし
- 根拠・限界: 228-248: remote session未作成時の停止条件とエラーのerrors.Is可視性を守る。
- 対応: 維持する。
- ケース別注意: 単一ケース。
- 実行: 1 leaf / 全件PASS。

### T149 — TestStartExecSessionValidatesExecuteCommandResponseAndCleansUp

- ソース: [pkg/command/exec_test.go:252](../pkg/command/exec_test.go#L252)
- 判定: **維持** / 状態: 維持
- 守る契約: 応答なし・sessionなし・IDなし/空白はterminateせず拒否。ID既知でstream/token/cluster/task/container欠落やARN不正なら正確なIDをterminateし、DescribeTasksを呼ばずエラーを返す。
- 本番経路: cmd/exec → handler.execHandler → StartExecSession → ECS ExecuteCommand / DescribeTasks → RemoteSession.Run
- 重複・対応先: なし
- 根拠・限界: 252-303: sessionを作成済みかどうかでcleanupが変わる重要な境界。identifier単体と違い、解析失敗時のremote cleanupまで確認する。
- 対応: 維持する。
- ケース別注意: 11ケース: nil output / nil session / missing session ID / blank session ID / missing stream URL / missing token / missing cluster ARN / missing task ARN / missing container name / invalid cluster ARN / invalid task ARN。最初4はterminateなし、残7はterminateあり。全てerrInvalidSessionResponseとExecuteCommand文脈、DescribeTasks未呼出を確認。
- 実行: 11 leaf / 全件PASS。

| 実行サブケース | 個別判定 |
| --- | --- |
| `blank_session_ID` | 維持 |
| `invalid_cluster_ARN` | 維持 |
| `invalid_task_ARN` | 維持 |
| `missing_cluster_ARN` | 維持 |
| `missing_container_name` | 維持 |
| `missing_session_ID` | 維持 |
| `missing_stream_URL` | 維持 |
| `missing_task_ARN` | 維持 |
| `missing_token` | 維持 |
| `nil_output` | 維持 |
| `nil_session` | 維持 |

### T150 — TestStartExecSessionValidatesDescribeTasksResponseAndCleansUp

- ソース: [pkg/command/exec_test.go:307](../pkg/command/exec_test.go#L307)
- 判定: **維持** / 状態: 維持
- 守る契約: ExecuteCommand後のDescribeTasks API失敗・nil応答・Failures・exact task不在・exact container不在・空白runtime IDを拒否し、作成済みsessionを1回terminate。元contextと原因を保持。
- 本番経路: cmd/exec → handler.execHandler → StartExecSession → ECS ExecuteCommand / DescribeTasks → RemoteSession.Run
- 重複・対応先: なし
- 根拠・限界: 307-381: 既存sessionがある状態でrefreshに失敗する経路を守る。resolver側DescribeTasks試験はsession未作成のため代替できない。
- 対応: 維持する。
- ケース別注意: 6ケース: API error / nil output / reported failure（有効taskと同居）/ missing exact task / missing exact container / blank runtime ID。
- 実行: 6 leaf / 全件PASS。

| 実行サブケース | 個別判定 |
| --- | --- |
| `API_error` | 維持 |
| `blank_runtime_ID` | 維持 |
| `missing_exact_container` | 維持 |
| `missing_exact_task` | 維持 |
| `nil_output` | 維持 |
| `reported_failure` | 維持 |

### T151 — TestStartExecSessionJoinsValidationAndTerminateErrors

- ソース: [pkg/command/exec_test.go:385](../pkg/command/exec_test.go#L385)
- 判定: **維持** / 状態: 維持
- 守る契約: IDはあるがstream URLがない応答で、validation errorとTerminateSession errorの両方をerrors.Isで保持し、正しいIDを1回terminateする。
- 本番経路: cmd/exec → handler.execHandler → StartExecSession → ECS ExecuteCommand / DescribeTasks → RemoteSession.Run
- 重複・対応先: TestRemoteSessionJoinsCleanupError（別入口）、TestStartPortForwardSessionJoinsValidationAndTerminateErrors（別constructor）
- 根拠・限界: 385-405: RemoteSession.Runではなくconstructor途中の失敗処理を通す。session単体のcleanup joinと異なる入口であり必要な結合保護。
- 対応: 維持する。
- ケース別注意: 単一ケース。
- 実行: 1 leaf / 全件PASS。

### T152 — TestStartPortForwardSessionUsesCallerContextAndExactInput

- ソース: [pkg/command/portforward_test.go:61](../pkg/command/portforward_test.go#L61)
- 判定: **維持** / 状態: 維持
- 守る契約: SSMへ元context、target/document/parametersを渡し、応答のID/stream/tokenとregion/targetから正確なRemoteSession.Invocation、5秒cleanupを作成。成功構築ではterminateしない。
- 本番経路: cmd/(remote)portforward → handler.portforwardHandler → StartPortForwardSession → SSM StartSession → RemoteSession.Run
- 重複・対応先: TestPortForwardHandlerSelectsSecondReplicaAndAllocatesLocalPort（異なる境界）
- 根拠・限界: adapterの透過的な値転送を守る。documentは引数透過を検証する立場なので入力定数と同じ期待値は妥当。AWS document literal自体の正しさはQ04対応によりhandler成功2試験が独立期待値で補完済み。
- 対応: 維持する。
- ケース別注意: 単一ケース。
- 実行: 1 leaf / 全件PASS。

### T153 — TestStartPortForwardSessionTerminatesExactSessionWhenPluginFails

- ソース: [pkg/command/portforward_test.go:117](../pkg/command/portforward_test.go#L117)
- 判定: **維持** / 状態: 維持
- 守る契約: SSM constructorが返したsessionからRunすると元contextとinvocationをpluginに渡し、pluginエラーを保持し正確なIDを1回terminate、context valueを維持する。
- 本番経路: cmd/(remote)portforward → handler.portforwardHandler → StartPortForwardSession → SSM StartSession → RemoteSession.Run
- 重複・対応先: TestPortForwardHandlerPluginFailureTerminatesAfterPlugin; TestRemoteSessionTerminatesWhenPluginFails
- 根拠・限界: 117-154: handlerのplugin失敗試験も同じconstructor→Run→SSM配線を通るが、元context・context value・Invocation全体は本試験で独立検証する。現在fixtureはremote documentにhostだけでportを欠くが、目的はremote入力妥当性でなく終了配線。
- 対応: 維持を第一候補とする。共通fixture/assertionの整理余地はある。remote fixtureはportNumber/localPortNumberを補えば本番入力形に近づく。
- ケース別注意: 単一ケース。
- 実行: 1 leaf / 全件PASS。

### T154 — TestStartPortForwardSessionRejectsInvalidInputBeforeStartSession

- ソース: [pkg/command/portforward_test.go:157](../pkg/command/portforward_test.go#L157)
- 判定: **維持** / 状態: 維持
- 守る契約: 空白target/region/documentを個別に拒否し、問題のフィールド名を返し、SSM start/terminateを呼ばない。
- 本番経路: cmd/(remote)portforward → handler.portforwardHandler → StartPortForwardSession → SSM StartSession → RemoteSession.Run
- 重複・対応先: なし
- 根拠・限界: 157-182: regionはAWS config、targetはresolved task、documentはwrapperから来るためCLI入力validateでは代替できないadapter境界。
- 対応: 維持する。
- ケース別注意: target / region / document の3ケース。
- 実行: 3 leaf / 全件PASS。

| 実行サブケース | 個別判定 |
| --- | --- |
| `document` | 維持 |
| `region` | 維持 |
| `target` | 維持 |

### T155 — TestStartPortForwardSessionWrapsStartSessionError

- ソース: [pkg/command/portforward_test.go:186](../pkg/command/portforward_test.go#L186)
- 判定: **維持** / 状態: 維持
- 守る契約: StartSession失敗のsentinelとtarget/API操作文脈を保持し、元contextで呼び、作成確認できないsessionをterminateしない。
- 本番経路: cmd/(remote)portforward → handler.portforwardHandler → StartPortForwardSession → SSM StartSession → RemoteSession.Run
- 重複・対応先: なし
- 根拠・限界: 186-213: remote API呼出失敗と作成後不正応答を区別する契約。
- 対応: 維持する。
- ケース別注意: 単一ケース。
- 実行: 1 leaf / 全件PASS。

### T156 — TestStartPortForwardSessionRejectsNilResponseWithoutTermination

- ソース: [pkg/command/portforward_test.go:216](../pkg/command/portforward_test.go#L216)
- 判定: **維持** / 状態: 維持
- 守る契約: SSMがnil応答を返すとinvalid responseとして拒否し、ID不明のterminateをしない。
- 本番経路: cmd/(remote)portforward → handler.portforwardHandler → StartPortForwardSession → SSM StartSession → RemoteSession.Run
- 重複・対応先: TestStartPortForwardSessionRejectsMissingSessionIDWithoutTermination（table統合可、ケース削除不可）
- 根拠・限界: 216-233: SDK境界の防御でありnil dereferenceや誤terminateを防ぐ。下のID欠落ケースとまとめられるがケース自体は別分岐。
- 対応: 維持する。
- ケース別注意: 単一ケース。
- 実行: 1 leaf / 全件PASS。

### T157 — TestStartPortForwardSessionRejectsMissingSessionIDWithoutTermination

- ソース: [pkg/command/portforward_test.go:236](../pkg/command/portforward_test.go#L236)
- 判定: **維持** / 状態: 維持
- 守る契約: SSM応答IDがnilまたは空白ならinvalid responseとして拒否し、terminateを呼ばない。
- 本番経路: cmd/(remote)portforward → handler.portforwardHandler → StartPortForwardSession → SSM StartSession → RemoteSession.Run
- 重複・対応先: なし
- 根拠・限界: 236-268: 本番constructorの実際のguardを通り、session.RunのID空文字手組み試験と異なりCLIで到達可能な外部応答を扱う。
- 対応: 維持する。
- ケース別注意: missing / blank の2ケース。
- 実行: 2 leaf / 全件PASS。

| 実行サブケース | 個別判定 |
| --- | --- |
| `blank` | 維持 |
| `missing` | 維持 |

### T158 — TestStartPortForwardSessionTerminatesWhenStartedResponseIsInvalid

- ソース: [pkg/command/portforward_test.go:272](../pkg/command/portforward_test.go#L272)
- 判定: **維持** / 状態: 維持
- 守る契約: IDがあるがstream URLまたはtokenが欠落する応答を拒否し、正しいIDを1度terminateしcontext valueを引き継ぐ。
- 本番経路: cmd/(remote)portforward → handler.portforwardHandler → StartPortForwardSession → SSM StartSession → RemoteSession.Run
- 重複・対応先: なし
- 根拠・限界: 272-319: remote resource作成後の不正応答でリソース漏れを防ぐ重要な契約。
- 対応: 維持する。
- ケース別注意: missing stream URL / missing token の2ケース。
- 実行: 2 leaf / 全件PASS。

| 実行サブケース | 個別判定 |
| --- | --- |
| `missing_stream_URL` | 維持 |
| `missing_token` | 維持 |

### T159 — TestStartPortForwardSessionJoinsValidationAndTerminateErrors

- ソース: [pkg/command/portforward_test.go:323](../pkg/command/portforward_test.go#L323)
- 判定: **維持** / 状態: 維持
- 守る契約: 不正応答のvalidation sentinelとterminate sentinelを両方保持し、正確なsession IDを1度terminateする。
- 本番経路: cmd/(remote)portforward → handler.portforwardHandler → StartPortForwardSession → SSM StartSession → RemoteSession.Run
- 重複・対応先: TestRemoteSessionJoinsCleanupError; TestStartExecSessionJoinsValidationAndTerminateErrors（別入口）
- 根拠・限界: 323-350: shared cleanupの単体と重なるがconstructorがcleanup結果をそのまま返すことを守る別入口。
- 対応: 維持する。
- ケース別注意: 単一ケース。
- 実行: 1 leaf / 全件PASS。

### T160 — TestRemoteSessionTerminatesWhenPluginFails

- ソース: [pkg/command/session_test.go:14](../pkg/command/session_test.go#L14)
- 判定: **維持** / 状態: 維持
- 守る契約: pluginへinvocationをそのまま渡し、失敗時に生きたcontextで自session IDを1度終了し、plugin原因とhandoff文脈を保持。
- 本番経路: handler.execHandler / portforwardHandler → Start*Session が返した RemoteSession.Run → plugin.Run → 失敗時 cleanupCreatedSession → SSM TerminateSession
- 重複・対応先: handlerのplugin failure/cancellation試験（結合範囲が異なる）
- 根拠・限界: 14-50: shared Runの中心契約を独立して説明・検証する。handler試験との重複はあるが、最小の原因切り分けとして有用。
- 対応: 維持する。
- ケース別注意: 単一ケース。
- 実行: 1 leaf / 全件PASS。

### T161 — TestRemoteSessionDoesNotTerminateAfterSuccessfulPlugin

- ソース: [pkg/command/session_test.go:53](../pkg/command/session_test.go#L53)
- 判定: **維持** / 状態: 維持
- 守る契約: pluginが正常終了した場合はRunがnilを返し、TerminateSessionを追加呼出ししない。
- 本番経路: handler.execHandler / portforwardHandler → Start*Session が返した RemoteSession.Run → plugin.Run → 失敗時 cleanupCreatedSession → SSM TerminateSession
- 重複・対応先: なし
- 根拠・限界: 53-67: pluginに正常なセッション寿命管理を任せる逆分岐を直接検証。失敗ケースでは代替できない。
- 対応: 維持する。
- ケース別注意: 単一ケース。
- 実行: 1 leaf / 全件PASS。

### T162 — TestRemoteSessionJoinsCleanupError

- ソース: [pkg/command/session_test.go:70](../pkg/command/session_test.go#L70)
- 判定: **維持** / 状態: 維持
- 守る契約: plugin失敗とcleanup失敗をerrors.Isで両方保持し、cleanupエラーにsession ID/操作文脈を付加する。
- 本番経路: handler.execHandler / portforwardHandler → Start*Session が返した RemoteSession.Run → plugin.Run → 失敗時 cleanupCreatedSession → SSM TerminateSession
- 重複・対応先: TestStartExecSessionJoinsValidationAndTerminateErrors; TestStartPortForwardSessionJoinsValidationAndTerminateErrors（入口の異なる結合保護）
- 根拠・限界: 70-89: shared cleanupの原因保持契約。constructor各入口はこの関数まで到達する配線を守るため同一ケースとは言い切れない。
- 対応: 維持する。
- ケース別注意: 単一ケース。
- 実行: 1 leaf / 全件PASS。

### T163 — TestRemoteSessionCleanupUsesLiveIndependentContext

- ソース: [pkg/command/session_test.go:92](../pkg/command/session_test.go#L92)
- 判定: **維持** / 状態: 維持
- 守る契約: 親が既にcancel済みでもcleanup開始時に生きたcontextを作り、context valueを保持し、設定timeout内の独立deadlineを与える。
- 本番経路: handler.execHandler / portforwardHandler → Start*Session が返した RemoteSession.Run → plugin.Run → 失敗時 cleanupCreatedSession → SSM TerminateSession
- 重複・対応先: TestExecHandlerPluginCancellationCleansUpWithDetachedContext（こちらはtimeout範囲まで検証）
- 根拠・限界: 92-137: WithTimeout(WithoutCancel(ctx))の意味をassert。handler側は期限存在しか見ないため、設定したtimeoutの上限を検証する独自の保護がある。
- 対応: 維持する。
- ケース別注意: 単一ケース。
- 実行: 1 leaf / 全件PASS。

### T164 — TestRemoteSessionDoesNotTerminateWithoutID

- ソース: [pkg/command/session_test.go:140](../pkg/command/session_test.go#L140)
- 判定: **維持** / 状態: 維持
- 守る契約: RemoteSession.IDを手動で空にした状態でpluginが失敗してもterminateを呼ばず、pluginエラーを返す。
- 本番経路: handler.execHandler / portforwardHandler → Start*Session が返した RemoteSession.Run → plugin.Run → 失敗時 cleanupCreatedSession → SSM TerminateSession
- 重複・対応先: TestRemoteSessionPassesOriginalContextToPlugin（ID空の同じguard）; constructorのID欠落試験（本番入口の保証）
- 根拠・限界: 現CLIのconstructorは空IDを拒否するがRemoteSessionは公開型で手組み可能。空IDでTerminateSessionを呼ばない防御を直接守る。contextテスト側の空IDは改善するが本ケースを単独削除しない。
- 対応: 防御契約として維持。ゼロ値sessionの保証を廃止する設計変更がある場合のみ実装と合わせて再評価。
- ケース別注意: 単一ケース。
- 実行: 1 leaf / 全件PASS。

### T165 — TestRemoteSessionCleanupTimeoutIsDiscoverable

- ソース: [pkg/command/session_test.go:161](../pkg/command/session_test.go#L161)
- 判定: **維持** / 状態: 対応済み
- 守る契約: cleanupの20ms期限がerror chainに残り、期限付与が欠落しても独立watchdogで局所的にFAILする。
- 本番経路: RemoteSession.Run → cleanupCreatedSession → terminate
- 重複・対応先: なし
- 根拠・限界: 検査対象ctx.Doneだけに依存せず1秒timerを持つ。deadline除去変異はwatchdog由来のFAILとなり全体test timeoutへ到達しない。
- 対応: Q13対応完了。deadline除去変異でwatchdogの失敗診断を確認。
- ケース別注意: 単一case。通常20msの期限、独立watchdog1秒。
- 実行: 1 leaf / 全件PASS。

### T166 — TestRemoteSessionPassesOriginalContextToPlugin

- ソース: [pkg/command/session_test.go:198](../pkg/command/session_test.go#L198)
- 判定: **維持** / 状態: 対応済み
- 守る契約: 元ctxの同一性・Value・取消Cause・Invocationをpluginへ保持し、plugin失敗後は有効session IDを1回cleanupする。
- 本番経路: RemoteSession.Run → plugin.Run → cleanupCreatedSession
- 重複・対応先: TestRemoteSessionDoesNotTerminateWithoutID（ID空guard部分）; TestExecHandlerPluginCancellationCleansUpWithDetachedContext（呼出時点のcancel状態は異なる）
- 根拠・限界: 空IDの便宜fixtureを有効IDへ修正。ctx/Causeの独立assertionを維持し、正常cleanup callbackで本番constructor後の状態を再現する。
- 対応: Q12対応完了。WithoutCancel変異を検出。
- ケース別注意: 単一case。有効ID s-1、caller取消、plugin失敗、cleanup1回を確認。
- 実行: 1 leaf / 全件PASS。

### T167 — TestAvailablePortClosesListenerBeforeReturningSelectedPort

- ソース: [pkg/port/available_test.go:29](../pkg/port/available_test.go#L29)
- 判定: **維持** / 状態: 維持
- 守る契約: tcp/127.0.0.1:0をlistenし、OSが選んだTCPAddr.Portを返す前にlistenerをcloseする。
- 本番経路: cmd portforward/remoteportforward → handler.portforwardHandler（portforward.go:75-83）→ productionDependencies.availablePort → AvailablePort → availablePort
- 重複・対応先: TestAvailablePortIsReleasedBeforeReturn（fake上の呼出契約と本番net.Listenの実効果が別）
- 根拠・限界: available_test.go:35-53はlisten引数・選択されたポート値・close済みを独立確認。実socket再bind試験だけでは127.0.0.1以外へのbindや定数ポート返却を安定して検知できず、消すべき重複ではない。
- 対応: 維持。
- ケース別注意: 単独ケース。
- 実行: 1 leaf / 全件PASS。

### T168 — TestAvailablePortPreservesListenError

- ソース: [pkg/port/available_test.go:57](../pkg/port/available_test.go#L57)
- 判定: **維持** / 状態: 維持
- 守る契約: listen失敗時に利用可能ポートを偽って返さず-1を返し、元エラーをerrors.Isで保持する。
- 本番経路: cmd portforward/remoteportforward → handler.portforwardHandler（portforward.go:75-83）→ productionDependencies.availablePort → AvailablePort → availablePort
- 重複・対応先: なし
- 根拠・限界: available.go:15-17のエラー分岐を決定的に発生させる。実socket失敗の再現は環境依存のためfakeの利用は適切。
- 対応: 維持。
- ケース別注意: 単独ケース。
- 実行: 1 leaf / 全件PASS。

### T169 — TestAvailablePortPreservesCloseError

- ソース: [pkg/port/available_test.go:71](../pkg/port/available_test.go#L71)
- 判定: **維持** / 状態: 維持
- 守る契約: ポート取得済みでもlistener.close失敗なら成功ポートではなく-1と元エラーを返し、closeが実行されたことも確認。
- 本番経路: cmd portforward/remoteportforward → handler.portforwardHandler（portforward.go:75-83）→ productionDependencies.availablePort → AvailablePort → availablePort
- 重複・対応先: なし
- 根拠・限界: available.go:21-22の分岐。通常のOS TCP listenerでは起こしにくい失敗を注入して返り値/因果を守る。実装のcloseエラー対応を契約として残す限り価値があり、稀少性だけで削除しない。
- 対応: 維持。
- ケース別注意: 単独ケース。
- 実行: 1 leaf / 全件PASS。

### T170 — TestAvailablePortIsReleasedBeforeReturn

- ソース: [pkg/port/available_test.go:92](../pkg/port/available_test.go#L92)
- 判定: **維持** / 状態: 維持
- 守る契約: 公開AvailablePortが有効なport番号を返し、その番号に直後に本物のloopback listenerを再bindできる。
- 本番経路: cmd portforward/remoteportforward → handler.portforwardHandler（portforward.go:75-83）→ productionDependencies.availablePort → AvailablePort → availablePort
- 重複・対応先: TestAvailablePortClosesListenerBeforeReturningSelectedPort（異なる境界を保護）
- 根拠・限界: available_test.go:93,101はpublic wrapperとreal net.Listenを通る唯一の試験。fake unitでは検知しないnet.Listen配線・実解放の退行を検知。closeから再bindまで他processに取得される競争の余地はあるが、本番API自体の予約保証ではない。
- 対応: 維持。空きポートの将来的予約まで保証するテストとは扱わず、実機で不安定化した場合のみ競合を調査。
- ケース別注意: 単独ケース。
- 実行: 1 leaf / 全件PASS。

## 削除・統合した関数の履歴

以下は現行関数数/CSVに含めない。ソース位置は監査時のcommitにおける位置であり、現行行番号ではない。

| 旧ID | 旧関数・監査時位置 | 移植先 / 判断 |
| --- | --- | --- |
| T015 | `TestPortforwardCommandInputFileHelpNamesParent` — `cmd/portforward/portforward_test.go:149` | [TestPortforwardHelpDocumentsInputAndAutomaticLocalPort](../cmd/portforward/portforward_test.go#L162)。詳細は上記対応ID表 |
| T021 | `TestRemotePortforwardCommandInputFileHelpNamesParent` — `cmd/remoteportforward/remoteportforward_test.go:181` | [TestRemotePortforwardHelpDocumentsRemoteHostAndAutomaticLocalPort](../cmd/remoteportforward/remoteportforward_test.go#L194)。詳細は上記対応ID表 |
| T027 | `TestVersionDefaultsToBuildInfo` — `cmd/root_test.go:152` | [TestCLIBinaryContracts](../main_test.go#L16)。詳細は上記対応ID表 |
| T056 | `TestNormalizeVersion` — `cmd/update/update_test.go:180` | [TestVerifyCandidateVersion](../cmd/update/install_test.go#L40)。詳細は上記対応ID表 |
| T057 | `TestAssetURL` — `cmd/update/update_test.go:213` | [TestUpdaterInstallsOnlyAfterVerification](../cmd/update/orchestration_test.go#L27)。詳細は上記対応ID表 |
| T095 | `TestValidatePortForwardJoinsTargetAndLocalErrors` — `internal/input/validate_test.go:100` | [TestResolvePortForwardReturnsZeroValueOnValidationError](../internal/input/resolve_test.go#L138)。詳細は上記対応ID表 |
| T125 | `TestResolverDescribeTaskBatchesUseIndependentSlices` — `internal/target/resolver_test.go:388` | [TestResolverEligibleTasksChunksDescribeRequestsAndPreservesResponseOrder](../internal/target/resolver_test.go#L372)。詳細は上記対応ID表 |
| T139 | `TestResolveTargetPropagatesChooserError` — `internal/view/resolve_test.go:447` | [TestResolveTargetPropagatesChooserErrors](../internal/view/resolve_test.go#L378)。詳細は上記対応ID表 |
| T142 | `TestTaskOptionsPreserveExactTaskIdentity` — `internal/view/resolve_test.go:589` | [TestResolveTargetSelectsExactReadyTaskARN](../internal/view/resolve_test.go#L56)。詳細は上記対応ID表 |

改名: T099 `TestItemFilterValueUsesLabelAndRetainsValue` → `TestItemFilterValueUsesLabel`、T138 `TestResolveTargetChooserErrorWinsOverCancellation` → `TestResolveTargetPropagatesChooserErrors`、T143 `TestTaskOptionsUsesFallbackLabelAndRejectsMalformedARN` → `TestTaskOptionsUsesFallbackLabel`。

ケース単位の整理: T066の特殊version4件、T141のcontainer metadata2件、T143のhelper不正ARN1件を削除。T007のfactory説明3件は代表1件へ縮約し実CLI3親へ移植。T135の空task成功fakeはWait errorへ置換、T138/T139のchooser errorは通常3段階＋同時quit代表1件へ統合。assertion単位の整理はLV05/LV10/LV13。


今回の追加ケース: Q08はpath判定3→10ケース＋実resolver8ケース。Q09はnil1→6ケース。Q10はcaller期限と自身の期限の2ケース。Q11はalready-latest/updatedの2ケース。既存の観測不足のケースを強化し、Q08以外では新しいTest関数を増やしていない。

## シェルテスト全件

実行元: `.github/workflows/test.yml:89`。本番経路: `.github/workflows/release.yml:63-65` → `script/verify-release-artifacts.sh`。下表9ケースはすべて実行PASS。

| ID | ケース・ソース | 守る契約 | 判定 |
| --- | --- | --- | --- |
| S01 | [valid release](../script/verify-release-artifacts_test.sh#L121) | checksum検証、実tar展開、実行可能tnnlのversion一致を通して正常完了 | 維持 |
| S02 | [unlisted publishable asset](../script/verify-release-artifacts_test.sh#L125) | 選択したasset以外もtnnl_*.tar.gzならmanifest掲載を必須にする | 維持 |
| S03 | [snapshot release](../script/verify-release-artifacts_test.sh#L131) | --allow-snapshotを明示したsnapshotタグとversionの一致を許可 | 維持 |
| S04 | [asset missing from manifest](../script/verify-release-artifacts_test.sh#L139) | 指定assetのchecksumがなければ展開前に拒否（tar呼出markerも確認） | 維持 |
| S05 | [corrupt checksum](../script/verify-release-artifacts_test.sh#L147) | sidecarを含むmanifest全体の検証が通るまで展開しない | 維持 |
| S06 | [wrong binary version](../script/verify-release-artifacts_test.sh#L156) | archive内tnnlの実行結果とreleaseタグが不一致なら拒否 | 維持 |
| S07 | [missing asset](../script/verify-release-artifacts_test.sh#L158) | 指定tar.gzがなければ文脈を持つ失敗になる | 維持 |
| S08 | [invalid tag](../script/verify-release-artifacts_test.sh#L159) | 先頭vのないタグをreject | 維持 |
| S09 | [missing checksum tool](../script/verify-release-artifacts_test.sh#L164) | sha256sum/shasumの両方がない時に検証をskipせず失敗 | 維持 |

シェルfixtureは実tar/実hash/実processを使うが、tnnl本体は生成したshスクリプト。GoReleaserのldflagsや各OS用実バイナリを検証したことにはならない。S05は「選択asset以外の不正checksumも拒否」という独自価値があり、Go updaterの単一asset検証と重複削除しない。

不足する境界: snapshotを許可flagなしで拒否するcase、dist/manifest欠落、非実行可能またはtnnl欠落archive、sha256sum/shasum各分岐の強制実行。現行のS08はinvalid tagの1形態のみ。未検証という記録であり、全枝にテストを追加する提案ではない。

## CIに埋め込まれたassertion・fixture

| ID/対象 | 守る契約・限界 | 判定 |
| --- | --- | --- |
| C01 [.github/workflows/test.yml:46-49](../.github/workflows/test.yml#L46) | 実mainを `go build -trimpath` し `tnnl version == dev` をassert。初回監査で同条件を再現してPASS。今回はCI定義を維持し、別途T171でlinker版の実バイナリを追加検証。通常版の開発ビルド配線は守るがlinker release版・他CLI登録・signal/exit契約を守らない。 | 維持・release版補完 |
| gofmt / go vet / actionlint / goreleaser check | 静的検査・設定検査。アプリのTest件数には含めない。今回はGoテスト・シェル・実バイナリ試験に加えgo vet/gofmtもPASS。actionlint/goreleaser checkやGitHub Actions全体の再実行ではない。 | 維持 |
| internal/session_manager/session_test.go:38 TestMain | 現テスト実行ファイルをhelper childとして再起動しargv・stdio・exit・blockを再現。Q10ではversion用helperとpipe継承する親子、release-file cleanupも追加。通常Test関数に算入しない。 | 実process境界を支えるfixture |
| cmd/update/install_test.go:28 TestMain | stdoutを継承するchild等を再現しWaitDelayの退行を捕捉。通常Test関数に算入しない。 | 実process境界を支えるfixture |



## 完了範囲と引き続き対象外の境界

台帳で改善IDを付けたLV01〜LV13・Q01〜Q13は全件完了。実AWS/実Session Manager Pluginとの接続、非空UIの実Program経路、productionDependencies全体の組立て、非ゼロwaitのhandler配線・本番Sleepはこの監査対応でも検証範囲外であり、完成済みとみなさない。公開Preflightと実PATH/symlink resolverの経路は今回検証済みになった。

本番変更は2箇所: 更新対象のgo-build directory判定を境界付きに修正、公開Preflightのexec.Cmdに200msのWaitDelayを設定。修正前再現・修正後PASS・退行検出の証拠はJSONを参照。

## マージ前検証

最新main `0bc466fed663dbeca2e953c0bebcd75cb1f95443` の依存関係を取り込み、race付き393ケース（fail/skip 0）、go vet、gofmtを再確認した。シェルverifierの9ケースもPASS。監査当初の基準commitと今回の検証ベースを証拠JSONで区別している。
