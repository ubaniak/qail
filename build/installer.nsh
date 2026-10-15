; Windows installer hooks (electron-builder picks this file up from the
; buildResources dir). Puts the bundled qail.exe on the user's PATH so one
; install gives both the tray app and the `qail` command, like the macOS
; app's "Install Command Line Tool…".
;
; The PATH edit runs in PowerShell rather than NSIS string ops: NSIS
; strings are length-limited and would truncate a long PATH. The script
; edits HKCU\Environment directly, keeping %VARS% unexpanded and the
; value's registry type, and only adds/removes our own entry. The bin
; dir reaches the script through an env var, so no path is ever quoted
; into the command line.

!macro qailEditUserPath ACTION
  System::Call 'Kernel32::SetEnvironmentVariable(t "QAIL_BIN_DIR", t "$INSTDIR\resources\bin")'
  System::Call 'Kernel32::SetEnvironmentVariable(t "QAIL_PATH_ACTION", t "${ACTION}")'
  nsExec::Exec `powershell.exe -NoProfile -NonInteractive -ExecutionPolicy Bypass -Command "$$d = $$env:QAIL_BIN_DIR; $$k = [Microsoft.Win32.Registry]::CurrentUser.CreateSubKey('Environment'); $$p = [string]$$k.GetValue('Path', '', [Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames); $$kind = if ($$k.GetValueNames() -contains 'Path') { $$k.GetValueKind('Path') } else { [Microsoft.Win32.RegistryValueKind]::ExpandString }; $$parts = @($$p.Split(';') | Where-Object { $$_ -and ($$_.TrimEnd('\') -ne $$d.TrimEnd('\')) }); if ($$env:QAIL_PATH_ACTION -eq 'add') { $$parts += $$d }; if ($$parts.Count) { $$k.SetValue('Path', ($$parts -join ';'), $$kind) } else { $$k.DeleteValue('Path', $$false) }"`
  Pop $0
  ; Tell running programs (Explorer, new terminals) that PATH changed.
  ; 0xFFFF = HWND_BROADCAST, 0x1A = WM_SETTINGCHANGE.
  SendMessage 0xFFFF 0x1A 0 "STR:Environment" /TIMEOUT=5000
!macroend

!macro customInstall
  !insertmacro qailEditUserPath "add"
!macroend

!macro customUnInstall
  !insertmacro qailEditUserPath "remove"
!macroend
