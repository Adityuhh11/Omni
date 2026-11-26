<script lang="ts">
  import { Events } from "@wailsio/runtime";
  import { fly } from "svelte/transition";
  import Icon from "../Icon.svelte";
  import { keycaps } from "../keycaps";
  import { combo } from "../platform";
  import { Service as Settings } from "../../../bindings/omni/internal/settingsui";
  import type { View } from "../../../bindings/omni/internal/settingsui";
  import HotkeyField from "../settings/HotkeyField.svelte";

  let { view }: { view: View } = $props();

  // svelte-ignore state_referenced_locally
  let hotkey = $state(view.Hotkey);
  // svelte-ignore state_referenced_locally
  let startup = $state(view.Autostart);
  let error = $state("");
  // Whether the bar has opened since the hotkey last changed: the launcher says so when it does.
  let tried = $state(false);

  $effect(() => Events.On("bar:shown", () => (tried = true)));

  // The field shows its own refusal ("that hotkey is taken"); this only saves what the person picked.
  async function saveHotkey(accelerator: string) {
    const refused = await Settings.SaveGeneral(accelerator, view.Browser, view.SearchURL);
    if (refused === "") {
      hotkey = accelerator;
      tried = false;
    }
    return refused;
  }

  async function saveStartup() {
    error = await Settings.SetAutostart(startup);
    if (error !== "") startup = !startup;
  }

  const TIPS: [string, string][] = [
    [combo("Tab"), "switches between everything, apps, files and the web"],
    [combo("Ctrl+K"), "shows what can be done with the selected row"],
    [combo("Ctrl+,"), "opens Settings"],
  ];
</script>

<h2>Open it from anywhere</h2>
<p class="lede">One key, in any program. Omni opens over what you are doing and gives the keyboard back when it closes.</p>

<div class="stage" class:tried>
  <div class="caps" aria-label={`The hotkey: ${keycaps(hotkey).join(" ")}`}>
    {#each keycaps(hotkey) as cap (cap)}<kbd class="cap">{cap}</kbd>{/each}
  </div>
  {#key tried}
    <p class="say" aria-live="polite" in:fly={{ y: 6, duration: 260 }}>
      {#if tried}
        <span class="tick"><Icon name="check" size={18} /></span>
        That is it. It opens over whatever you were doing.
      {:else}
        Try it now: press them. <kbd>{combo("Esc")}</kbd> closes the bar again.
      {/if}
    </p>
  {/key}
</div>

<div class="field">
  <span class="label" id="setup-hotkey-label">Prefer another?</span>
  <HotkeyField labelId="setup-hotkey-label" value={view.Hotkey} defaultValue={view.DefaultHotkey} onsave={saveHotkey} />
  <p class="help">Click it, then press the keys you want. Omni says if another program already uses them.</p>
</div>

<div class="field">
  <label class="check"><input type="checkbox" bind:checked={startup} onchange={saveStartup} /> Start when I sign in</label>
  {#if error}<p class="error">{error}</p>{/if}
</div>

<div class="field">
  <span class="label">Three keys worth knowing, once the bar is open</span>
  <ul class="tips">
    {#each TIPS as [key, does] (key)}
      <li><kbd>{key}</kbd><span>{does}</span></li>
    {/each}
  </ul>
</div>

<style>
  .stage {
    position: relative;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 16px;
    margin: 0 0 26px;
    padding: 30px 20px 24px;
    border-radius: 22px;
    background: var(--tile);
    box-shadow: inset 0 0 0 1px var(--line);
    transition: background 400ms var(--ease);
    overflow: hidden;
  }
  /* A glow of the accent that calls for the press, and settles once the bar has opened. */
  .stage::before {
    content: "";
    position: absolute;
    inset: 0;
    background: radial-gradient(ellipse at 50% 38%, color-mix(in srgb, var(--chrome) 24%, transparent), transparent 62%);
    opacity: 0.9;
    pointer-events: none;
    will-change: opacity;
    animation: call 2.4s ease-in-out infinite;
  }
  .stage.tried::before {
    animation: none;
    opacity: 0.4;
  }
  @keyframes call {
    50% {
      opacity: 0.25;
    }
  }
  .caps {
    position: relative;
    display: flex;
    gap: 10px;
  }
  /* Each key is a cap with a lit top edge and a lip beneath, as a keyboard's. */
  .caps .cap {
    display: grid;
    place-items: center;
    min-width: 56px;
    height: 56px;
    padding: 0 18px;
    box-sizing: border-box;
    border-radius: 14px;
    background: var(--lift-keyed);
    font: 600 18px/1 var(--sans);
    box-shadow:
      inset 0 1px 0 color-mix(in srgb, var(--ink) 20%, transparent),
      0 3px 0 color-mix(in srgb, var(--ink) 11%, transparent),
      0 0 0 1px color-mix(in srgb, var(--ink) 9%, transparent);
    transition:
      color 300ms var(--ease),
      box-shadow 300ms var(--ease);
  }
  .tried .caps .cap {
    color: var(--chrome);
    box-shadow:
      inset 0 1px 0 color-mix(in srgb, var(--ink) 20%, transparent),
      0 3px 0 color-mix(in srgb, var(--chrome) 40%, transparent),
      0 0 0 1px color-mix(in srgb, var(--chrome) 55%, transparent);
  }
  .say {
    position: relative;
    display: flex;
    align-items: center;
    gap: 8px;
    min-height: 22px;
    margin: 0;
    color: var(--text-2);
    font-size: 13.5px;
  }
  .tried .say {
    color: var(--text);
  }
  .tick {
    color: var(--chrome);
  }
  .tips {
    display: flex;
    flex-direction: column;
    gap: 9px;
    margin: 2px 0 0;
    padding: 0;
    list-style: none;
  }
  .tips li {
    display: flex;
    align-items: baseline;
    gap: 12px;
    color: var(--text-2);
    font-size: 13px;
  }
  .tips kbd {
    flex: none;
    min-width: 54px;
  }
  @media (prefers-reduced-motion: reduce) {
    .stage::before {
      animation: none;
    }
  }
</style>
