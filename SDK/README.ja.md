# モジュール SDK

| [English](README.md) | **日本語** |

トラッキングモジュールは独立したプログラムです。VRCFaceTrackingGO は有効なモジュールを子プロセスとして起動し、その標準入出力を通して、小さなバイナリのリクエスト/レスポンス形式でやり取りします。

SDK がプロトコルのモジュール側を実装しているので、モジュールはトラッキングデータを用意するだけで済みます。

| 言語 | SDK | 例 |
|---|---|---|
| Go | [`pkg/module`](../pkg/module)([ガイド](Golang/README.ja.md)) | [`Golang/example`](Golang/example) |
| Rust | [`Rust`](Rust)(クレート `vrcftgo-sdk`、tokio) | [`Rust/examples/demo.rs`](Rust/examples/demo.rs) |

このドキュメントの残りでは、SDK を使わずにモジュールを書くためのプロトコルそのものを説明します。リファレンス実装は [`pkg/module/plugin`](../pkg/module/plugin) です。`Serve` がモジュール側、`Client` がホスト側です。

## モジュールプログラム

モジュールプログラムは `<id>.<os>.<arch>.vfgmod` という名前の実行ファイルで、アプリのデータフォルダ内の `Modules` にインストールされます。

| OS | モジュールフォルダ |
|---|---|
| Windows | `%APPDATA%\VRCFaceTrackingGO\Modules` |
| Linux | `~/.config/VRCFaceTrackingGO/Modules` |
| macOS | `~/Library/Application Support/VRCFaceTrackingGO/Modules` |

ホストはファイル名の末尾だけで判別するので、どの言語で書いても構いません。`<os>` と `<arch>` はビルド対象のプラットフォームを Go の `GOOS`・`GOARCH` の値で表したもので、ホストは自分のプラットフォーム用のものだけを認識・インストールします。

| プラットフォーム | 末尾 |
|---|---|
| Windows x64 | `.windows.amd64.vfgmod` |
| Windows ARM64 | `.windows.arm64.vfgmod` |
| Linux x64 | `.linux.amd64.vfgmod` |
| Linux ARM64 | `.linux.arm64.vfgmod` |
| macOS Intel | `.darwin.amd64.vfgmod` |
| macOS Apple シリコン | `.darwin.arm64.vfgmod` |

Windows 以外ではファイルに実行権限が必要です。モジュールページからインストールすると実行権限付きで保存されます。

実行ファイル以外のファイル(ライブラリ、データ、モデルなど)も必要なモジュールは、**モジュールアーカイブ**として配布します。`<name>.<os>.<arch>.vfgmodf`(例: `MyModule.windows.amd64.vfgmodf`)という名前の zip ファイルで、最上位にプログラムを `index.vfgmod` として、ほかのファイルと一緒に入れます。モジュールページからインストールすると、モジュールフォルダー内の `<id>.<os>.<arch>.vfgmodf` フォルダーに展開され、ホストはその中の `index.vfgmod` を実行します。それ以外は単一ファイルの場合と同じです。アンインストールするとフォルダーごと削除されます。

```sh
# Windows x64: index.vfgmod と同梱ファイルをフォルダーに入れ、その中身を zip にする
cd MyModule && zip -r ../MyModule.windows.amd64.vfgmodf .
```

モジュールプログラムは、自分のあるフォルダーを作業ディレクトリとして起動されるので、同梱したファイルは相対パスで開けます。

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

## ドライバープログラム

ドライバーは、複数のモジュールを1つにまとめたモジュールプログラムです。ファイル名・通信の形式・プロセスの流れはモジュールと同じで、コマンドラインと `--info` だけが異なります。ホストはドライバーをモジュールページの別の欄に表示し、ドライバー自体は読み込みません。内包されるモジュールはモジュール一覧に通常のモジュールとして並び、優先順位・有効化・アイ／フェイスの選択も同じように使えますが、アンインストールはドライバーごとにしかできません。

| コマンドライン | 動作 |
|---|---|
| `<prog> --info` | ドライバーの情報(下記)を出力し、終了コード 0 で終了する |
| `<prog> --vfgmod=true --source=<id>` | モジュール `<id>` を、モジュールプログラムと同じように提供する |
| それ以外 | 起動を拒否する(ソースのない `--vfgmod=true` も含む) |

```json
{
  "kind": "driver",
  "id": "PicoDriver",
  "metadata": { "name": "Pico Driver", "...": "..." },
  "modules": [
    { "id": "PicoFace", "metadata": { "...": "..." }, "supported": { "eye": false, "expression": true } },
    { "id": "PicoEye",  "metadata": { "...": "..." }, "supported": { "eye": true,  "expression": false } }
  ]
}
```

- `kind` は `"driver"` です。モジュールプログラムでは省略します。ドライバー自体の `supported` は使われません。
- `modules` は空にできません。各 ID はモジュールの ID と同じ規則に従い、インストール済みのすべてのモジュール・ドライバーと同じ名前空間を共有します。すでに使われている ID を含むプログラムは読み込まれません(インストール時は拒否されます)。
- ホストは読み込むモジュールごとに、それぞれの `--source` を付けて 1 プロセスずつ起動し、異常終了後は同じ `--source` で再起動します。同じドライバーの 2 つのモジュールが 1 つのプロセスを共有することはありません。

Go SDK には `plugin.ServeDriver`、Rust SDK には `serve_driver` があります。

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

## Go・Rust でモジュールを書く

SDK を使ってください。[Go のガイド](Golang/README.ja.md)と [Rust のクレート](Rust/README.ja.md)を参照してください。

## ほかの言語でモジュールを書く

アプリが動く OS の実行ファイルにできれば、どの言語でも書けます。ホストは `.vfgmod` ファイルを直接起動するので、スクリプトではなくネイティブのプログラム(Windows は PE、Linux は ELF、macOS は Mach-O)である必要があります。スクリプト言語の場合は PyInstaller などで実行ファイルにまとめ、`<id>.<os>.<arch>.vfgmod` に名前を変えてください。Windows ではコンソールウィンドウをホストが非表示にするので、コンソールアプリのままで問題ありません。情報の `platforms` には、ビルドした OS を指定してください。

チェックリスト:

- `--info` と `--vfgmod=true` だけを受け付け、それ以外は起動を拒否する。
- Windows で改行コードの変換が起きないよう、標準入力と標準出力を**バイナリモード**にする(Python は `sys.stdin.buffer`、C は `_setmode(_fileno(stdin), _O_BINARY)`、.NET は `Console.OpenStandardInput()`)。
- レスポンスを書くたびに標準出力を **flush** する。ホストは届くまで待ち続けます。
- 標準出力にはレスポンス以外を絶対に書かない。ログやライブラリの出力は標準エラー出力に送る。
- トラッキングデータのバッファはセッション中 1 つだけ持ち、上記の初期値で始めて、Update のたびにその場で更新する。
- Update には最新の状態ですぐに応答する。デバイスからの受信はバックグラウンド(別スレッドやデバイス側のイベント)で行い、一番新しい値だけを保持する。Update の中でデバイスを読むと、デバイスの送信がホストの要求より速いときにデータが溜まって遅れます。5 秒応答がないとハングとみなされます。
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
            # Copy in the newest values your receiving thread stored, e.g. JawOpen:
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

対象のプラットフォーム上でビルドし、そのプラットフォームの名前を付けます。

```sh
pyinstaller --onefile --name MyModule mymodule.py
# Windows x64
copy dist\MyModule.exe MyModule.windows.amd64.vfgmod
# Linux x64
cp dist/MyModule MyModule.linux.amd64.vfgmod
# macOS Apple シリコン
cp dist/MyModule MyModule.darwin.arm64.vfgmod
```

あとはモジュールページからインストールします。シェイプの番号は [`pkg/tracking/expressions.go`](../pkg/tracking/expressions.go) での並び順(0 始まり)です。
