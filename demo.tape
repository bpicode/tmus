# Demo
Output images/demo.gif

Set Shell "bash"
Set FontFamily "JetBrainsMono Nerd Font"
Set Theme "Catppuccin Frappe"
Set FontSize 14
Set Width 1280
Set Height 640
Set Margin 100
Set Padding 0
Set Framerate 30
Set PlaybackSpeed 1
Set TypingSpeed 50ms

Hide
Type "export COLORTERM=truecolor"
Enter
Type "rm -rf /tmp/tmus-demo/"
Enter
Type "mkdir -p /tmp/tmus-demo/Music"
Enter
Type "cp testdata/*.mp3 /tmp/tmus-demo/Music"
Enter
Type "mkdir -p /tmp/tmus-demo/.config"
Enter
Type "export $XDG_CONFIG_HOME=/tmp/tmus-demo/.config"
Enter
Type "./tmus -d /tmp/tmus-demo/Music"
Enter
Sleep 1s
Show

Sleep 2s
Type "b"
Sleep 0.3s
Type "a"
Sleep 1s
Down
Sleep 0.5s
Type "a"
Sleep 0.5s
Down
Sleep 0.5s
Type "a"
Sleep 2s

Type "b"
Sleep 1s
Down
Enter
Sleep 4s

Type "L"
Sleep 4s
Escape
Sleep 1s

Type "i"
Sleep 4s
Escape
Sleep 2s
