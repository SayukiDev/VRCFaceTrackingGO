; Installer strings. Keep this file UTF-8 with BOM: makensis reads files
; without one in the system code page and would garble the Japanese text.

LangString DESC_SecApp ${LANG_ENGLISH} "The ${INFO_PRODUCTNAME} application."
LangString DESC_SecApp ${LANG_JAPANESE} "${INFO_PRODUCTNAME} 本体です。"

LangString NAME_SecModules ${LANG_ENGLISH} "Modules"
LangString NAME_SecModules ${LANG_JAPANESE} "モジュール"

LangString DESC_SecModules ${LANG_ENGLISH} "Tracking modules. They can also be installed or removed later from the Modules page."
LangString DESC_SecModules ${LANG_JAPANESE} "トラッキングモジュールです。後からアプリのモジュールページでもインストール・削除できます。"

; Messages of wails_tools.nsh, hooked up through its defines in project.nsi.
LangString WAILS_WIN10_REQUIRED ${LANG_ENGLISH} "This product is only supported on Windows 10 (Server 2016) and later."
LangString WAILS_WIN10_REQUIRED ${LANG_JAPANESE} "この製品は Windows 10 (Server 2016) 以降でのみサポートされています。"

LangString WAILS_ARCHITECTURE_NOT_SUPPORTED ${LANG_ENGLISH} "This product can't be installed on the current Windows architecture. Supports: ${ARCH}"
LangString WAILS_ARCHITECTURE_NOT_SUPPORTED ${LANG_JAPANESE} "この製品は現在の Windows のアーキテクチャにはインストールできません。対応: ${ARCH}"

LangString WAILS_INSTALL_WEBVIEW_DETAILPRINT ${LANG_ENGLISH} "Installing: WebView2 Runtime"
LangString WAILS_INSTALL_WEBVIEW_DETAILPRINT ${LANG_JAPANESE} "インストール中: WebView2 ランタイム"
