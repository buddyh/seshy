# seshy TUI showcase — drives the built ./seshy binary and screenshots key states.
# Run:  vhs tools/showcase.tape   (outputs PNGs into tools/shots/)
# NOTE: never press Enter in the picker — Enter execs the agent's resume command.

Output tools/shots/walkthrough.gif

Set Shell "zsh"
Set FontSize 16
Set Width 1400
Set Height 900
Set Padding 24
Set Margin 0

# Global picker (most recent across every repo), preview pane open by default off.
Hide
Type "./seshy all"
Enter
Show
Sleep 2s
Screenshot tools/shots/01-picker.png

# Move the cursor down to show the selected-row treatment.
Down
Sleep 300ms
Down
Sleep 300ms
Down
Sleep 600ms
Screenshot tools/shots/02-selected.png

# Open the split preview pane.
Type "p"
Sleep 1.2s
Screenshot tools/shots/03-preview.png

# Scroll a couple rows with the preview live so it refreshes.
Down
Sleep 400ms
Down
Sleep 900ms
Screenshot tools/shots/04-preview-scroll.png

# Filter / search.
Type "/"
Sleep 400ms
Type "claude"
Sleep 1s
Screenshot tools/shots/05-search.png
