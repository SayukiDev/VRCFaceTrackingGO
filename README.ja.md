# VRCFaceTrackingGO

| [English](README.md) | **日本語** |

[VRCFaceTracking](https://github.com/benaclejames/VRCFaceTracking) を Go で再実装した Windows 向けアプリです。[Wails](https://wails.io) と Vue で作られています。Linux と macOS にも[一部制限付き](#linux)で対応しています。

トラッキングモジュールから目と顔のトラッキングを受け取り、OSC で VRChat に送ります。送り方は Unified Expressions (v2) のアバターパラメーター、VRChat のネイティブアイトラッキング、またはその両方から選べます。

> [!WARNING]
> 本プロジェクトは、元の VRCFaceTracking の忠実な再実装ではありません。元のロジックを参考にしつつ、独自の考え方で実装したバージョンです。そのため、動作や設定、モジュールの仕組みは元のものと異なります。

## 機能

- Unified Expressions (v2) と頭のパラメーター
- OSCQuery と通常 OSC の切り替え(OSCQuery は VRChat を自動で見つけて現在のアバターのパラメーターだけを送り、通常 OSC は固定ポートを使い、アバターが分からないときはすべてのパラメーターを送ります)
- VRChat のネイティブアイトラッキングへの出力
- ミューテーション(パラメーター調整、Unified Correctors、データフィルター)
- モジュールは別プロセスで動作し、落ちても自動で再起動
- モジュールは言語を問わず実装可能([Go・Rust 向けの SDK](SDK/README.ja.md) あり)
- モジュールの優先順位、モジュールごとの目/表情の選択、手動での割り当て
- Material Design 3 の UI(ライト/ダークテーマ、アクセントカラーの変更)
- 英語と日本語に対応
- マルチプラットフォーム対応(Windows 以外では[制限](#linux)があります)

## モジュール

| モジュール | 対応機器 | 対応 OS | 移植元 |
|---|---|---|---|
| [ALVR](modules/ALVR) | [ALVR](https://github.com/alvr-org/ALVR) 経由のフェイストラッキング対応ヘッドセット | Windows、Linux | [VRCFT-ALVR](https://github.com/alvr-org/VRCFT-ALVR) |
| [GXRFT](modules/GXRFT) | [gxr-root-ft](https://github.com/compdoge/gxr-root-ft) 経由の Samsung Galaxy XR | Windows、Linux、macOS | [VRCFT-ALVR-UE](https://github.com/compdoge/VRCFT-ALVR-UE) |
| [SteamLink](modules/SteamLink) | Steam Link 経由の Meta Quest | Windows | [LinkFT](https://github.com/ykeara/LinkFT) |
| [VirtualDesktop](modules/VirtualDesktop) | Virtual Desktop 経由の Quest Pro など | Windows | [VirtualDesktop.VRCFaceTracking](https://github.com/guygodin/VirtualDesktop.VRCFaceTracking) |
| [VRCFTPicoModule](modules/VRCFTPicoModule) | PICO 4 Pro / Enterprise | Windows | [VRCFTPicoModule](https://github.com/lonelyicer/VRCFTPicoModule) |

モジュールは独立したプログラム(`.vfgmod`)です。同梱のモジュールはインストーラーで、それ以外はアプリのモジュールページからインストールします。自作する場合は、Go・Rust の SDK とプロトコルの説明がある[モジュール SDK](SDK/README.ja.md) を参照してください。

## 動作環境

### Windows

- Windows 10 以降(x64)
- [WebView2 ランタイム](https://developer.microsoft.com/microsoft-edge/webview2/)(未インストールならインストーラーが導入します)

### Linux

- x64
- GTK 4 と WebKitGTK 6.0(Ubuntu 24.04 以降なら `libgtk-4-1` と `libwebkitgtk-6.0-4`)
- 制限:
  - Linux に対応したモジュールしか使用できません。
  - VRChat の OSC 設定を自動で有効にしません。また、OSC モードではアバター設定ファイルを読めないため、すべてのパラメーターを送信します。現在のアバターのパラメーターだけを送るには OSCQuery モードを使ってください。
  - システムのアクセントカラーは使えません。

### macOS

- Apple シリコン(arm64)の Mac、macOS 12 以降
- 署名していないため、初回は右クリックの「開く」から起動してください。
- 制限:
  - macOS に対応したモジュールしか使用できません。
  - VRChat の OSC 設定を自動で有効にしません。また、OSC モードではアバター設定ファイルを読めないため、すべてのパラメーターを送信します。現在のアバターのパラメーターだけを送るには OSCQuery モードを使ってください。
  - システムのアクセントカラーは使えません。

## ビルド

必要なもの: Go 1.26 以降、Node.js、[Wails v3 CLI](https://v3.wails.io/ja/quick-start/installation/)(`go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0`)、インストーラーを作る場合は [NSIS](https://nsis.sourceforge.io)。Linux では `libgtk-4-dev` と `libwebkitgtk-6.0-dev` も必要です。

```sh
go run ./cmd/build
```

`metadata.json` に今の OS が含まれるモジュールを `build/bin/modules` に、アプリとインストーラー(Windows のみ)を `build/bin` にビルドします。オプション:

| オプション | 既定値 | |
|---|---|---|
| `-app` | `true` | モジュールの後にアプリをビルドする |
| `-nsis` | Windows では `true` | インストーラーも作る(Windows のみ) |
| `-install` | `false` | モジュールをモジュールフォルダにコピーする(Windows は `%APPDATA%\VRCFaceTrackingGO\Modules`、ほかの OS は [SDK/README.ja.md](SDK/README.ja.md#モジュールプログラム) を参照) |
| `-tags` | | アプリに追加するビルドタグ(Linux で GTK 3 と WebKitGTK 4.1 を使う場合は `gtk3` など) |

開発中は `wails3 dev` でホットリロード付きで起動できます。その他のタスク(`wails3 task --list`)は `Taskfile.yml` と `build/` にあります。アプリのメタデータ(名前・バージョン・About のリンク)は `build/config.yml` にあり、変更したら `wails3 task common:update:build-assets` を実行してください。

GitHub でリリースを公開すると [`.github/workflows/release.yml`](.github/workflows/release.yml) が動き、Windows・Linux・macOS でビルドして、それぞれの ZIP をリリースに添付します。

## LLM の使用について

個人的な使用を目的に個人で開発しており、時間と労力に限りがあるため、このプロジェクトでは開発に LLM(大規模言語モデル)を積極的に使用しています。コードはひととおり確認していますが、気になる場合は各自でレビューすることをおすすめします。

## ライセンス

VRCFaceTrackingGO は、[`modules`](modules) 内のモジュールも含めて [GNU Affero General Public License v3.0](LICENSE)(AGPL-3.0)で公開しています。

- 商用を含め、自由に使用・改変・再配布できます。
- 配布する場合(改変版を含む)は、AGPL-3.0 のもとで完全なソースコードを入手できるようにし、著作権表示とライセンス表示を残し、変更点を明記する必要があります。
- 改変版をネットワーク越しに他人に使わせる場合も、その利用者にソースコードを提供する必要があります。
- 無保証です。

上記は要約です。正式な条件は [LICENSE](LICENSE) の英語原文に従います。

モジュールは、[モジュールプロトコル](SDK/README.ja.md)だけでアプリとやり取りする別のプログラムです。このリポジトリのコードを使わず、このプロトコルを実装しただけの自作モジュールは、AGPL-3.0 にする必要はありません。

コードの一部は Apache License 2.0 と MIT License のプロジェクトから移植しています。それらと同梱ライブラリのライセンスは [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md) を参照してください。
