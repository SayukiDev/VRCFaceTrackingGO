# VRCFaceTrackingGO

| [English](README.md) | **日本語** |

[VRCFaceTracking](https://github.com/benaclejames/VRCFaceTracking) を Go で再実装した Windows 向けアプリです。[Wails](https://wails.io) と Vue で作られています。

トラッキングモジュールから目と顔のトラッキングを受け取り、OSC で VRChat に送ります。送り方は Unified Expressions (v2) のアバターパラメーター、VRChat のネイティブアイトラッキング、またはその両方から選べます。

## 機能

- Unified Expressions (v2) と頭のパラメーター、OSCQuery によるアバター検出
- VRChat のネイティブアイトラッキングへの出力
- ミューテーション(パラメーター調整、Unified Correctors、データフィルター)
- モジュールは別プロセスで動作し、落ちても自動で再起動
- モジュールの優先順位、モジュールごとの目/表情の選択、手動での割り当て
- Material Design 3 の UI(ライト/ダークテーマ、アクセントカラーの変更)
- 英語と日本語に対応

## モジュール

| モジュール | 対応機器 | 移植元 |
|---|---|---|
| [GXRFT](modules/GXRFT) | [gxr-root-ft](https://github.com/compdoge/gxr-root-ft) 経由の Samsung Galaxy XR | [VRCFT-ALVR-UE](https://github.com/compdoge/VRCFT-ALVR-UE) |
| [VirtualDesktop](modules/VirtualDesktop) | Virtual Desktop 経由の Quest Pro など | [VirtualDesktop.VRCFaceTracking](https://github.com/guygodin/VirtualDesktop.VRCFaceTracking) |
| [VRCFTPicoModule](modules/VRCFTPicoModule) | PICO 4 Pro / Enterprise | [VRCFTPicoModule](https://github.com/lonelyicer/VRCFTPicoModule) |

モジュールは独立したプログラム(`.vfgmod`)です。同梱のモジュールはインストーラーで、それ以外はアプリのモジュールページからインストールします。自作する場合は [docs/module-protocol.ja.md](docs/module-protocol.ja.md) を参照してください。

## 動作環境

- Windows 10 以降
- [WebView2 ランタイム](https://developer.microsoft.com/microsoft-edge/webview2/)(未インストールならインストーラーが導入します)

## ビルド

必要なもの: Go 1.25 以降、Node.js、[Wails v2 CLI](https://wails.io/docs/gettingstarted/installation)、インストーラーを作る場合は [NSIS](https://nsis.sourceforge.io)。

```sh
go run ./cmd/build
```

すべてのモジュールを `build/bin/modules` に、アプリとインストーラーを `build/bin` にビルドします。オプション:

| オプション | 既定値 | |
|---|---|---|
| `-app` | `true` | モジュールの後にアプリをビルドする |
| `-nsis` | `true` | インストーラーも作る |
| `-install` | `false` | モジュールを `%APPDATA%\VRCFaceTrackingGO\Modules` にコピーする |

開発中は `wails dev` でホットリロード付きで起動できます。

## ライセンス

VRCFaceTrackingGO は、[`modules`](modules) 内のモジュールも含めて [GNU Affero General Public License v3.0](LICENSE)(AGPL-3.0)で公開しています。

- 商用を含め、自由に使用・改変・再配布できます。
- 配布する場合(改変版を含む)は、AGPL-3.0 のもとで完全なソースコードを入手できるようにし、著作権表示とライセンス表示を残し、変更点を明記する必要があります。
- 改変版をネットワーク越しに他人に使わせる場合も、その利用者にソースコードを提供する必要があります。
- 無保証です。

上記は要約です。正式な条件は [LICENSE](LICENSE) の英語原文に従います。

モジュールは、[モジュールプロトコル](docs/module-protocol.ja.md)だけでアプリとやり取りする別のプログラムです。このリポジトリのコードを使わず、このプロトコルを実装しただけの自作モジュールは、AGPL-3.0 にする必要はありません。

コードの一部は Apache License 2.0 と MIT License のプロジェクトから移植しています。それらと同梱ライブラリのライセンスは [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md) を参照してください。
