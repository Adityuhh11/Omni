# Design

The visual system of the launcher panel, recorded from the built page (`frontend/src`) on 2026-10-15. Tokens live in
`frontend/src/theme.css`.

## World

Pure black and chrome. The plate is flat `#000`, like an OLED screen or black paint on a sports car: no gradient,
no texture (a brushed grain, a band of light, a graphite gradient and frosted glass were all tried on 2026-10-16 and
dropped; they read as stray lines or turned the plate grey). The premium comes from a lit top rim and a hairline edge, and
from the ring. It is opaque, so it stays black whatever is behind the panel. The interface has one accent hue,
ember orange until the person picks another (emphasis is weight and brightness); the only other colour is the ring, a metal that changes every time the panel closes.

Three choices in Settings, General change this, and every window follows at once (`ui:look`):

- **Glass theme**: Liquid Glass, the owner's pick of the two glass studies (2026-11-26; the first Glass, a frosted plate with floating
  lights, is gone). A clear plate over a blur of what is behind the window. On a Mac the system's `NSGlassEffectView` (macOS 26+) under the
  bar, its quick-open tiles and the card windows, the plate at 20% in the bar and 60% in Settings and the cards, where text must stay
  readable over busy windows. On Windows the compositor's blur (`internal/glass`), which has no tint of its own: the plate at 50% with a
  faint text shadow. Where the system cannot blur (Windows' energy saver: nearly opaque) or the person asked for less transparency
  (opaque), the plate is dense again. What floats inside a card is denser and blurs what it covers (`--glass-raised`, `--glass-blur`).
  No ring: the bar is slimmer (50 px), its edge a hairline in the accent (`[ui] bar_border` turns it off), and its rim lit brightest at
  the top left, the angle turning slowly. Motion is one sheet across the bar and the tiles: three accent blobs drift and change shape, a
  soft band of light glides through every few seconds, and a faint light follows the pointer; it grows brighter while the bar has the
  keyboard and while you type, and stops for reduced motion. The tiles do not grow on hover, which would pull them off the glass
  behind them.
- **Burgundy** (`#1a0609`), a deep dark red among the dark themes, with its own rings (garnet, ruby, wine, rose gold, ember).
- **The ring** can be switched off: the bar and the chat's box then have a plain hairline edge and the plate fills the card.
- **An accent** takes the place of the chrome where it marks something: focus, the caret, the bar's icon and edge, filled buttons,
  switches, the text selection, and a little of it in the selected row. Ember orange (`#f2601d`) is the default, chosen by the owner on
  2026-11-26; blue, violet, green, amber and rose are the other presets, "Chrome" keeps the interface without a hue, and any colour
  can be picked (`[ui] accent` holds a preset's name or `#rrggbb`). A light shade on the dark themes, a deep one on the light themes;
  a colour of the person's own is used as picked, with dark or white text on filled buttons, whichever reads better.

## Shape and motion

- **Capsule to card.** At rest the bar is a 680 x 65 capsule (radius 32.5 px, a perfect semicircle end). Results grow it
  into a card of the same width and the same corner radius; only the height animates (240 ms, exponential ease-out), and the
  list fades in and drops 6 px as it appears (not on every keystroke). Opening, the card is there at once and its content rises
  5 px into it (200 ms); the notes, chat, settings and calendar pages rise 4 px into their cards (110 ms), and Settings' sections rise
  4 px into the page when chosen (100 ms): motion here is short, because a window that is slow to appear is worse than one with
  no animation. The settings and calendar windows are built ahead of time, hidden, once they have been opened (`warm` in
  `internal/settingsui/window.go`), so every opening after the first shows a page that is already loaded: the card is up in about
  50 ms instead of 200. Where Windows' energy saver stops the blur the glass is the nearly opaque plate, and Settings says so. Only the
  content moves: the glass is the system's and cannot be animated. All of it fills backwards only, so no opacity or transform
  stays on the text, and none of it with reduced motion.
  The window around it is clear and transparent (`BackgroundTypeTransparent`), 20 px wider on each side and 12 px above
  and 40 px below the card, for the shadow. The shadow (`--shadow`) must fit inside those margins (blur minus spread under
  the margin on every side); a larger one is cut off by the window edge as a hard straight line, which showed over white pages.
- **The ring.** Our own WebGL shader (`lib/ringShader.ts`, run by Paper's vanilla `ShaderMount`) draws on the whole card; the
  plate sits 2.5 px inside it, so only the rim shows. The light is laid along the distance travelled
  round the card's outline, not across it as Paper's liquid-metal stripes do (on a thin rim those seemed to come and go from
  different sides): it circulates clockwise only, two fronts on opposite sides keep the whole outline lit, and a slow wobble in
  the phase makes the bands breathe; the colour channels sit a hair apart for coloured fringes. The metal is one of ten in `lib/palettes.ts` (chrome, copper, ember,
  gold, ice, violet, emerald, rose, and two two-colour ones, sunset and aurora, whose highlights are another hue). A new one is
  chosen, never the same twice in a row, at the moment the panel loses the keyboard, so the colour never changes while it is
  looked at. Speed settles at 0.6, or 1.5 while you type or a
  search is in flight and for 0.7 s after; every change eases in over about 350 ms (a ring that lurches with each keystroke
  looks broken), and the ring starts from rest each time the panel opens. It stops completely, with no animation frames,
  when the panel does not have the keyboard or the user prefers reduced motion. Rendered at 1 pixel per CSS pixel, 700k pixels at most.
- **Rows.** Four kinds share one list, in this order: a quicklink the text triggers (`gh rust`), apps (their own icon on the
  plate, name, "Application" at the right), files (tile, name, folder, the matching chunk) and last the web search ("Search the
  web for ...", a globe tile), which is the fallback and always the last row. When apps and files both show, each group has a
  small label. A strong app match is the selected row, so Enter launches it. Beside apps, files without a confident match
  are not shown. Alt+Enter searches the web for the text from any row.
- **Mode chip.** The chip at the right of the bar says what a search covers: nothing for everything, "Apps", "Files", "Web" or "Tags" when Tab
  chose it (Shift+Tab backwards, a prefix such as `app:` does the same), or the file kind a `code:` prefix chose. Esc leaves the
  mode, then clears the text, then closes; closing the bar always returns to everything.
- **Tags.** Ctrl+T on a file, an app or a tagged file opens the tag picker, the action panel's own list with the tags that
  exist (a check marks those the row has) and, for text typed that is not a tag yet, "Add the tag ...": choosing a tag
  toggles it, and the panel stays open to add more. A row shows its tags as small `#tag` chips before its kind. `#work` (or
  the Tags chip) lists everything tagged work, apps and files together, `#work budget` narrows it by name, `#` alone or a
  partial name lists the tags; Enter on a tag row fills in `#name ` to show what carries it.
- **Action panel.** Ctrl+K opens a small panel above the footer with what can be done with the selected row, filtered by typing,
  each action with the key that does it directly. It acts on the row that was selected when it opened and owns the keyboard
  while open (Esc or Ctrl+K closes it). The card keeps room for it however few rows there are.
- **Search glyph.** A hairline lens (1.3 px stroke), a short detached handle and a small glint inside the lens. The glint
  is the only moving part: still at the top left, circling once every 1.1 s while a search is in flight.
- **Loader.** A turning chrome orb and the words under way, each letter breathing a beat after the last (90 ms stagger,
  1.8 s). Used for "Warming up" and "Indexing". Everything animated pauses via `html[data-active="false"]`.
- **Content.** The list fades in 60 ms after the card starts growing; old results dim to 55% while a new search is in
  flight; the window grows at once and shrinks 280 ms after the card. A list that has just appeared brings its first rows
  in 25 ms apart (never on a keystroke). The card itself does not overshoot: the window is already its final height, so
  an overshoot would cut the shadow.
- **Selection and press.** One highlight lies under the selected row and slides to the next (220 ms, `--spring`, a small
  overshoot); a new answer places it at once. Pressing a row darkens the highlight and sinks the icon tile to 90%, which
  springs back on release. Only the tile scales, never text. None of it with reduced motion.

## Color

| Token | Value | Use |
|---|---|---|
| `--plate` | `#000` | the plate, flat |
| `--glass` | the plate; plate at 55% in Glass | the cards' fill |
| `--card-rim`, `--card-edge` | 16%, 10% of ink (24%, 14% in Glass) | the cards' lit top rim and hairline edge |
| `--bar-rim`, `--bar-edge` | 13%, 4.5% (22%, 8% in Glass; edge 12% without the ring) | the bar's rim and edge |
| `--well`, `--well-edge` | `#0d0d0e`, 0% (12% without the ring) | the chat's box inside its ring |
| `--ring` | 2.5px (0 without the ring) | how much of the card the ring takes |
| `--glass-raised` + `--glass-blur` | the plate, no blur; plate at 84% and the page's own blur in Glass | what floats inside a card: the action panel, an event's details, the chat's history |
| `--lift` | `rgba(255,255,255,.075)` | selected row |
| `--tile` | `rgba(255,255,255,.055)` | icon tiles, count and key chips |
| `--line` | `rgba(255,255,255,.07)` | hairlines between bar, list and hints |
| `--text` | `#eceef2` | names, input |
| `--text-2` | `#aaacb2` | snippets, notes |
| `--text-3` | `#8a8c93` | placeholder, paths, locations (5:1 on the plate) |
| `--chrome` | `#e6e8ec` | caret, focused glyph, selected tile |
| `--danger` | `#e0877f` | problems only |

## Type

The system's own typeface everywhere (SF Pro on a Mac, Segoe UI Variable on Windows; Cascadia Mono or SF Mono for paths, locations,
counts and key caps, which are measurement and data only), chosen by the owner on 2026-11-26 from Glass's look in place of Geist and
Sora. Nothing is bundled or loaded from a network, and the OS picks the Text or Display cut by size. The bar's input is 16.5 px / 400,
tracking -0.01em. Scale: names 14.5 / 520,
snippets 12.5, notes 12.5, paths 11.5 mono, locations and chips 11 mono, hints 11.5.

## Components

- **Bar:** 60 px tall, search glyph (drawn, 1.6 stroke), input, loader chip at the right.
- **Row:** 44 px tile (kind icon, `PDF` in mono for PDFs, or the image's thumbnail), name with the matched words in
  weight 620 and white, the last two folders as a front-trimmed path, up to two lines of snippet with its location, `+N`
  chip for further matches. Selected: lifted plate with a 1 px top highlight. Expanded matches indent under the file.
- **Scope chip:** `Code`, `Images`, `PDFs` or `Documents` at the right of the bar when the query began with `code:`, `img:`,
  `pdf:` or `notes:` (a mono chip on `--tile`). Typing only the scope says what to do next.
- **Hints:** one row of key caps (`↑↓`, `Enter`, `Ctrl+Enter`, `→` when the file has more matches, `Esc`), with the
  index status at the right while indexing.
- **Notes:** one line each above the results (hotkey taken, engine failed, nothing matched, no strong match).
- **Settings pages:** a page is titled groups (`.group-title`, `.group`) of rows (`.item`): the name on the left, its control on the
  right, a line under the name (`.sub`) only when it changes a decision (what is sent, what is deleted, how much space), a hint for
  the whole group under it (`.group-note`). On and off is a switch (`input.switch`), never a checkbox. The sidebar is in three parts
  (General; Features; Intelligence) and About. The shared styles are in `lib/settings/forms.css`.
- **Calendar window:** a week grid (an hour is 46 px), today's date in a chrome disc and a thin red line at the time now, events
  as tinted blocks with a 2 px bar of their calendar's colour. The calendar colours are the one exception to "no accent hue":
  they tell calendars apart, so they are data, and they are muted metals (`lib/calendar.ts`) from the ring's family.
  An event opens a panel at the bottom right: the title beside a 3 px bar of its colour, then time (with "in 12 min" or "Now"),
  place and calendar as icon rows, a wide Join button tinted with the event's colour that names the service (Zoom, Google Meet,
  Teams, Webex), and the notes rendered as markdown (HTML from Google or Outlook is turned into markdown first).
- **Map window:** night (`#0e0f12`) whatever the look, so every hue reads the same. Each chunk is a solid dot with a thin dark rim,
  coloured by its folder or its file type: the eight categorical hues of the dark data palette in a fixed order (validated
  on the night for colour-blind separation and contrast), grey for a ninth group on, "other" and a folder no longer
  searched; the legend (bottom left, a group hides on a click) and the hover card name it, so colour never carries it
  alone. By age it is the upper part of viridis from the oldest file to the newest, placed by rank so a few old files do not
  flatten the rest, with the dates under the ramp. Named regions float over the map in the ink with a night halo, larger for
  a larger region, fading toward the far side; L or the Labels button hides them. Clicking a dot lights every piece of its
  file and joins them with a thin white line in reading order; a card at the top right names the file and opens it. Dots are opaque and
  depth-tested: a near dot covers a far one, so a dense region stays a crowd of dots, not a haze of light. The camera eases
  toward where input puts it (never jumps),
  turns slowly by itself after 4 s untouched in 3D, and draws no frames when nothing moves. 3D and 2D are one camera: 2D
  squeezes the third axis to nothing and looks straight down it, animated. A query lands as a white pulsing ring among its
  hits, which grow and light up while the rest dim. Reduced motion: no turning, no easing, no pulse.
- **Browser surfaces:** themed selection, caret and scrollbar; no focus ring on rows (selection is the focus).

## Rules

- Nothing animated may run while the panel is closed; nothing may delay the hotkey path (the opening rise starts once the panel
  is shown and focused, and the input takes keys at once).
- One accent colour in the interface, the person's, and only where the chrome would be. Emphasis by weight and
  brightness; the ring is the one other coloured thing.
- The ring is real shader output, never a CSS imitation (the static gradient behind it is only the no-WebGL fallback). It is masked to
  its rim, so it never shows through the glass.
- Nothing inside a card paints a band of the plate's colour to fade content (it shows as a stripe on glass): fade with a mask.
