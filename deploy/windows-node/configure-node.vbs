Option Explicit

Dim shell, fso, scriptDir, nodeExe, configPath, command

Set shell = CreateObject("WScript.Shell")
Set fso = CreateObject("Scripting.FileSystemObject")

scriptDir = fso.GetParentFolderName(WScript.ScriptFullName)
nodeExe = fso.BuildPath(scriptDir, "tinyclaw-node.exe")
configPath = shell.ExpandEnvironmentStrings("%ProgramData%") & "\TinyClawNode\config.json"

If WScript.Arguments.Count > 0 Then
    configPath = WScript.Arguments(0)
End If

If Not fso.FileExists(nodeExe) Then
    MsgBox "tinyclaw-node.exe was not found next to configure-node.vbs.", vbCritical, "TinyClaw Node"
    WScript.Quit 1
End If

command = """" & nodeExe & """ --configure --config """ & configPath & """"
shell.Run command, 0, False
