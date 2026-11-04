# Design

The visual system of the launcher panel, recorded from the built page (`frontend/src`) on 2026-10-15. Tokens live in
`frontend/src/theme.css`.

## World

Black glass and chrome. Every card (the bar, the timer, the settings, notes, chat and calendar) is frosted glass, chosen by the
owner on 2026-11-04 from five studies (black plate, frosted, clear liquid glass, smoked, frosted with flowing liquid): the
system blurs what is behind the card and the theme's plate, 55% opaque (`--glass`; 68% on the light themes), tints it dark
enough to read over a white page. A web page cannot blur what is behind its own window, so the blur is the system's, cut to
the card's exact rounded shape (`internal/glass`). Frosted glass was tried once before, on 2026-10-16, as a flat grey plate, and
dropped; this one keeps the plate's colour and its lit rim. No grain, no gradient. The premium comes from a lit top rim and a
hairline edge, and from the ring. Where the system cannot blur, or the person asked it for less transparency, the plate is
opaque again (`html[data-glass="off"]`, `prefers-reduced-transparency`). The interface itself has no accent hue (emphasis is
weight and brightness); the only colour is the ring, a metal that changes every time the panel closes.

## Shape and motion

- **Capsule to card.** At rest the bar is a 680 x 65 capsule (radius 32.5 px, a perfect semicircle end). Results grow it
  into a card of the same width and the same corner radius; only the height animates (240 ms, exponential ease-out).
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
  flight; the window grows at once and shrinks 280 ms after the card.

## Color

| Token | Value | Use |
|---|---|---|
| `--plate` | `#000` | the theme's colour; opaque only where there is no glass |
| `--glass` | plate at 55% | the cards: tint over the system's blur |
| `--glass-raised` + `--glass-blur` | plate at 84%, the page's own blur | what floats inside a card: the action panel, an event's details, the chat's composer and history |
| `--lift` | `rgba(255,255,255,.075)` | selected row |
| `--tile` | `rgba(255,255,255,.055)` | icon tiles, count and key chips |
| `--line` | `rgba(255,255,255,.07)` | hairlines between bar, list and hints |
| `--text` | `#eceef2` | names, input |
| `--text-2` | `#aaacb2` | snippets, notes |
| `--text-3` | `#8a8c93` | placeholder, paths, locations (5:1 on the plate) |
| `--chrome` | `#e6e8ec` | caret, focused glyph, selected tile |
| `--danger` | `#e0877f` | problems only |

## Type

Geist Variable for text, Geist Mono Variable for paths, locations, counts and key caps (measurement and data only), and
Sora Variable (19 px / 400, tracking -0.01em) for the one thing you type and its placeholder, chosen by the owner from six
candidates on 2026-10-16 (Bricolage Grotesque was tried first).
All three are bundled from `@fontsource-variable` (OFL), never loaded from a network. Scale: names 14.5 / 520,
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
- **Browser surfaces:** themed selection, caret and scrollbar; no focus ring on rows (selection is the focus).

## Rules

- Nothing animated may run while the panel is closed; nothing visual may be added to the hotkey path.
- No accent color in the interface. Emphasis by weight and brightness; the ring is the one coloured thing.
- The ring is real shader output, never a CSS imitation (the static gradient behind it is only the no-WebGL fallback). It is masked to
  its rim, so it never shows through the glass.
- Nothing inside a card paints a band of the plate's colour to fade content (it shows as a stripe on glass): fade with a mask.
