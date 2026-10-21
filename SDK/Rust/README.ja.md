# Rust SDK

| [English](README.md) | **日本語** |

`vrcftgo-sdk` は、[モジュールプロトコル](../README.ja.md) のモジュール側を Rust で実装したクレートです(tokio ベース)。[`serve`](src/serve.rs) が、コマンドライン(`--info`、`--vfgmod=true`)、通信の形式、非同期ランタイム、ホストがいなくなったときの終了処理を受け持ちます。

## インストール

```toml
[dependencies]
vrcftgo-sdk = { git = "https://github.com/SayukiDev/VRCFaceTrackingGO" }
tokio = { version = "1", features = ["time"] }   # 自分でも tokio を使う場合のみ
serde_json = "1"                                 # ログの項目用
```

## モジュールを書く

```rust
use vrcftgo_sdk::{Logger, Metadata, Module, Result, Support, TrackingData, UnifiedExpression};

struct MyModule;

impl Module for MyModule {
    // 変えない ID。設定のキーとインストール時のファイル名に使われます。
    fn id(&self) -> &str { "MyModule" }

    fn metadata(&self) -> Metadata {
        Metadata {
            name: "My Module".into(),
            description: "Provides ...".into(),
            icons: vec![include_bytes!("icon.png").to_vec()],
            platforms: vec!["windows".into(), "linux".into()],
            ..Default::default()
        }
    }

    fn supported(&self) -> Support { Support::BOTH }

    // デバイスを開く。avail はホストがまだ必要としている範囲。
    async fn initialize(&mut self, logger: &Logger, avail: Support) -> Result<Support> {
        logger.info("Started", serde_json::Value::Null);
        Ok(avail)
    }

    // 現在の状態を書き込む。data の値は呼び出し間で保持される。
    async fn update(&mut self, data: &mut TrackingData) -> Result<()> {
        data.set_shape(UnifiedExpression::JawOpen, 0.5);
        Ok(())
    }

    // デバイスを解放する。
    async fn teardown(&mut self) -> Result<()> { Ok(()) }
}

fn main() {
    vrcftgo_sdk::serve(MyModule);
}
```

- `initialize` は、`avail` のうち実際に提供する範囲を返します。範囲外のものは `serve` が取り除きます。
- `update` で新しいデータを待つのは長くても数ミリ秒にしてください。エラーを返すとそのフレームを飛ばすだけです。
- ログは `initialize` に渡される `Logger` で出します。標準エラー出力に JSON で書かれ、ホストのログに転送されます。**標準出力には絶対に書かないでください**(プロトコル専用です)。
- panic するとプロセスが終了し、ホストがモジュールを再起動します。

ビルドして `<id>.vfgmod` をモジュールページからインストールします。

```sh
cargo build --release
# target/release/<バイナリ>(.exe) を MyModule.vfgmod としてコピー
```

## 例

[`examples/demo.rs`](examples/demo.rs) は、ハードウェアなしで動くデモモジュールです。顎・笑顔・視線・瞬きを自動で動かします。

```sh
cargo build --release --example demo
# target/release/examples/demo(.exe) を DemoRust.vfgmod としてコピー
```

## ライセンス

VRCFaceTrackingGO のほかの部分と同じく AGPL-3.0 です。このクレートを使わず、プロトコルを実装しただけのモジュールは AGPL-3.0 にする必要はありません。
