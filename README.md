## pcdeditor

[![ci](https://github.com/seqsense/webgl-go/actions/workflows/ci.yml/badge.svg)](https://github.com/seqsense/webgl-go/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

### ローカルでの実行

読み込みたい `map.pcd`, `map.yaml`, `map.png` ファイルを `./fixture` 下にコピーして
```shell
make
```
を実行し、 http://localhost:8080/ を開き、 `load` ボタンを押す。

### 操作

操作                 | 動作
-------------------- | --------------------
左クリック           | 1番目、3番目の点選択 [\*1](#footnote1)
Shift + 左クリック   | 2番目、4点目の点選択 [\*1](#footnote1)
左ドラッグ           | 視点回転
中ドラッグ           | 視点移動
Shift + 左ドラッグ   | 視点移動
Alt + 左クリック     | 隣接する点群を選択 [\*2](#footnote2)
Q/E                  | 視点回転
W/A/S/D              | 視点移動
Wheel                | 視点距離
[/]                  | 視野+/-
F10                  | 表示範囲を選択範囲に限定 (無選択での場合は解除)
F11                  | 視点の水平方向回転スナップ
F12                  | 視点の上下方向回転スナップ
Ctrl + Wheel         | 選択領域厚の拡大縮小
Ctrl + Shift + Wheel | 選択領域厚の拡大縮小 (高速)
Shift + Wheel        | 選択領域の拡大縮小
ESC                  | 選択解除
Del                  | 削除
Ctrl + Del           | 連続削除 (選択を解除しない)
F                    | 面作成
V                    | 3点目を垂直スナップ
H                    | 2, 3点目を水平スナップ
0, 1                 | ラベル設定
U, Ctrl+Z            | Undo [\*3](#footnote2)
Ctrl+C               | 選択された点群をコピー
Ctrl+V               | 点群を貼り付け

<dl>
  <dt><a id="footnote1">[1] 左クリック</a></dt><dd>
    3点を選択すると、1-2点目を結ぶ線分を1辺とし、3点目を通る長方形の領域が選択される。
    4点を選択すると、3点で選択された長方形を1面とし、4点目を通る直方体の領域が選択される。
  </dd>
  <dt><a id="footnote2">[2] Alt + 左クリック</a></dt><dd>
    透視投影モードでのみ有効。Gnome3移行以前のUbuntuでは、Alt+Win+左クリック。
  </dd>
  <dt><a id="footnote2">[3] Undo</a></dt><dd>
    点群に対する編集のみUndoバッファに記録される。選択範囲の移動・回転操作はUndo非対応。
  </dd>
</dl>

### 操作

### 選択範囲の移動・回転操作

操作               | 動作
------------------ | -------
左ドラッグ         | 水平移動 [\*2](#footnoteSelect1)
Shift + 左ドラッグ | 回転 [\*2](#footnoteSelect1)
↑/↓/←/→            | 選択領域を水平移動 (視点奥方向が↑)
PageUp/Down        | 選択領域を上下移動
Home/End           | 選択領域をYaw回転
Enter              | 貼り付けの確定

<dl>
  <dt><a id="footnoteSelect1">[1] マウス操作による移動・回転</a></dt><dd>
    マウスのボタンを離した時点でのShiftキーの押下状態で、移動か回転か決定する。
    ドラッグ中にESCキーで現在の移動・回転操作をキャンセル。
  </dd>
</dl>

### タッチデバイスでの操作

操作                              | 動作
--------------------------------- | --------------------
スワイプ                          | 視点回転
ピンチ                            | 視点距離
3点スワイプ, ダブルタップスワイプ | 視点移動
タップ                            | 1, 3点目選択
ダブルタップ                      | 2, 4点目選択
トリプルタップ                    | 隣接する点群を選択 [\*2](#footnote2)


### コマンド操作

コマンド                      | 動作
----------------------------- | -------------------------------------------------------
cursor                        | 選択中の点の一覧を表示 (`ID` `X` `Y` `Z`) [\*1](#footnoteKey1)
cursor `X` `Y` `Z`            | 新しい点(`X`, `Y`, `Z`)を選択
cursor `ID` `X` `Y` `Z`       | 指定した `ID` の選択中の点の座標を(`X`, `Y`, `Z`)に設定
unset\_cursor                 | 点の選択を解除
select\_range                 | 選択領域厚を表示 (`R`) [\*1](#footnoteKey1)
select\_range `R`             | 選択領域厚を `R` \[メートル\]に設定
snap\_v                       | 3点目を垂直スナップ
snap\_h                       | 2, 3点目を水平スナップ
translate\_cursor `X` `Y` `Z` | 選択中の点を平行移動
add\_surface                  | 面作成
add\_surface `R`              | 面作成 (点の間隔 `R` \[メートル\])
delete                        | 削除
label `L`                     | ラベル設定 (`L`)
undo                          | Undo
max\_history                  | Undo回数を表示
max\_history `A`              | Undo回数を設定 (`A`: 0-)
crop                          | 表示範囲を選択範囲に限定 (無選択での場合は解除)
map\_alpha                    | 2Dマップの透明度を表示 (`A`) [\*1](#footnoteKey1)
map\_alpha `A`                | 2Dマップの透明度を設定 (`A`: 0-1)
voxel\_grid                   | VoxelGridフィルタで点数を削減
voxel\_grid `R`               | VoxelGridフィルタで点数を削減 (voxelサイズ `R` \[メートル\])
z\_range                      | 色をつけるZ座標の範囲を表示 [\*1](#footnoteKey1)
z\_range `Min` `Max`          | 色をつけるZ座標の範囲を `Min` - `Max` \[メートル\]に設定
perspective                   | 透視投影モード
ortho                         | 正投影モード
point\_size                   | 点の表示サイズを表示 [\*1](#footnoteKey1)
point\_size `Size`            | 点の表示サイズを `Size` に設定
segmentation\_param           | セグメンテーション時の分離距離を表示 [\*1](#footnoteKey1)
segmentation\_param `D` `R`   | セグメンテーション時の分離距離を `D` \[メートル\]、適用範囲を `R` \[メートル\]に設定

<dl>
  <dt><a id="footnoteKey1">[1] 数値の表示</a></dt><dd>
    小数点以下3桁まで表示
  </dd>
</dl>

## License

This package is licensed under [Apache License Version 2.0](./LICENSE).
