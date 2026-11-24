# WinAirDrop

自宅LAN専用の小さな WebDAV サーバー。
設定は config.json の4項目だけ (host / port / root=受信フォルダ / username / password)。

## WsgiDAV 版との違い
- Python も pip も不要。単一 exe で動く
- 認証は Basic 認証 (自宅LAN内での利用を前提。Digest が必要なら追加実装する)
- WebDAV のメソッド (PUT / MKCOL / PROPFIND など) は golang.org/x/net/webdav が処理する

## ビルド (Windows)
```
go build -o WinAirDrop.exe .
```

## 起動
config.example.json を config.json にコピーして書き換えてから:
```
WinAirDrop.exe -config config.json
```
あるいは

WinAirDrop.exe をダブルクリック


## 疎通テスト (curl の例)
```
curl -u test:パスワード -X PROPFIND http://localhost:8080/
curl -u test:パスワード -X MKCOL http://localhost:8080/2026-11-25/
curl -u test:パスワード -T photo.jpg http://localhost:8080/2026-11-25/photo.jpg
```
認証なしで 401 が返ること、PUT したファイルが受信フォルダに届くことを確認してください。

## 注意
- 自宅LAN内でのみ使ってください。インターネットへ直接公開しないでください
- 初回起動時に Windows ファイアウォールの許可ダイアログが出たら許可してください
