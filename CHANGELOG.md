# CHANGELOG

## [v0.1.0-beta.3](https://github.com/chriserin/sq/compare/v0.1.0-beta.2...v0.1.0-beta.3) (2026-07-16)

### Features

No new features.


### Fixes

* ci: golang version to 1.26 [125bf0b](https://github.com/chriserin/sq/commit/125bf0b) 


## [v0.1.0-beta.2](https://github.com/chriserin/sq/compare/v0.1.0-beta.1...v0.1.0-beta.2) (2026-02-26)

### Features

* view: Show editing/playing part [531f7d4](https://github.com/chriserin/sq/commit/531f7d4) 
* grid: Duplicate chords [8a3fa41](https://github.com/chriserin/sq/commit/8a3fa41) 
* grid: Duplicate note with > 1 beat gate [312acc3](https://github.com/chriserin/sq/commit/312acc3) 
* grid: Duplicate note, selection, line(s) [c4517d9](https://github.com/chriserin/sq/commit/c4517d9) 
* grid: Reverse line [d96c8ac](https://github.com/chriserin/sq/commit/d96c8ac) 
* grid: Euclidean rhtyhms with bu [3875232](https://github.com/chriserin/sq/commit/3875232) 
* grid: bounded loop responds to viz mode [ebdab08](https://github.com/chriserin/sq/commit/ebdab08) 
* play: Play user defined loop in overlay [2319a98](https://github.com/chriserin/sq/commit/2319a98) 
* chord: Name chord in chord view [2841531](https://github.com/chriserin/sq/commit/2841531) 
* view: Label the root note as 'Foundation' [5403009](https://github.com/chriserin/sq/commit/5403009) 

### Fixes

* play: only ensure OL if not editing [56df37b](https://github.com/chriserin/sq/commit/56df37b) 
* grid: Overlay on reverse zeros under notes [6cfb313](https://github.com/chriserin/sq/commit/6cfb313) 
* accents: clamp accent start/end correctly [5721a42](https://github.com/chriserin/sq/commit/5721a42) 
* play: ensuere midi notes played in line order [7914776](https://github.com/chriserin/sq/commit/7914776) 
* grid: Eucl hits more than 9 [74f309b](https://github.com/chriserin/sq/commit/74f309b) 
* play: actions in positions play as notes [0dc1133](https://github.com/chriserin/sq/commit/0dc1133) 
* chord: Count inversions in 2nd octave again [85c0032](https://github.com/chriserin/sq/commit/85c0032) 
* view: Remove semi intervals from chord view [b593415](https://github.com/chriserin/sq/commit/b593415) 
* grid: Update playstate when NewLine added [3c067b5](https://github.com/chriserin/sq/commit/3c067b5) 
* grid: Update playstate when adding a new line [51e6a4c](https://github.com/chriserin/sq/commit/51e6a4c) 


## [v0.1.0-beta.1](https://github.com/chriserin/sq/compare/v0.1.0-alpha.15...v0.1.0-beta.1) (2025-11-18)

### Features

* cli: Completion for instruments flag [d84d2ff](https://github.com/chriserin/sq/commit/d84d2ff) 
* cli: Completion for templates [f74fd46](https://github.com/chriserin/sq/commit/f74fd46) 
* cli: Completion for midiouts [3d44441](https://github.com/chriserin/sq/commit/3d44441) 
* cli: Completion for themes [0448b4d](https://github.com/chriserin/sq/commit/0448b4d) 
* arrangement: Allow number data entry [a51f02b](https://github.com/chriserin/sq/commit/a51f02b) 
* theory: Add names to higher intervals [831f4fe](https://github.com/chriserin/sq/commit/831f4fe) 
* Use numbers to set values [ab27a33](https://github.com/chriserin/sq/commit/ab27a33) 

### Fixes

* cli: Pass new filename into cli [0c2dec8](https://github.com/chriserin/sq/commit/0c2dec8) 
* cli: Choose non-existent theme -> default [72f04e7](https://github.com/chriserin/sq/commit/72f04e7) 
* manual: Improve manual table formatting [44ce4ea](https://github.com/chriserin/sq/commit/44ce4ea) 
* midisetup: Stop +/- when reaching boundary [4f5b801](https://github.com/chriserin/sq/commit/4f5b801) 


## [v0.1.0-alpha.15](https://github.com/chriserin/sq/compare/v0.1.0-alpha.14...v0.1.0-alpha.15) (2025-11-13)

### Features

No new features.


### Fixes

* tests: rename test files [4193842](https://github.com/chriserin/sq/commit/4193842) 


## [v0.1.0-alpha.14](https://github.com/chriserin/sq/compare/v0.1.0-alpha.13...v0.1.0-alpha.14) (2025-11-11)

### Features

No new features.

### Fixes

No bug fixes.


## [v0.1.0-alpha.13](https://github.com/chriserin/sq/compare/v0.1.0-alpha.12...v0.1.0-alpha.13) (2025-11-11)

### Features

* write config if not exists [668bc08](https://github.com/chriserin/sq/commit/668bc08) 
* Print mappings with [5019db5](https://github.com/chriserin/sq/commit/5019db5) 

### Fixes

* grid: cannot add line if it exists [cc60af3](https://github.com/chriserin/sq/commit/cc60af3) 


## [v0.1.0-alpha.12](https://github.com/chriserin/seq/compare/v0.1.0-alpha.11...v0.1.0-alpha.12) (2025-10-27)

### Features

* Move cursor to chord on chord change [3c807a3](https://github.com/chriserin/seq/commit/3c807a3) 
* Clocks on preroll to set ext tempo [02ca259](https://github.com/chriserin/seq/commit/02ca259) 
* Order same beat notes for mono modify [a8ece7f](https://github.com/chriserin/seq/commit/a8ece7f) 
* calc program value based on accent [f58b504](https://github.com/chriserin/seq/commit/f58b504) 
* Add hydrasynth instrument [51abdf8](https://github.com/chriserin/seq/commit/51abdf8) 
* Extract ppqn to constant set to 840 [bb9335e](https://github.com/chriserin/seq/commit/bb9335e) 
* Add sub phatty instrument [cef1505](https://github.com/chriserin/seq/commit/cef1505) 
* Remove skip all action [08832e2](https://github.com/chriserin/seq/commit/08832e2) 

### Fixes

* Update arp ccs for hydrasynth [3e40ddd](https://github.com/chriserin/seq/commit/3e40ddd) 
* Always set chord before change [d2017fb](https://github.com/chriserin/seq/commit/d2017fb) 
* prefer vector for variable size list [36ac1e4](https://github.com/chriserin/seq/commit/36ac1e4) 
* Find chord with matching key [551b618](https://github.com/chriserin/seq/commit/551b618) 
* +/- Sync beat loop while playing [a1c8be3](https://github.com/chriserin/seq/commit/a1c8be3) 
* Prevent on stop race conditions [311657e](https://github.com/chriserin/seq/commit/311657e) 


## [v0.1.0-alpha.11](https://github.com/chriserin/seq/compare/v0.1.0-alpha.10...v0.1.0-alpha.11) (2025-10-17)

### Features

* daw+midi play/stop [83d35fe](https://github.com/chriserin/seq/commit/83d35fe) 

### Fixes

* Mother-32 on/off ccs [1c677dd](https://github.com/chriserin/seq/commit/1c677dd) 
* Loop whole seq before last node loop [9aad138](https://github.com/chriserin/seq/commit/9aad138) 
* Ensure correct overlay for clear [f2181fc](https://github.com/chriserin/seq/commit/f2181fc) 
* Protect all map access with mutex [8144bbb](https://github.com/chriserin/seq/commit/8144bbb) 
* handle on/off correctly [1cca991](https://github.com/chriserin/seq/commit/1cca991) 
* midi panic for all notes [342f0a2](https://github.com/chriserin/seq/commit/342f0a2) 
* nil check for err [0886f86](https://github.com/chriserin/seq/commit/0886f86) 
* midi panic for all relevant channels [dbce256](https://github.com/chriserin/seq/commit/dbce256) 
* Return correct reference from find [6a9da56](https://github.com/chriserin/seq/commit/6a9da56) 
* Ensure confirmations are always shown [6179d13](https://github.com/chriserin/seq/commit/6179d13) 
* Kill noteoffs ensure note off at stop [aeeffc2](https://github.com/chriserin/seq/commit/aeeffc2) 


## [v0.1.0-alpha.10](https://github.com/chriserin/seq/compare/v0.1.0-alpha.9...v0.1.0-alpha.10) (2025-10-10)

### Features

* allow rec to continue to acc iters [6584526](https://github.com/chriserin/seq/commit/6584526) 
* automatic rec/trans determination [a3a7242](https://github.com/chriserin/seq/commit/a3a7242) 
* Write/Read blockers from file [cb0a576](https://github.com/chriserin/seq/commit/cb0a576) 
* Move Wait for devices to after go func [34c2b1f](https://github.com/chriserin/seq/commit/34c2b1f) 
* Re-connect transmitter+receiver [8ccb46c](https://github.com/chriserin/seq/commit/8ccb46c) 

### Fixes

* cursor moves when toggle hide lines [44c956b](https://github.com/chriserin/seq/commit/44c956b) 
* Remove render loop work/wait [15b8f13](https://github.com/chriserin/seq/commit/15b8f13) 
* show playCursor above gate long tail [6870ff7](https://github.com/chriserin/seq/commit/6870ff7) 
* Clear line of below notes on overlay [517d662](https://github.com/chriserin/seq/commit/517d662) 
* gridmode RotateL/R for chords at bounds [c6ca0db](https://github.com/chriserin/seq/commit/c6ca0db) 
* gridmode RotateUp/Down for chords [7aa7bee](https://github.com/chriserin/seq/commit/7aa7bee) 
* prevent error when notes != definition [1706b38](https://github.com/chriserin/seq/commit/1706b38) 
* use outport if outport flag present [937287d](https://github.com/chriserin/seq/commit/937287d) 
* keep daw connection open [0dfc434](https://github.com/chriserin/seq/commit/0dfc434) 
* Rotate note down + up w/overlays [0eb81e1](https://github.com/chriserin/seq/commit/0eb81e1) 
* get math order of operations right [eb61d7b](https://github.com/chriserin/seq/commit/eb61d7b) 
* list cmd lists the midi outputs [e18eeb5](https://github.com/chriserin/seq/commit/e18eeb5) 
* Reset time rec clock time at start [664f78a](https://github.com/chriserin/seq/commit/664f78a) 
* Donot listen while playing in standby [d843961](https://github.com/chriserin/seq/commit/d843961) 
* only listen to midi w active callback [a67b340](https://github.com/chriserin/seq/commit/a67b340) 
* Only reset transmitter once [2b11261](https://github.com/chriserin/seq/commit/2b11261) 
* Remove debug code [21cb3aa](https://github.com/chriserin/seq/commit/21cb3aa) 
* Cannot move blocked chord [b8289c8](https://github.com/chriserin/seq/commit/b8289c8) 
* Ensure connection after update [4a15270](https://github.com/chriserin/seq/commit/4a15270) 
* don't destroy connection [8ad40d0](https://github.com/chriserin/seq/commit/8ad40d0) 
* Note value increase on whole chord [74e81ed](https://github.com/chriserin/seq/commit/74e81ed) 
* blockers should move with blocked chord [41ee54a](https://github.com/chriserin/seq/commit/41ee54a) 
* don't set overlaykey if focused [1db888d](https://github.com/chriserin/seq/commit/1db888d) 
* move selection on last/first line [f527d3e](https://github.com/chriserin/seq/commit/f527d3e) 
* Ratchet edit view render bug [a1a1640](https://github.com/chriserin/seq/commit/a1a1640) 
* No undo unless key is different [35c9370](https://github.com/chriserin/seq/commit/35c9370) 
* Driver must wait for devices loop [5782ee9](https://github.com/chriserin/seq/commit/5782ee9) 


## [v0.1.0-alpha.9](https://github.com/chriserin/seq/compare/v0.1.0-alpha.8...v0.1.0-alpha.9) (2025-09-25)

### Features

* update midi devices as added/removed [7645eb1](https://github.com/chriserin/seq/commit/7645eb1) 
* expose chord/mono mode in same place [e91e5a5](https://github.com/chriserin/seq/commit/e91e5a5) 
* Fill open spaces with numpattern [499162b](https://github.com/chriserin/seq/commit/499162b) 
* Visual mode line wise [9f2eeb6](https://github.com/chriserin/seq/commit/9f2eeb6) 
* Mono Note pattern behaviour [ef9aba0](https://github.com/chriserin/seq/commit/ef9aba0) 
* Mono Mode change pattern behaviour [2ef267c](https://github.com/chriserin/seq/commit/2ef267c) 
* Toggle whether to transmit or not [5958e25](https://github.com/chriserin/seq/commit/5958e25) 
* mute/unmute all [31f9f66](https://github.com/chriserin/seq/commit/31f9f66) 

### Fixes

* On linux get devices before loop start [9b277ef](https://github.com/chriserin/seq/commit/9b277ef) 
* Get midi device for daw [98aea23](https://github.com/chriserin/seq/commit/98aea23) 
* receiver uses tick interval [5459f49](https://github.com/chriserin/seq/commit/5459f49) 
* s/off/on for midi mistake [7965c98](https://github.com/chriserin/seq/commit/7965c98) 
* Visual rotate up/down [7ab2598](https://github.com/chriserin/seq/commit/7ab2598) 
* long gate notes have correct bg [df81881](https://github.com/chriserin/seq/commit/df81881) 
* Find better spot for mono indication [53801b3](https://github.com/chriserin/seq/commit/53801b3) 
* fix ratchets with by fixing notereg [efd6e57](https://github.com/chriserin/seq/commit/efd6e57) 
* preroll applies to receivers as well [571ee6e](https://github.com/chriserin/seq/commit/571ee6e) 
* transmitter pulse adjustment [4ea3a54](https://github.com/chriserin/seq/commit/4ea3a54) 


## [v0.1.0-alpha.8](https://github.com/chriserin/seq/compare/v0.1.0-alpha.7...v0.1.0-alpha.8) (2025-09-08)

### Features

- Clear All Parts/Overlays [8e50f10](https://github.com/chriserin/seq/commit/8e50f10)
- Save As (ctrl+w) [46f78c9](https://github.com/chriserin/seq/commit/46f78c9)

### Fixes

- Allow bigger gate values [528e560](https://github.com/chriserin/seq/commit/528e560)
- set to default loop state on stop [e5f4f4f](https://github.com/chriserin/seq/commit/e5f4f4f)
- Loop whole song [7f18382](https://github.com/chriserin/seq/commit/7f18382)
- Escape from mode before save [03e4a16](https://github.com/chriserin/seq/commit/03e4a16)
- CC msg on/off 1 or 0 [d081fad](https://github.com/chriserin/seq/commit/d081fad)
- SyncBeatLoop on linestate change [3c0bf80](https://github.com/chriserin/seq/commit/3c0bf80)
- send stop on stop when receiver [650367d](https://github.com/chriserin/seq/commit/650367d)

## [v0.1.0-alpha.7](https://github.com/chriserin/seq/compare/v0.1.0-alpha.6...v0.1.0-alpha.7) (2025-09-04)

### Features

- Intro pattern mode note variation [4cd1a45](https://github.com/chriserin/seq/commit/4cd1a45)
- modify existing key [68e59c9](https://github.com/chriserin/seq/commit/68e59c9)
- Display overlay loop indicator in side [35acc53](https://github.com/chriserin/seq/commit/35acc53)
- Undo remove/new overlay [3041a9b](https://github.com/chriserin/seq/commit/3041a9b)
- Remove Overlay [d2e9135](https://github.com/chriserin/seq/commit/d2e9135)
- Display connected indicator for rcvr [e81621d](https://github.com/chriserin/seq/commit/e81621d)
- Inc/Dec all notes/channels [41c286c](https://github.com/chriserin/seq/commit/41c286c)
- Send loopMode to receiver with SPP msg [b02da07](https://github.com/chriserin/seq/commit/b02da07)

### Fixes

- New Line determines note val better [3ec2b6c](https://github.com/chriserin/seq/commit/3ec2b6c)
- CC msgs are sent before Note msgs [b9fa645](https://github.com/chriserin/seq/commit/b9fa645)
- Print to Stderr [f01669b](https://github.com/chriserin/seq/commit/f01669b)
- cache the wrapped sendFn not the naked [4b29bd1](https://github.com/chriserin/seq/commit/4b29bd1)
- line names for CC and PC [cdc8fc8](https://github.com/chriserin/seq/commit/cdc8fc8)
- don't combine standard and instr CCs [bc6c541](https://github.com/chriserin/seq/commit/bc6c541)
- Default width to 1 [844be84](https://github.com/chriserin/seq/commit/844be84)
- clear looped arr on play [6e29d5e](https://github.com/chriserin/seq/commit/6e29d5e)
- don't send final pulse for trans stop [a76f715](https://github.com/chriserin/seq/commit/a76f715)
- don't stop at end if receiver mode [05ea505](https://github.com/chriserin/seq/commit/05ea505)
- ui should follow msg from timing [1a11ad4](https://github.com/chriserin/seq/commit/1a11ad4)
- cache acquired send fn [848a444](https://github.com/chriserin/seq/commit/848a444)
- prevent runaway beat loop [bd24053](https://github.com/chriserin/seq/commit/bd24053)
- sync receiver to beat loop [42cd49e](https://github.com/chriserin/seq/commit/42cd49e)
- Add beat channel to transmitter loop [ecb8a5b](https://github.com/chriserin/seq/commit/ecb8a5b)

## [v0.1.0-alpha.6](https://github.com/chriserin/seq/compare/v0.1.0-alpha.5...v0.1.0-alpha.6) (2025-08-19)

### Features

- cursor on valid line when hiding lines [ee2de85](https://github.com/chriserin/seq/commit/ee2de85)
- nav with hidden lines [452c20e](https://github.com/chriserin/seq/commit/452c20e)
- choose midi outport at the command line [4937915](https://github.com/chriserin/seq/commit/4937915)
- Start delay for record latency compensation [8c122f3](https://github.com/chriserin/seq/commit/8c122f3)
- move key combo view to below side [7bb38d9](https://github.com/chriserin/seq/commit/7bb38d9)
- Long gates everywhere [144d6dc](https://github.com/chriserin/seq/commit/144d6dc)
- Midi panic (mapping: bp) [a20d298](https://github.com/chriserin/seq/commit/a20d298)

### Fixes

- Use a no-bizlogic fn for read add note [209d004](https://github.com/chriserin/seq/commit/209d004)
- add back new line char in view [9d322d5](https://github.com/chriserin/seq/commit/9d322d5)
- Switch to overlay with overlay edit [3d8b034](https://github.com/chriserin/seq/commit/3d8b034)
- Accents end allow down to 0 [479164f](https://github.com/chriserin/seq/commit/479164f)
- pattern mode - accent on chords [88fa1e2](https://github.com/chriserin/seq/commit/88fa1e2)
- New sequence gets new undo state [223cb43](https://github.com/chriserin/seq/commit/223cb43)
- error on stop in standalone mode [e8e56c4](https://github.com/chriserin/seq/commit/e8e56c4)
- ensure first beat played for record play [ee320fe](https://github.com/chriserin/seq/commit/ee320fe)
- Escape selection before focus [cf6c3f2](https://github.com/chriserin/seq/commit/cf6c3f2)
- Inc/Dec from arr view [2762db7](https://github.com/chriserin/seq/commit/2762db7)
- Play Overlay Loop w/Arr Focus [0045ae7](https://github.com/chriserin/seq/commit/0045ae7)
- Reset depth after cursor move on play [d879b0e](https://github.com/chriserin/seq/commit/d879b0e)
- Find substring of virtual midi outs for DAW [5cee707](https://github.com/chriserin/seq/commit/5cee707)
- Loop Overlays loops correct overlay [d6c054d](https://github.com/chriserin/seq/commit/d6c054d)

## [v0.1.0-alpha.5](https://github.com/chriserin/seq/compare/v0.1.0-alpha.4...v0.1.0-alpha.5) (2025-08-08)

### Features

- Reconfigure accent ui for ++intuitive [ce64da1](https://github.com/chriserin/seq/commit/ce64da1)
- Only complete files with seq ext [52abd72](https://github.com/chriserin/seq/commit/52abd72)
- Wait until DAW gets a chance to record [ab003bc](https://github.com/chriserin/seq/commit/ab003bc)
- hide lines without notes [aec0bc5](https://github.com/chriserin/seq/commit/aec0bc5)

### Fixes

- Send Stop message in own process [f66f572](https://github.com/chriserin/seq/commit/f66f572)
- Reset depth after play move [5389336](https://github.com/chriserin/seq/commit/5389336)
- Improve recording dest err message [a15dad3](https://github.com/chriserin/seq/commit/a15dad3)
- Group as second sibling need reset iterations [995287b](https://github.com/chriserin/seq/commit/995287b)
- Exit from arrView focus with Enter [a4c9ec4](https://github.com/chriserin/seq/commit/a4c9ec4)

## [v0.1.0-alpha.4](https://github.com/chriserin/seq/compare/v0.1.0-alpha.3...v0.1.0-alpha.4) (2025-08-01)

### Features

- Capture panic in View function [e24be3c](https://github.com/chriserin/seq/commit/e24be3c)
- Look in standard dirs for init.lua file [c630d83](https://github.com/chriserin/seq/commit/c630d83)

### Fixes

- initial state for file [77fc971](https://github.com/chriserin/seq/commit/77fc971)
- first digit application false on ok enter [37e28d0](https://github.com/chriserin/seq/commit/37e28d0)
- escape from overlay key edit [a045ae2](https://github.com/chriserin/seq/commit/a045ae2)
- escape from arr view [c75ac3d](https://github.com/chriserin/seq/commit/c75ac3d)
- arr cursor when grouping groups [eb9e383](https://github.com/chriserin/seq/commit/eb9e383)
- copy cursor for arr undo [3de2392](https://github.com/chriserin/seq/commit/3de2392)
- Only one focus at a time [78d081c](https://github.com/chriserin/seq/commit/78d081c)
- Error for multiple inbetween keys [b3f1bd0](https://github.com/chriserin/seq/commit/b3f1bd0)
- Specific Value Undo [2c56ebe](https://github.com/chriserin/seq/commit/2c56ebe)
- Spacing issue on edit key [074410b](https://github.com/chriserin/seq/commit/074410b)
- ensure length of line name [a5e8415](https://github.com/chriserin/seq/commit/a5e8415)

## [v0.1.0-alpha.4](https://github.com/chriserin/seq/compare/v0.1.0-alpha.3...v0.1.0-alpha.4) (2025-08-01)

### Features

- Capture panic in View function [e24be3c](https://github.com/chriserin/seq/commit/e24be3c)
- Look in standard dirs for init.lua file [c630d83](https://github.com/chriserin/seq/commit/c630d83)

### Fixes

- initial state for file [77fc971](https://github.com/chriserin/seq/commit/77fc971)
- first digit application false on ok enter [37e28d0](https://github.com/chriserin/seq/commit/37e28d0)
- escape from overlay key edit [a045ae2](https://github.com/chriserin/seq/commit/a045ae2)
- escape from arr view [c75ac3d](https://github.com/chriserin/seq/commit/c75ac3d)
- arr cursor when grouping groups [eb9e383](https://github.com/chriserin/seq/commit/eb9e383)
- copy cursor for arr undo [3de2392](https://github.com/chriserin/seq/commit/3de2392)
- Only one focus at a time [78d081c](https://github.com/chriserin/seq/commit/78d081c)
- Error for multiple inbetween keys [b3f1bd0](https://github.com/chriserin/seq/commit/b3f1bd0)
- Specific Value Undo [2c56ebe](https://github.com/chriserin/seq/commit/2c56ebe)
- Spacing issue on edit key [074410b](https://github.com/chriserin/seq/commit/074410b)
- ensure length of line name [a5e8415](https://github.com/chriserin/seq/commit/a5e8415)

## [v0.1.0-alpha.4](https://github.com/chriserin/seq/compare/v0.1.0-alpha.3...v0.1.0-alpha.4) (2025-08-01)

### Features

- Capture panic in View function [e24be3c](https://github.com/chriserin/seq/commit/e24be3c)
- Look in standard dirs for init.lua file [c630d83](https://github.com/chriserin/seq/commit/c630d83)

### Fixes

- first digit application false on ok enter [37e28d0](https://github.com/chriserin/seq/commit/37e28d0)
- escape from overlay key edit [a045ae2](https://github.com/chriserin/seq/commit/a045ae2)
- escape from arr view [c75ac3d](https://github.com/chriserin/seq/commit/c75ac3d)
- arr cursor when grouping groups [eb9e383](https://github.com/chriserin/seq/commit/eb9e383)
- copy cursor for arr undo [3de2392](https://github.com/chriserin/seq/commit/3de2392)
- Only one focus at a time [78d081c](https://github.com/chriserin/seq/commit/78d081c)
- Error for multiple inbetween keys [b3f1bd0](https://github.com/chriserin/seq/commit/b3f1bd0)
- Specific Value Undo [2c56ebe](https://github.com/chriserin/seq/commit/2c56ebe)
- Spacing issue on edit key [074410b](https://github.com/chriserin/seq/commit/074410b)
- ensure length of line name [a5e8415](https://github.com/chriserin/seq/commit/a5e8415)

## [v0.1.0-alpha.3](https://github.com/chriserin/seq/compare/v0.1.0-alpha.2...v0.1.0-alpha.3) (2025-07-25)

### Features

- Allow new line when more than 16 lines [328000e](https://github.com/chriserin/seq/commit/328000e)
- Reload file confirmation [0145d14](https://github.com/chriserin/seq/commit/0145d14)
- Group groups and focus group after creation [911ac56](https://github.com/chriserin/seq/commit/911ac56)

### Fixes

- Next/Prev Section work with arr focus [a772452](https://github.com/chriserin/seq/commit/a772452)
- Tempo responds to undo/redo [2fa3604](https://github.com/chriserin/seq/commit/2fa3604)
- Ensure overlay after overlaykey enter [8073e01](https://github.com/chriserin/seq/commit/8073e01)
- Reset overlay key on new sequence [aaacc42](https://github.com/chriserin/seq/commit/aaacc42)
- escape from filename prompt [4cb63cf](https://github.com/chriserin/seq/commit/4cb63cf)
- save to ctrl+s. setup to ctrl+d [199e54b](https://github.com/chriserin/seq/commit/199e54b)

## [v0.1.0-alpha.3](https://github.com/chriserin/seq/compare/v0.1.0-alpha.2...v0.1.0-alpha.3) (2025-07-25)

### Features

- Allow new line when more than 16 lines [328000e](https://github.com/chriserin/seq/commit/328000e)
- Reload file confirmation [0145d14](https://github.com/chriserin/seq/commit/0145d14)
- Group groups and focus group after creation [911ac56](https://github.com/chriserin/seq/commit/911ac56)

### Fixes

- Next/Prev Section work with arr focus [a772452](https://github.com/chriserin/seq/commit/a772452)
- Tempo responds to undo/redo [2fa3604](https://github.com/chriserin/seq/commit/2fa3604)
- Ensure overlay after overlaykey enter [8073e01](https://github.com/chriserin/seq/commit/8073e01)
- Reset overlay key on new sequence [aaacc42](https://github.com/chriserin/seq/commit/aaacc42)
- escape from filename prompt [4cb63cf](https://github.com/chriserin/seq/commit/4cb63cf)
- save to ctrl+s. setup to ctrl+d [199e54b](https://github.com/chriserin/seq/commit/199e54b)

## [v0.1.0-alpha.3](https://github.com/chriserin/seq/compare/v0.1.0-alpha.2...v0.1.0-alpha.3) (2025-07-25)

### Features

- Allow new line when more than 16 lines [328000e](https://github.com/chriserin/seq/commit/328000e)
- Reload file confirmation [0145d14](https://github.com/chriserin/seq/commit/0145d14)
- Group groups and focus group after creation [911ac56](https://github.com/chriserin/seq/commit/911ac56)

### Fixes

- Next/Prev Section work with arr focus [a772452](https://github.com/chriserin/seq/commit/a772452)
- Tempo responds to undo/redo [2fa3604](https://github.com/chriserin/seq/commit/2fa3604)
- Ensure overlay after overlaykey enter [8073e01](https://github.com/chriserin/seq/commit/8073e01)
- Reset overlay key on new sequence [aaacc42](https://github.com/chriserin/seq/commit/aaacc42)
- escape from filename prompt [4cb63cf](https://github.com/chriserin/seq/commit/4cb63cf)
- save to ctrl+s. setup to ctrl+d [199e54b](https://github.com/chriserin/seq/commit/199e54b)

## [v0.1.0-alpha.3](https://github.com/chriserin/seq/compare/v0.1.0-alpha.2...v0.1.0-alpha.3) (2025-07-25)

### Features

- Allow new line when more than 16 lines [328000e](https://github.com/chriserin/seq/commit/328000e)
- Reload file confirmation [0145d14](https://github.com/chriserin/seq/commit/0145d14)
- Group groups and focus group after creation [911ac56](https://github.com/chriserin/seq/commit/911ac56)

### Fixes

- Next/Prev Section work with arr focus [a772452](https://github.com/chriserin/seq/commit/a772452)
- Tempo responds to undo/redo [2fa3604](https://github.com/chriserin/seq/commit/2fa3604)
- Ensure overlay after overlaykey enter [8073e01](https://github.com/chriserin/seq/commit/8073e01)
- Reset overlay key on new sequence [aaacc42](https://github.com/chriserin/seq/commit/aaacc42)
- escape from filename prompt [4cb63cf](https://github.com/chriserin/seq/commit/4cb63cf)
- save to ctrl+s. setup to ctrl+d [199e54b](https://github.com/chriserin/seq/commit/199e54b)

## [v0.1.0-alpha.2](https://github.com/chriserin/seq/compare/v0.1.0-alpha.1...v0.1.0-alpha.2) (2025-07-22)

### Features

- Add actions for BounceAll and SkipAll [6bb44d9](https://github.com/chriserin/seq/commit/6bb44d9)
- Omit Octave from chord [b66958f](https://github.com/chriserin/seq/commit/b66958f)

### Fixes

- Rearrange arrangement view attributes [90290e0](https://github.com/chriserin/seq/commit/90290e0)
- Gate Decrease/Increse g/G [94fc6f5](https://github.com/chriserin/seq/commit/94fc6f5)
- Cursor should stay seen when reducing beats [769b6a2](https://github.com/chriserin/seq/commit/769b6a2)
- Default template should have ascending notes [7d8bcd1](https://github.com/chriserin/seq/commit/7d8bcd1)
