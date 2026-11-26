<script lang="ts">
  import { Service as Settings } from "../../../bindings/omni/internal/settingsui";
  import { ACCENTS, applyLook, THEMES } from "../theme";
  import ColorPicker from "./ColorPicker.svelte";

  interface Props {
    /** The theme and accent in use; a change is applied to every window, saved, and reported here. */
    theme: string;
    accent: string;
    /** Called after a change was saved. */
    onsaved?: () => void;
  }
  let { theme = $bindable(), accent = $bindable(), onsaved = () => {} }: Props = $props();

  let error = $state("");

  async function saveTheme(id: string) {
    const before = theme;
    theme = id;
    applyLook({ Theme: id });
    error = await Settings.SetTheme(id);
    if (error !== "") {
      theme = before;
      applyLook({ Theme: before });
    } else onsaved();
  }

  async function saveAccent(id: string) {
    const before = accent;
    accent = id;
    applyLook({ Accent: id });
    error = await Settings.SetAccent(id);
    if (error !== "") {
      accent = before;
      applyLook({ Accent: before });
    } else onsaved();
  }

  // A colour of their own is shown on the page while the picker is still open, and saved when it lets go.
  const previewAccent = (hex: string) => applyLook({ Accent: hex });
  const isCustom = $derived(!ACCENTS.some((a) => a.id === accent));
  let picking = $state(false);
  const accentName = $derived(ACCENTS.find((a) => a.id === accent)?.label ?? accent);
  // The picker opens on the colour in use: a custom one, or the preset's own.
  const pickerStart = $derived(isCustom ? accent : (ACCENTS.find((a) => a.id === accent)?.swatch ?? "#f2601d"));
</script>

<div class="item stack">
  <span class="name" id="theme-label">Theme</span>
  <div class="tiles" role="radiogroup" aria-labelledby="theme-label">
    {#each THEMES as t (t.id)}
      <button type="button" role="radio" aria-checked={theme === t.id} class:on={theme === t.id} onclick={() => saveTheme(t.id)}>
        <span class="preview" style:background={t.plate} style:color={t.text}>
          Aa
          <i></i>
        </span>
        <span class="label">{t.label}</span>
      </button>
    {/each}
  </div>
</div>
<div class="item stack">
  <div class="head">
    <span class="name" id="accent-label">Accent</span>
    <span class="sub" aria-live="polite">{accentName}</span>
  </div>
  <div class="swatches" role="radiogroup" aria-labelledby="accent-label">
    {#each ACCENTS as a (a.id)}
      <button type="button" role="radio" aria-checked={accent === a.id} aria-label={a.label} title={a.label} class:on={accent === a.id} style:--dot={a.swatch} onclick={() => saveAccent(a.id)}></button>
    {/each}
    <span class="anchor">
      <!-- the picker closes on a press anywhere outside it; this button's own press is not one, or its click would only reopen it -->
      <button
        type="button"
        class="custom"
        class:on={isCustom}
        aria-label="Choose any colour"
        title={picking ? undefined : "Any colour"}
        aria-haspopup="dialog"
        aria-expanded={picking}
        style:--dot={isCustom ? accent : null}
        onpointerdown={(event) => event.stopPropagation()}
        onclick={() => (picking = !picking)}
      ></button>
      {#if picking}
        <ColorPicker value={pickerStart} onpreview={previewAccent} oncommit={saveAccent} onclose={() => (picking = false)} />
      {/if}
    </span>
  </div>
</div>
{#if error}<p class="error">{error}</p>{/if}

<style>
  /* The themes as small previews of their plate and type: what a theme looks like is the whole choice. */
  .tiles {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(104px, 1fr));
    gap: 16px 10px;
  }
  .tiles button {
    display: flex;
    flex-direction: column;
    gap: 7px;
    padding: 0;
    border: 0;
    background: transparent;
    color: var(--text-2);
    font: 500 12.5px/1 var(--sans);
    text-align: left;
    cursor: pointer;
  }
  .preview {
    position: relative;
    display: flex;
    align-items: flex-end;
    height: 52px;
    padding: 0 0 8px 11px;
    border-radius: 11px;
    box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--ink) 16%, transparent);
    font: 620 15px/1 var(--sans);
    transition: transform 160ms var(--ease), box-shadow 160ms var(--ease);
  }
  /* a stroke of the accent on each plate, as the bar's caret and edge wear it */
  .preview i {
    position: absolute;
    top: 10px;
    right: 11px;
    width: 18px;
    height: 4px;
    border-radius: 2px;
    background: var(--chrome);
  }
  .tiles button:hover .preview {
    transform: translateY(-1px);
  }
  .tiles button.on {
    color: var(--text);
    font-weight: 600;
  }
  .tiles button.on .preview {
    box-shadow:
      inset 0 0 0 1px color-mix(in srgb, var(--ink) 16%, transparent),
      0 0 0 2px var(--chrome);
  }
  .tiles button:focus-visible {
    outline: none;
  }
  .tiles button:focus-visible .preview {
    box-shadow:
      inset 0 0 0 1px color-mix(in srgb, var(--ink) 16%, transparent),
      0 0 0 2px var(--text-2);
  }
  .head {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    gap: 12px;
  }
  /* The accents as round swatches, the chosen one ringed in its own colour, and any colour at the end. */
  .swatches {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 12px;
  }
  .swatches button {
    flex: none;
    width: 26px;
    height: 26px;
    padding: 0;
    border: 0;
    border-radius: 50%;
    background: var(--dot);
    box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--ink) 20%, transparent);
    cursor: pointer;
    transition: transform 160ms var(--ease), box-shadow 160ms var(--ease);
  }
  .swatches button:hover {
    transform: scale(1.1);
  }
  .swatches button.on {
    box-shadow:
      0 0 0 2px var(--plate),
      0 0 0 4px var(--dot);
  }
  .swatches button:focus-visible {
    outline: 2px solid var(--text-2);
    outline-offset: 3px;
  }
  /* Any colour: a ring of hues, filled with the colour once one is chosen. */
  .swatches .custom {
    background:
      radial-gradient(circle, var(--dot, transparent) 0 46%, transparent 48%),
      conic-gradient(#e5484d, #f5d90a, #46a758, #5bb8d0, #3e63dd, #8e4ec6, #d6409f, #e5484d);
  }
  .swatches .custom.on {
    box-shadow:
      0 0 0 2px var(--plate),
      0 0 0 4px color-mix(in srgb, var(--dot) 80%, var(--text));
  }
  .anchor {
    position: relative;
  }
  @media (prefers-reduced-motion: reduce) {
    .preview,
    .swatches button {
      transition: none;
    }
    .tiles button:hover .preview,
    .swatches button:hover {
      transform: none;
    }
  }
</style>
