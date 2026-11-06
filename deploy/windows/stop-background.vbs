Option Explicit
Dim shell, fso, folder, command
Set shell = CreateObject("WScript.Shell")
Set fso = CreateObject("Scripting.FileSystemObject")
folder = fso.GetParentFolderName(WScript.ScriptFullName)
shell.CurrentDirectory = folder
command = "powershell.exe -NoProfile -ExecutionPolicy Bypass -WindowStyle Hidden -File """ & fso.BuildPath(folder, "navhub-background.ps1") & """ -Action Stop"
shell.Run command, 0, False
