; ============================================================================
; TradeMind AI — NSIS Windows 安装包脚本
;
; 用法（在 Windows 上，安装 NSIS 后）:
;   1. 先在 macOS/Linux 上交叉编译: make build-win
;   2. 将 build/windows/ 目录拷贝到 Windows
;   3. 右键 installer.nsi → "Compile NSIS Script"
;   或命令行: makensis installer.nsi
;
; 生成: TradeMind-Setup-v1.0.0.exe（单文件安装包）
; ============================================================================

!define APP_NAME "TradeMind AI"
!define APP_VERSION "1.0.0"
!define APP_PUBLISHER "TradeMind"
!define APP_EXE "trademind.exe"
!define APP_URL "https://trade.aisense.top"
!define APP_REGKEY "Software\TradeMind"

; ============================================================================
; 基本设置
; ============================================================================

Name "${APP_NAME}"
OutFile "TradeMind-Setup-v${APP_VERSION}.exe"
Unicode True
RequestExecutionLevel admin
ShowInstDetails show

InstallDir "$PROGRAMFILES64\TradeMind"

; ============================================================================
; 现代界面
; ============================================================================

!include "MUI2.nsh"
!include "nsDialogs.nsh"
!include "LogicLib.nsh"

!define MULTIUSER_EXECUTIONLEVEL Admin
!define MUI_ABORTWARNING
!define MUI_ICON "${NSISDIR}\Contrib\Graphics\Icons\modern-install.ico"
!define MUI_UNICON "${NSISDIR}\Contrib\Graphics\Icons\modern-uninstall.ico"

; 页面顺序
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

; 卸载页面
!insertmacro MUI_UNPAGE_WELCOME
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_UNPAGE_FINISH

; 语言
!insertmacro MUI_LANGUAGE "SimpChinese"
!insertmacro MUI_LANGUAGE "English"

; ============================================================================
; 安装区段
; ============================================================================

Section "Install" SecInstall
  SetOutPath "$INSTDIR"

  ; 主程序（已通过 go:embed 包含前端）
  File "trademind.exe"

  ; 数据目录（程序运行时自动创建，但预创建更安全）
  CreateDirectory "$INSTDIR\runtime"
  CreateDirectory "$INSTDIR\logs"

  ; 写入注册表卸载信息
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\TradeMind" \
    "DisplayName" "${APP_NAME}"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\TradeMind" \
    "UninstallString" "$\"$INSTDIR\uninstall.exe$\""
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\TradeMind" \
    "DisplayVersion" "${APP_VERSION}"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\TradeMind" \
    "Publisher" "${APP_PUBLISHER}"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\TradeMind" \
    "DisplayIcon" "$INSTDIR\${APP_EXE}"
  WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\TradeMind" \
    "NoModify" 1
  WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\TradeMind" \
    "NoRepair" 1
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\TradeMind" \
    "InstallLocation" "$INSTDIR"

  ; 注册应用
  WriteRegStr HKLM "${APP_REGKEY}" "InstallDir" "$INSTDIR"
  WriteRegStr HKLM "${APP_REGKEY}" "Version" "${APP_VERSION}"

  ; 卸载程序
  WriteUninstaller "$INSTDIR\uninstall.exe"

  ; ===== 开始菜单快捷方式 =====
  CreateDirectory "$SMPROGRAMS\${APP_NAME}"
  CreateShortcut "$SMPROGRAMS\${APP_NAME}\${APP_NAME}.lnk" \
    "$INSTDIR\${APP_EXE}" "" \
    "$INSTDIR\${APP_EXE}" 0
  CreateShortcut "$SMPROGRAMS\${APP_NAME}\卸载 ${APP_NAME}.lnk" \
    "$INSTDIR\uninstall.exe" "" \
    "$INSTDIR\uninstall.exe" 0

  ; ===== 桌面快捷方式 =====
  CreateShortcut "$DESKTOP\${APP_NAME}.lnk" \
    "$INSTDIR\${APP_EXE}" "" \
    "$INSTDIR\${APP_EXE}" 0

  ; ===== 开机自启动（可选，写入注册表 Run 键）=====
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Run" \
    "TradeMind" "$\"$INSTDIR\${APP_EXE}$\""

  SetAutoClose True
SectionEnd

; ============================================================================
; 卸载区段
; ============================================================================

Section "Uninstall"
  ; 停止运行中的程序
  nsExec::ExecToLog 'taskkill /IM "${APP_EXE}" /F'

  ; 删除程序文件
  Delete "$INSTDIR\${APP_EXE}"
  Delete "$INSTDIR\uninstall.exe"

  ; 询问是否删除数据（SQLite 数据库 + 运行时文件）
  MessageBox MB_YESNO|MB_ICONQUESTION "是否同时删除数据文件（SQLite 数据库、日志）？$\n$\n选择「是」将彻底清除所有 TradeMind 数据。" IDNO skip_data
    RMDir /r "$INSTDIR\runtime"
    RMDir /r "$INSTDIR\logs"
  skip_data:

  ; 删除目录
  RMDir "$INSTDIR"

  ; 删除快捷方式
  Delete "$DESKTOP\${APP_NAME}.lnk"
  RMDir /r "$SMPROGRAMS\${APP_NAME}"

  ; 清理注册表
  DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\TradeMind"
  DeleteRegKey HKLM "${APP_REGKEY}"
  DeleteRegValue HKCU "Software\Microsoft\Windows\CurrentVersion\Run" "TradeMind"

  SetAutoClose True
SectionEnd
