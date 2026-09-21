# Tasky demo capture

This is a deterministic, model-driven capture of the current TUI. The Go
script sends real Bubble Tea key messages to tui.Model, writes each exact
View() result with a caption and duration, and never starts an interactive
terminal session.

From the repository root:

    go run ./scripts/demo -o scripts/demo/frames.json
    py scripts/demo/render.py --input scripts/demo/frames.json --output assets/tasky-demo.gif

The renderer uses Pillow and common system monospaced fonts when available;
fonts are not bundled with the repository. On another platform, replace py
with python3 and install Pillow first:

    python3 -m pip install Pillow

The optional MP4 share copy can be regenerated with ffmpeg:

    ffmpeg -y -i assets/tasky-demo.gif -c:v libx264 -pix_fmt yuv420p -movflags +faststart -vf "scale=1000:740:flags=lanczos,fps=10" assets/tasky-demo.mp4

frames.json and contact sheets are ignored generated files. The committed GIF
and MP4 show the scripted Tasks and Groups workflow, including completion, task
creation, all-groups mode, rename, reorder, disable/hide, reveal, and
re-enable.
