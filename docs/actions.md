# Actions

Each line has a playback cursor. There are various actions that can be added
to a line that will manipulate the playback cursor in different ways. Each of
these actions can be added to a line with a keycombo begining with `s`.

There are also two value actions, Specific Value and Tempo Change, that carry
a number instead of manipulating the playback cursor. These are added with a
keycombo beginning with `b` instead of `s`. See [Value Actions](#value-actions)
below.

## Line Reset

`ss` — Add a Line Reset Action

When the playback cursor reaches this action, it will reset to the first beat.

## Line Reset All

`sS` — Add a Line Reset All Action

When the playback cursor reaches this action, it will reset all playback cursors
to the first beat.

## Line Bounce

`sb` — Add a Line Bounce Action

When the playback cursor reaches this action, it will go backwards until
reaching the first beat. When reaching the first beat it will change directions
and go forwards until again reaching the bounce action. Unless some other
action intervenes it will bounce back and forth between the first beat and the
bounce action.

## Line Bounce All

`sB` — Add a Line Bounce All Action

When the playback cursor reaches this action, all playback cursors will go
backwards until reaching the first beat. When each playback cursor reaches the
first beat they will change directions and go forwards until the line with the
playback cursor again reaches the Line Bounce All action. Unless some other
action intervenes each playback cursor will bounce back and forth between the
first beat and this action.

## Line Skip Beat

`sk` — Add a Line Skip Action

When the playback cursor reaches this action, it will skip this beat. This will
place this line's playback cursor ahead of other playback cursors.

## Line Delay

`sz` — Add a Line Delay Action

When the playback cursor reaches this action it will pause on the beat before
this action and play the note at that location repeatedly until either
interrupted by another action or that part or overlay changes.

## Line Reverse

`sr` — Add a Line Reverse

When the playback cursor reaches this action the playback cursor moves in the
opposite direction. When the playback cursor reaches the start of the line it
will reset to the location of the action and will continue to move backwards.

## Value Actions

Unlike the actions above, these don't affect the playback cursor. Each note
carries a number instead, consumed when the playback cursor reaches it.
They're added with a keycombo beginning with `b`, and once added, `+`/`-` or
typing digits (0-9) sets the note's value while the cursor is on it.

### Specific Value

`bv` — Add a Specific Value note

Only available on CC or Program Change lines. Sends the exact value entered,
bypassing the line's normal accent-based scaling — useful for a precise CC or
program change number that doesn't land on one of the line's accent levels.

### Tempo Change

`bT` — Add a Tempo Change note

Available on any line. When the playback cursor reaches this note, the tempo
changes to the note's value (20-300). This never changes the sequence's start
tempo — the tempo the song begins at — so playback stays repeatable across
sessions even after a Tempo Change note has fired during a previous play.

Manually adjusting the tempo with the tempo controls (`Ctrl+t`, then `+`/`-`
or digit entry) while playing moves both the start tempo and the current
playing tempo together, since that's a deliberate change to the song rather
than a momentary playback effect.

Because a Tempo Change note's effect on playback is transient, it is not
undoable. Editing the note's own stored value is undoable, like any other
note edit.
