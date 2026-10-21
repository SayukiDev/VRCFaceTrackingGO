# モジュールプロトコル

| [English](module-protocol.md) | **日本語** |

トラッキングモジュールは独立したプログラムです。VRCFaceTrackingGO は有効なモジュールを子プロセスとして起動し、その標準入出力を通して、小さなバイナリのリクエスト/レスポンス形式でやり取りします。

リファレンス実装は [`internal/module/plugin`](../internal/module/plugin) です。`Serve` がモジュール側、`Client` がホスト側です。

## モジュールプログラム

モジュールプログラムは `<id>.vfgmod` という名前の実行ファイルで、アプリのデータフォルダ内の `Modules` にインストールされます。

| OS | モジュールフォルダ |
|---|---|
| Windows | `%APPDATA%\VRCFaceTrackingGO\Modules` |
| Linux | `~/.config/VRCFaceTrackingGO/Modules` |
| macOS | `~/Library/Application Support/VRCFaceTrackingGO/Modules` |

ホストは `.vfgmod` という拡張子だけで判別するので、どの言語で書いても構いません。Windows 以外ではファイルに実行権限が必要です。モジュールページからインストールすると実行権限付きで保存されます。

次のコマンドラインに対応する必要があります。

| コマンドライン | 動作 |
|---|---|
| `<prog> --info` | モジュールの情報を JSON で標準出力に出し、終了コード 0 で終了する |
| `<prog> --vfgmod=true` | 標準入力が閉じられるまで、標準入出力でホストに応答する |
| それ以外 | 起動を拒否する(標準エラー出力に案内を出し、0 以外の終了コードで終了するなど) |

最後の決まりは、誤って開かれたモジュールがデバイスに触れないようにするためです。

### 情報 (`--info`)

```json
{
  "id": "VirtualDesktop",
  "metadata": {
    "name": "Virtual Desktop",
    "description": "Provides face, eye and tongue tracking ...",
    "author": "SayukiDev",
    "version": "1.0.0",
    "pageUrl": "",
    "usageInstructions": "Make sure to install ...",
    "icons": ["<base64 PNG>"],
    "platforms": ["windows"]
  },
  "supported": { "eye": true, "expression": true }
}
```

- `id` は必須で、`^[A-Za-z0-9][A-Za-z0-9._-]*$` に一致する必要があります。設定のキーとインストール時のファイル名に使われるので、変更してはいけません。ID が重複したモジュールは読み込まれません。
- `metadata` はモジュールページに表示されます。`icons` は PNG 画像を base64 文字列にしたものです。
- `platforms` は、モジュールをビルド・配布する OS の一覧です(Go の `GOOS` の値: `windows`、`linux`、`darwin`)。省略可能で、空ならすべての OS が対象です。`go run ./cmd/build` は、今の OS が含まれないモジュールをビルドしません。
- `supported` は、初期化前の時点でモジュールが提供できるものを表します。
- ホストは `--info` の応答を最大 5 秒待ちます。情報を読み直すのは、ファイルのサイズか更新日時が変わったときだけです。

## プロセスの流れ

1. ホストが `<prog> --vfgmod=true` を起動します。標準入力・標準出力・標準エラー出力はパイプにつながります(Windows ではコンソールウィンドウは表示されません)。
2. **Initialize** を 1 回送ります。時間制限は 15 秒です(設定で変更可)。
3. モジュールの使用中は、10 ミリ秒ごとに **Update** を送ります(設定で変更可)。各 Update には 5 秒以内に応答する必要があります。
4. モジュールを止めるときは **Teardown** を送り(時間制限 3 秒)、標準入力を閉じます。モジュールは EOF を受けたら終了しなければなりません。さらに 3 秒経っても終了しない場合は強制終了されます。

ホストが異常終了した場合も、モジュールは一緒に終了します。Windows ではホストがすべてのモジュールを kill-on-close のジョブオブジェクトに入れ、Linux では親プロセスが終わるとカーネルがモジュールを終了させます(`PR_SET_PDEATHSIG`)。macOS にはこの仕組みがないため、`plugin.Serve` は親プロセスが変わったら終了します。ほかの言語のモジュールも、最低限、標準入力が EOF になったら終了してください。

モジュールが終了した場合や Update が時間切れになった場合、ホストはプログラムを再起動し、以前と同じ対応範囲で初期化し直します。最初の再起動までは 1 秒待ち、その後は失敗のたびに待ち時間を倍にします。1 分以内に 3 回を超えて失敗すると、再起動をあきらめます。再起動後に以前と異なる対応範囲を返した場合も失敗として扱います。

## 通信の形式

ホストはリクエストを 1 つ送り、そのレスポンスを受け取ってから次を送ります。整数と浮動小数点数はすべて**リトルエンディアン**です。

### リクエスト

| 命令 | バイト | ペイロード |
|---|---|---|
| Initialize | `'I'` (0x49) | 対応範囲 (1 バイト) |
| Update | `'U'` (0x55) | なし |
| Teardown | `'T'` (0x54) | なし |

不明な命令はプロトコルエラーです。モジュールは終了してください。

### レスポンス

```
成功: 0x00  ペイロード
失敗: 0x01  長さ (uint32)  メッセージ (長さ分のバイト、UTF-8)
```

エラーメッセージは 64 KiB までです。Initialize のエラーは起動失敗を意味し、ホストはプロセスを終了させます。Update のエラーはそのフレームを飛ばすだけです。このようなエラー(「デバイスがない」など)は想定内で、同じエラーが続く間は 10 秒に 1 回だけログに出ます。Teardown のエラーはログに記録されます。

| リクエスト | 成功時のペイロード |
|---|---|
| Initialize | 許可された対応範囲 (1 バイト) |
| Update | トラッキングデータ (416 バイト) |
| Teardown | なし |

### 対応範囲のバイト

| ビット | 意味 |
|---|---|
| 0 (0x01) | 目のトラッキング |
| 1 (0x02) | 表情のトラッキング |

Initialize では、ホストがこのモジュールにまだ必要としている種類を送ります(残りは別のモジュールがすでに担当している場合があります)。モジュールは実際に提供するものを返します。これは送られた範囲の部分集合でなければなりません。

### トラッキングデータ

`float32` を 104 個(416 バイト)、次の順に並べます。

| オフセット | 個数 | 内容 |
|---|---|---|
| 0 | 4 | 左目: 視線 X、視線 Y、瞳孔径 (mm)、開き具合 |
| 16 | 4 | 右目: 視線 X、視線 Y、瞳孔径 (mm)、開き具合 |
| 32 | 2 | 瞳孔径の最大値、最小値 |
| 40 | 88 | Unified Expressions のシェイプの重み |
| 392 | 6 | 頭: ヨー、ピッチ、ロール、位置 X、Y、Z |

- 視線は接空間での方向です。X = tan(ヨー) で右が正、Y = tan(ピッチ) で上が正です。
- 開き具合とシェイプの重みは 0(閉じている/無表情)から 1 です。
- 瞳孔径の最大値・最小値は、これまでに観測した瞳孔径の範囲です。ホストはこれを使って瞳孔の大きさを正規化します。
- シェイプの順番は [`pkg/tracking/expressions.go`](../pkg/tracking/expressions.go) の `tracking.UnifiedExpression` の順(`EyeSquintRight`、`EyeSquintLeft`、`EyeWideRight`、...)で、VRCFaceTracking の Unified Expressions の仕様に従っています。

データは差分ではなく、モジュールの現在の状態すべてです。モジュールがトラッキングしない値は初期値のままにします。初期値は、開き具合が 1、瞳孔径の最小値が 999、それ以外は 0 です。

## ログ

標準出力はプロトコル専用なので、モジュールはそれ以外のものを決して出力してはいけません。ログは**標準エラー出力**に、zap 形式の JSON を 1 行ずつ書きます。

```json
{"level":"info","ts":1729500000.0,"logger":"PICO","msg":"Using port","port":29765,"legacy":false}
```

ホストは `level`、`logger`、`msg` とそれ以外のすべての項目をそのまま記録し直します。`error` より上のレベルは `error` に下げるので、モジュールがホストを panic させたり終了させたりすることはできません。JSON でない行(クラッシュ時のスタックトレースなど)は警告として記録されます。

## Go でモジュールを書く

このリポジトリでは、モジュールは `modules/<id>` に置く `main` パッケージです。

```go
package main

import "VRCFaceTrackingGo/internal/module/plugin"

func main() { plugin.Serve(New()) }
```

`New` は `adapter.Module`([`internal/module/adapter`](../internal/module/adapter/module.go))を返します。`plugin.Serve` が、コマンドラインの処理、プロトコル、ログ、panic からの復帰を受け持ちます。メタデータはコードと同じフォルダの `metadata.json` に置き、`embed.FS` から `adapter.MustLoadMetadata` で読み込みます。ビルドスクリプトとインストーラーも同じファイルを読みます。特定の OS でしか動かないモジュールは、ここで `platforms` を指定してください。アイコンはこのファイルにファイル名で書き、PNG として埋め込みます。

すべてのモジュールは `go run ./cmd/build` で、1 つだけなら次のコマンドでビルドできます。

```sh
go build -ldflags="-s -w" -o build/bin/modules/<id>.vfgmod ./modules/<id>
```

Go の `internal` の規則により、このリポジトリの外のコードはこれらのパッケージを import できません。外部で開発するモジュールは、次の節のとおり上記のプロトコルを自分で実装する必要があります。

## ほかの言語でモジュールを書く

アプリが動く OS の実行ファイルにできれば、どの言語でも書けます。ホストは `.vfgmod` ファイルを直接起動するので、スクリプトではなくネイティブのプログラム(Windows は PE、Linux は ELF、macOS は Mach-O)である必要があります。スクリプト言語の場合は PyInstaller などで実行ファイルにまとめ、`<id>.vfgmod` に名前を変えてください。Windows ではコンソールウィンドウをホストが非表示にするので、コンソールアプリのままで問題ありません。情報の `platforms` には、ビルドした OS を指定してください。

チェックリスト:

- `--info` と `--vfgmod=true` だけを受け付け、それ以外は起動を拒否する。
- Windows で改行コードの変換が起きないよう、標準入力と標準出力を**バイナリモード**にする(Python は `sys.stdin.buffer`、C は `_setmode(_fileno(stdin), _O_BINARY)`、.NET は `Console.OpenStandardInput()`)。
- レスポンスを書くたびに標準出力を **flush** する。ホストは届くまで待ち続けます。
- 標準出力にはレスポンス以外を絶対に書かない。ログやライブラリの出力は標準エラー出力に送る。
- トラッキングデータのバッファはセッション中 1 つだけ持ち、上記の初期値で始めて、Update のたびにその場で更新する。
- Update はすぐに返す。新しいデータを待つのは長くても数ミリ秒にとどめ、その時点の状態を返す。5 秒応答がないとハングとみなされます。
- Teardown でデバイスを解放し、標準入力が EOF になったら終了する。

### 例 (Python)

表情のトラッキングだけを提供する最小のモジュールです。

```python
import json
import struct
import sys

EXPRESSIONS = 88
FLOATS = 10 + EXPRESSIONS + 6          # eyes, shapes, head: 416 bytes
SHAPES = 10                            # float index of the first shape


def describe():
    return {
        "id": "MyModule",
        "metadata": {"name": "My Module", "description": "Example module",
                     "author": "me", "version": "1.0.0", "icons": []},
        "supported": {"eye": False, "expression": True},
    }


def log(level, msg, **fields):
    print(json.dumps({"level": level, "logger": "MyModule", "msg": msg, **fields}),
          file=sys.stderr, flush=True)


def ok(out, payload=b""):
    out.write(b"\x00" + payload)
    out.flush()


def fail(out, msg):
    b = msg.encode()
    out.write(b"\x01" + struct.pack("<I", len(b)) + b)
    out.flush()


def main():
    args = sys.argv[1:]
    if args == ["--info"]:
        json.dump(describe(), sys.stdout)
        return
    if args != ["--vfgmod=true"]:
        print("Install this module from the Modules page.", file=sys.stderr)
        sys.exit(2)

    stdin, out = sys.stdin.buffer, sys.stdout.buffer
    sys.stdout = sys.stderr            # stray prints must not corrupt the stream

    data = [0.0] * FLOATS
    data[3] = data[7] = 1.0            # left / right eye openness
    data[9] = 999.0                    # min dilation

    while op := stdin.read(1):         # b"" means the host closed stdin
        if op == b"I":
            avail = stdin.read(1)[0]
            granted = avail & 0b10     # expressions only, if still needed
            log("info", "Initialized", granted=granted)
            ok(out, bytes([granted]))
        elif op == b"U":
            # Read the device here and write the weights, e.g. JawOpen:
            # data[SHAPES + 22] = jaw_open
            ok(out, struct.pack(f"<{FLOATS}f", *data))
        elif op == b"T":
            ok(out)                    # release the device here
        else:
            log("error", "Unknown request", op=op.hex())
            sys.exit(1)


if __name__ == "__main__":
    main()
```

対象の OS 上で `MyModule.vfgmod` にビルドします。

```sh
pyinstaller --onefile --name MyModule mymodule.py
# Windows
copy dist\MyModule.exe MyModule.vfgmod
# Linux / macOS
cp dist/MyModule MyModule.vfgmod
```

あとはモジュールページからインストールします。シェイプの番号は [`pkg/tracking/expressions.go`](../pkg/tracking/expressions.go) での並び順(0 始まり)です。
