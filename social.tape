# Final canvas: 1280 x 640. Keep important content inside an 80 px safe border,
# matching repository-open-graph-template.png. The player occupies
# x=80..1200, y=80..560. Tune the scene, dimensions, and font here.

Output images/social.gif

Set Shell "bash"
Set FontFamily "JetBrainsMono Nerd Font"
Set Theme "Catppuccin Frappe"
Set FontSize 16
Set Width 1280
Set Height 640
Set Padding 80
Set LineHeight 1.0
Set Framerate 10
Set TypingSpeed 0ms
Set CursorBlink false

# All player state lives in the disposable container.
Env COLORTERM "truecolor"
Env XDG_CONFIG_HOME "/tmp/tmus-social/.config"
Env XDG_CACHE_HOME "/tmp/tmus-social/.cache"
Env TMUS_TUI_FPS "10"
Env TMUS_MPRIS_ENABLED "false"
Env TMUS_IPC_SINGLE_INSTANCE "off"
Env TMUS_LYRICS_LRCLIB_ENABLED "false"

Hide
Type `./tmus "testdata/Metalguy-ca - Master of Carpets.mp3" "testdata/Britney Sheers - Maybe One More Line.mp3" "testdata/Nervana - Smells Like Cheap Spirit.mp3"`
Enter
Wait+Screen@30s /\[playing\|linear\]/
# Let metadata and the spectrum settle before capturing the playing track.
Sleep 2s
Show
Sleep 3s
Space
Sleep 1s
Space
Sleep 3s

Hide
Ctrl+C
