Unicode true

####
## go-peerblock NSIS Installer
## Bazuje na szablonie Wails z dodaną obsługą WinDivert i deinstalacji sterownika.
##
## Build:
##   1. wails build --target windows/amd64 --nsis
##   2. Skopiuj WinDivert.dll, WinDivert64.sys do build/bin/
##   3. makensis build/windows/installer/project.nsi
##
## Lub ręcznie z własnym binarnym:
##   makensis -DARG_WAILS_AMD64_BINARY=..\..\bin\go-peerblock.exe build/windows/installer/project.nsi
####

!include "wails_tools.nsh"
!include "MUI.nsh"

# ── Metadane wersji (4 części wymagane przez Windows) ──────────────────────────
VIProductVersion "${INFO_PRODUCTVERSION}.0"
VIFileVersion    "${INFO_PRODUCTVERSION}.0"

VIAddVersionKey "CompanyName"     "${INFO_COMPANYNAME}"
VIAddVersionKey "FileDescription" "${INFO_PRODUCTNAME} Installer"
VIAddVersionKey "ProductVersion"  "${INFO_PRODUCTVERSION}"
VIAddVersionKey "FileVersion"     "${INFO_PRODUCTVERSION}"
VIAddVersionKey "LegalCopyright"  "${INFO_COPYRIGHT}"
VIAddVersionKey "ProductName"     "${INFO_PRODUCTNAME}"

# ── HiDPI ──────────────────────────────────────────────────────────────────────
ManifestDPIAware true

# ── MUI – ikony i strony ───────────────────────────────────────────────────────
!define MUI_ICON   "..\icon.ico"
!define MUI_UNICON "..\icon.ico"

!define MUI_FINISHPAGE_NOAUTOCLOSE  # Zostaw na stronie postępu żeby użytkownik zobaczył logi
!define MUI_ABORTWARNING            # Ostrzeż przy przerwaniu instalacji

# Strony instalatora
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_LICENSE "..\license.txt"
!insertmacro MUI_PAGE_COMPONENTS
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!define MUI_FINISHPAGE_RUN "$INSTDIR\${PRODUCT_EXECUTABLE}"
!define MUI_FINISHPAGE_RUN_TEXT "Uruchom go-peerblock"
!insertmacro MUI_PAGE_FINISH

# Strony deinstalatora
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

# Języki (angielski jako główny, polski jako drugi)
!insertmacro MUI_LANGUAGE "English"
!insertmacro MUI_LANGUAGE "Polish"

# ── Ogólne ustawienia ──────────────────────────────────────────────────────────
Name "${INFO_PRODUCTNAME}"
OutFile "..\..\bin\${INFO_PROJECTNAME}-${ARCH}-installer.exe"
InstallDir "$PROGRAMFILES64\${INFO_PRODUCTNAME}"
ShowInstDetails show
ShowUninstDetails show

# ── Inicjalizacja: sprawdź architekturę ────────────────────────────────────────
Function .onInit
    !insertmacro wails.checkArchitecture
FunctionEnd

# ══════════════════════════════════════════════════════════════════════════════
# SEKCJA INSTALACJI
# ══════════════════════════════════════════════════════════════════════════════
Section "go-peerblock" SecMain
    !insertmacro wails.setShellContext

    # ── 1. WebView2 Runtime (wymagany przez Wails GUI) ─────────────────────────
    # Sprawdza czy WebView2 jest zainstalowany. Jeśli nie — pobiera i instaluje
    # bootstrapper od Microsoftu (cicha instalacja).
    !insertmacro wails.webview2runtime

    # ── 2. Kopiowanie plików aplikacji ────────────────────────────────────────
    SetOutPath "$INSTDIR"

    # Pliki Wails (EXE + zasoby frontendu)
    !insertmacro wails.files

    # WinDivert — muszą być w tym samym folderze co EXE (execDir() szuka ich tam)
    File "..\..\bin\WinDivert.dll"
    File "..\..\bin\WinDivert64.sys"

    # ── 3. Instalacja sterownika WinDivert ────────────────────────────────────
    # Sterownik musi być zainstalowany PRZED pierwszym uruchomieniem aplikacji.
    # sc create może zwrócić błąd jeśli sterownik już istnieje — ignorujemy go.
    DetailPrint "Instalacja sterownika WinDivert..."

    # Zatrzymaj stary sterownik jeśli istnieje (np. reinstalacja)
    ExecWait 'sc stop WinDivert' $0
    Sleep 500

    # Usuń stary wpis sterownika
    ExecWait 'sc delete WinDivert' $0
    Sleep 500

    # Zarejestruj nowy sterownik (binPath musi wskazywać na absolutną ścieżkę)
    ExecWait 'sc create WinDivert type= kernel start= demand binPath= "$INSTDIR\WinDivert64.sys"' $0
    ${If} $0 != 0
        DetailPrint "UWAGA: sc create WinDivert zwrócił kod $0 (może być już zarejestrowany)"
    ${EndIf}

    # Uruchom sterownik
    ExecWait 'sc start WinDivert' $0
    ${If} $0 != 0
        DetailPrint "UWAGA: sc start WinDivert zwrócił kod $0 (może już działać)"
    ${EndIf}

    DetailPrint "Sterownik WinDivert zainstalowany."

        # ── 4. Skrót w menu Start (zawsze) ──────────────────────────────────────
    CreateShortcut "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"

    # ── 5. Wpis w Add/Remove Programs ─────────────────────────────────────────
    !insertmacro wails.writeUninstaller

SectionEnd

# ── Opcjonalny skrót na pulpicie (SectionIn 2 = odznaczony domyślnie) ─────────
Section "Skrót na pulpicie" SecDesktop
    SectionIn 2
    CreateShortCut "$DESKTOP\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"
SectionEnd

# ══════════════════════════════════════════════════════════════════════════════
# SEKCJA DEINSTALACJI
# ══════════════════════════════════════════════════════════════════════════════
Section "Uninstall"
    !insertmacro wails.setShellContext

    # ── 1. Zatrzymaj aplikację jeśli działa ───────────────────────────────────
    DetailPrint "Zatrzymywanie go-peerblock..."
    ExecWait 'taskkill /F /IM go-peerblock.exe' $0
    Sleep 1000

    # ── 2. Zatrzymaj i usuń sterownik WinDivert ───────────────────────────────
    # KRYTYCZNE: bez tego sterownik zostaje w systemie jako zombie
    DetailPrint "Zatrzymywanie sterownika WinDivert..."
    ExecWait 'sc stop WinDivert' $0
    Sleep 1000

    DetailPrint "Usuwanie sterownika WinDivert..."
    ExecWait 'sc delete WinDivert' $0
    ${If} $0 != 0
        DetailPrint "UWAGA: sc delete WinDivert zwrócił kod $0"
    ${EndIf}

    # ── 3. Usuń pliki aplikacji ───────────────────────────────────────────────
    # Usuń WebView2 cache (dane przeglądarki Wails)
    RMDir /r "$AppData\${PRODUCT_EXECUTABLE}"

    # Usuń folder instalacji
    RMDir /r "$INSTDIR"

    # ── 4. Usuń skróty ────────────────────────────────────────────────────────
    Delete "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk"
    Delete "$DESKTOP\${INFO_PRODUCTNAME}.lnk"

    # ── 5. Usuń wpis z Add/Remove Programs ───────────────────────────────────
    !insertmacro wails.deleteUninstaller

    # ── 6. UWAGA: dane użytkownika ────────────────────────────────────────────
    # Celowo NIE usuwamy %APPDATA%\go-peerblock\ (config.json, logi, cache list IP)
    # żeby użytkownik nie stracił konfiguracji przy reinstalacji.
    # Jeśli chcesz czyste usunięcie, odkomentuj:
    # RMDir /r "$APPDATA\go-peerblock"

SectionEnd

# ══════════════════════════════════════════════════════════════════════════════
# MAKRA POMOCNICZE
# ══════════════════════════════════════════════════════════════════════════════

# Sprawdź czy plik WinDivert.dll istnieje w bin/ przed pakowaniem
!macro CheckWinDivertFiles
    !ifndef SKIP_WINDIVERT_CHECK
        !ifdef NSIS_WIN32_MAKENSIS
            !if not FileExists "..\..\bin\WinDivert.dll"
                !error "Brak pliku WinDivert.dll w build/bin/ — skopiuj go przed budowaniem instalatora"
            !endif
            !if not FileExists "..\..\bin\WinDivert64.sys"
                !error "Brak pliku WinDivert64.sys w build/bin/ — skopiuj go przed budowaniem instalatora"
            !endif
        !endif
    !endif
!macroend
