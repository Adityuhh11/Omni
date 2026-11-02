# Scripts

Any script you put in Omni's scripts folder becomes a command in the bar, and if you want, a tool Omni AI can use. You don't
need an SDK or a manifest: a few comment lines at the top of the file are enough.

The folder is `%APPDATA%\Omni\scripts` on Windows and `~/Library/Application Support/Omni/scripts` on a Mac. Type `scripts`
in the bar, or open Settings, Omni AI, Tools, to open it.

## A first script

`hello.ps1`:

```powershell
# @omni.title Say hello
# @omni.output copy
Write-Output "Hello from $env:COMPUTERNAME"
```

Type `say` in the bar, press Enter, and the greeting is on your clipboard. Edits and new files take effect at once, with no
restart.

## The lines Omni reads

They can be in any comment style (`#`, `//`, `REM`, `--`) within the first 30 lines.

| Line | Meaning |
|---|---|
| `@omni.title Kill port` | Makes the file a command. Without it the file is ignored, so helpers and READMEs can live in the folder. |
| `@omni.keyword port` | The word that runs it with text after it: `port 3000`. Defaults to the file's name in lower case. |
| `@omni.argument a port number` | The script takes text. Choosing it from its title types the keyword for you. |
| `@omni.output copy` | What to do with what the script prints: `silent` (the default), `copy` (to the clipboard) or `paste` (typed into the program you were in). |
| `@omni.tool Ends the process listening on a TCP port.` | Offers it to Omni AI with this description. See below. |

The text arrives as the script's first argument and as the `OMNI_INPUT` environment variable. A `.bat` or `.cmd` file gets only
`%OMNI_INPUT%`, because `cmd.exe` would treat quotes in an argument as commands. Scripts run in your home folder with no console
window, for at most 60 seconds. If a script fails, the last line it wrote to its error output is shown in the bar.

**What runs a script.** On Windows: `.ps1` (PowerShell), `.py` (the `py` launcher), `.js` (Node), `.sh` (`bash`, for example
from Git for Windows), `.bat` and `.cmd`, and `.exe`. On a Mac: `.sh`, `.zsh`, `.py` (`python3`), `.js`, `.rb` and
`.applescript`, plus any executable file with no extension (its `#!` line says how it runs).

## Scripts as Omni AI tools

With an `@omni.tool` line, the script appears under Settings, Omni AI, Tools, and in the chat's **+** menu. Switch it on there.
The model is given the description, and an `input` it must fill in when the script has an `@omni.argument`. Write the
description for the model: what the script does and when it is worth calling.

A script can do anything, so by default the chat asks you before each run and shows the input. **Run without asking me first**
in Settings turns that off for one script. The model sees what the script prints, up to 4,000 characters. Scripts are never
offered to other AI apps through MCP. The tool's name comes from the file's name without its extension, so of two scripts called
`port.ps1` and `port.py` only the first is offered (both stay commands in the bar).

## Examples

`kill-port.ps1` (Windows):

```powershell
# @omni.title Kill port
# @omni.keyword port
# @omni.argument a port number
# @omni.output copy
# @omni.tool Ends the process listening on a TCP port on this computer.
$port = [int]$args[0]
$c = Get-NetTCPConnection -LocalPort $port -State Listen -ErrorAction SilentlyContinue | Select-Object -First 1
if (-not $c) { [Console]::Error.WriteLine("nothing listens on port $port"); exit 1 }
$p = Get-Process -Id $c.OwningProcess
Stop-Process -Id $p.Id -Force
Write-Output "Stopped $($p.ProcessName) ($($p.Id)) on port $port"
```

`uuid.sh` (Mac):

```sh
# @omni.title New UUID
# @omni.keyword uuid
# @omni.output paste
uuidgen | tr 'A-Z' 'a-z'
```

`wifi.ps1`, a tool that only reads:

```powershell
# @omni.title Wi-Fi status
# @omni.tool Shows the Wi-Fi network this computer is on and its signal strength.
netsh wlan show interfaces | Select-String 'SSID|Signal|State'
```
