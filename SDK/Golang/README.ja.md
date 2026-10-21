# Go SDK

| [English](README.md) | **日本語** |

Go SDK は [モジュールプロトコル](../README.ja.md) のモジュール側の実装で、このリポジトリの次のパッケージです。

| パッケージ | 内容 |
|---|---|
| [`pkg/module/adapter`](../../pkg/module/adapter) | `Module` インターフェース、`Metadata`、`Support`、`LoadMetadata` |
| [`pkg/module/plugin`](../../pkg/module/plugin) | `Module` をモジュールプログラムとして動かす `Serve` |
| [`pkg/tracking`](../../pkg/tracking) | `TrackingData` と Unified Expressions のシェイプ番号 |

`plugin.Serve` が、コマンドライン(`--info`、`--vfgmod=true`)、通信の形式、ホストへのログ、panic からの復帰を受け持ちます。

## インストール

```sh
go get github.com/SayukiDev/VRCFaceTrackingGO@latest
```

## モジュールを書く

1. `adapter.Module` を実装します。

   ```go
   type Module interface {
       ID() string                 // 変えない ID。例: "MyModule"
       Metadata() Metadata
       Supported() Support         // 提供できるもの
       Initialize(ctx context.Context, logger *zap.Logger, avail Support) (Support, error)
       Update(ctx context.Context, data *tracking.TrackingData) error
       Teardown() error
   }
   ```

   - `Initialize` でデバイスを開きます。`avail` はホストがまだ必要としている範囲です。実際に提供する範囲(その部分集合)を返します。
   - `Update` で現在の状態を `data` に書き込みます。`data` の値は呼び出し間で保持されます。新しいデータを待つのは長くても数ミリ秒にしてください。
   - `Teardown` でデバイスを解放します。
   - ログは `Initialize` に渡される `logger` で出します。ホストのログに転送されます。標準出力には書かないでください。

2. メタデータはコードと同じフォルダの `metadata.json` に置き、アイコンと一緒に埋め込みます。

   ```json
   {
     "name": "My Module",
     "description": "Provides ...",
     "author": "me",
     "version": "1.0.0",
     "usageInstructions": "Start ... before activating this module",
     "icons": ["icon.png"],
     "platforms": ["windows", "linux"]
   }
   ```

   ```go
   //go:embed metadata.json icon.png
   var files embed.FS

   var metadata = adapter.MustLoadMetadata(files, "metadata.json")
   ```

3. `main` から起動します。

   ```go
   func main() { plugin.Serve(New()) }
   ```

4. `<id>.vfgmod` としてビルドし、モジュールページからインストールします。

   ```sh
   go build -ldflags="-s -w" -o MyModule.vfgmod .
   ```

このリポジトリ内のモジュールは `modules/<id>` に置き、`go run ./cmd/build` でビルドします。`platforms` の判定とインストーラーへの追加もこのコマンドが行います。

## 例

[`example`](example) は、ハードウェアなしで動くデモモジュールです。顎・笑顔・視線・瞬きを自動で動かします。

```sh
go build -o Demo.vfgmod ./SDK/Golang/example
```
