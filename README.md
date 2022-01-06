# Netgear Unmanaged Plus Switch Agent for Zabbix

NetgearのUnmanaged Plus SwitchをZabbixから監視するためのプログラム

## 使い方

GoでビルドしたバイナリをZabbixのExternal Scriptのフォルダにコピーして、Zabbixのフロントエンドから以下の設定を行う。

なお、[zabbix_sample_templates.yaml](/zabbix_sample_templates.yaml) は Zabbix 5.4にて設定を行なったテンプレートファイルである。

### メインアイテムの作成

ホストのアイテム追加で以下のアイテムを追加する。

- タイプ: 外部チェック
- キー: `zabbix-netgear-agent[-host,{HOST.CONN},-mode,get]`

### インターフェイスのディスカバリルールの作成

ディスカバリルールの追加画面で以下の通り設定する。

- タイプ: 外部チェック
- キー: `zabbix-netgear-agent[-host,{HOST.CONN},-mode,discoverif]`

`{#PORT}` にスイッチのポート番号、 `{#PATH}` にjsonのパスが入るため、アイテムのプロトタイプで以下のように設定する。

(受信バイト数の場合)

- 名前: Interface #{#PORT}: Bits received
- タイプ: 依存アイテム
- マスターアイテム: 設定したメインアイテムを指定
- キー: 任意の文字列
- データ型: 数値 (整数)
- 単位: bps
- 保存前処理
    1. JSONPath: `$.{#PATH}.recv`
    1. 乗数: 8
    1. 1秒あたりの差分

## 動作例

```
Usage of ./zabbix-netgear-agent:
  -debug
    	Debug flag
  -host string
    	host address (default "localhost")
  -mode string
    	Mode flag (default "get")
mode:
  get = Get Device Data
  discoverif = Discover Device Interfaces
  discoverdev = Discover Devices (please set host to broadcast address)
```

以下のコマンドを実行することで、指定したIPアドレスのUnmanaged Plus Switchの各種情報を含んだjsonが返却される。

```
./zabbix-netgear-agent -host (IPアドレス) -mode get
```

(jsonの例)

```json
{
    "hostname": "gs108pe",
    "ipaddress": "IPアドレス",
    "location": "",
    "macaddress": "XX:XX:XX:XX:XX:XX",
    "model": "GS108PEv3",
    "status": 0,
    "interface1": {
        "error": 0,
        "recv": 89171253301,
        "sent": 36207859737,
        "speed": 5
    },
    "interface2": {
        "error": 0,
        "recv": 794766465,
        "sent": 52562363531,
        "speed": 0
    }, ...
```

- model: 製品名
- hostname: スイッチ名
- ipaddress: 機器のIPアドレス
- macaddress: MACアドレス
- status: スイッチからの応答にエラーがあったかどうか, 0はエラーなし
- interfaceX: 各インターフェイスの情報
    - recv: 受信バイト数
    - sent: 送信バイト数
    - error: CRCエラー数
    - speed: リンク速度, 0は未接続, 5は1000M

以下のコマンドを実行することで、指定したIPアドレスのUnmanaged Plus Switchのインターフェイス一覧がZabbixのディスカバリ形式で返却される。

```
./zabbix-netgear-agent -host (IPアドレス) -mode discoverif
```

(jsonの例)

```json
{
    "data": [
        {
            "{#PATH}": "interface1",
            "{#PORT}": "1"
        },
        {
            "{#PATH}": "interface2",
            "{#PORT}": "2"
        }, ...
```

- {#PORT}: ポート番号
- {#PATH}: `-mode get` 実行時に返却されるjsonのパス

