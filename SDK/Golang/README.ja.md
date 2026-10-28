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
   - `Update` で現在の状態を `data` に書き込みます。`data` の値は呼び出し間で保持されます。`Update` はすぐに返してください。デバイスからの受信は `Initialize` で起動した goroutine で行い、一番新しい値だけを保持して `Update` でそれをコピーします。`Update` の中でデバイスを読むと、デバイスの送信がホストの要求より速いときにデータが溜まって遅れます。
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

4. `<id>.<GOOS>.<GOARCH>.vfgmod`([末尾の一覧](../README.ja.md#モジュールプログラム))としてビルドし、モジュールページからインストールします。

   ```sh
   # Windows x64
   go build -ldflags="-s -w" -o MyModule.windows.amd64.vfgmod .
   ```

このリポジトリ内のモジュールは `modules/<id>` に置き、`go run ./cmd/build` でビルドします。`platforms` の判定とインストーラーへの追加もこのコマンドが行います。

## ドライバーを書く

ドライバーは複数のモジュールを1つのプログラムにまとめたものです([ドライバープログラム](../README.ja.md#ドライバープログラム))。各モジュールを上と同じように実装し、まとめて起動します。

```go
func main() {
	plugin.ServeDriver(plugin.Driver{
		ID:       "MyDriver",
		Metadata: adapter.MustLoadMetadata(files, "metadata.json"),
	}, face.New(), eye.New())
}
```

ホストは読み込むモジュールごとに `--source=<id>` を付けて起動します。`ServeDriver` はその 1 つのモジュールを提供します。

## 例

[`example`](example) は、ハードウェアなしで動くデモモジュールです。顎・笑顔・視線・瞬きを自動で動かします。

```sh
go build -o Demo.windows.amd64.vfgmod ./SDK/Golang/example
```

[`example-driver`](example-driver) は、同じようなモジュールを 3 つ(アイのみ・フェイスのみ・両方)内包するデモドライバーです。

```sh
go build -o DemoDriver.windows.amd64.vfgmod ./SDK/Golang/example-driver
```
