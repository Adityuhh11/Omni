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
</script>

<h2>Open it from anywhere</h2>
<p class="lede">Press the hotkey in any program, then type.</p>

<div class="stage" class:tried>
  <div class="caps" aria-label={`The hotkey: ${keycaps(hotkey).join(" ")}`}>
    {#each keycaps(hotkey) as cap (cap)}<kbd class="cap">{cap}</kbd>{/each}
  </div>
  {#key tried}
    <p class="say" aria-live="polite" in:fly={{ y: 6, duration: 260 }}>
      {#if tried}
        <span class="tick"><Icon name="check" size={16} /></span>It opens over whatever you were doing.
      {:else}
        Try it now. <kbd>{combo("Esc")}</kbd> closes the bar again.
      {/if}
    </p>
  {/key}
</div>

<div class="group">
  <div class="item">
    <span class="name" id="setup-hotkey-label">Hotkey</span>
    <HotkeyField labelId="setup-hotkey-label" value={view.Hotkey} defaultValue={view.DefaultHotkey} onsave={saveHotkey} />
  </div>
  <label class="item">
    <span class="name">Start when I sign in</span>
    <input type="checkbox" class="switch" bind:checked={startup} onchange={saveStartup} />
  </label>
</div>
{#if error}<p class="error">{error}</p>{/if}

<p class="help tips">
  Once it is open: <kbd>{combo("Tab")}</kbd> changes what is searched, <kbd>{combo("Ctrl+K")}</kbd> shows actions.
</p>

<style>
  .stage {
    position: relative;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 14px;
    margin: 0 0 22px;
    padding: 18px 0 8px;
  }
  /* A glow of the accent that calls for the press, and settles once the bar has opened. */
  .stage::before {
    content: "";
    position: absolute;
    inset: -10px -40px 0;
    background: radial-gradient(ellipse at 50% 40%, color-mix(in srgb, var(--chrome) 20%, transparent), transparent 62%);
    opacity: 0.9;
    pointer-events: none;
    will-change: opacity;
    animation: call 2.4s ease-in-out infinite;
  }
  .stage.tried::before {
    animation: none;
    opacity: 0.35;
  }
  @keyframes call {
    50% {
      opacity: 0.25;
    }
  }
  .caps {
    position: relative;
    display: flex;
    gap: 8px;
  }
  /* Each key is a cap with a lit top edge and a lip beneath, as a keyboard's. */
  .caps .cap {
    display: grid;
    place-items: center;
    min-width: 48px;
    height: 48px;
    padding: 0 16px;
    box-sizing: border-box;
    border-radius: 12px;
    background: var(--lift-keyed);
    font: 600 16px/1 var(--sans);
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
    gap: 7px;
    min-height: 20px;
    margin: 0;
    color: var(--text-2);
    font-size: 13px;
  }
  .tried .say {
    color: var(--text);
  }
  .tick {
    display: grid;
    color: var(--chrome);
  }
  .tips {
    margin-top: 16px;
    line-height: 1.9;
  }
  @media (prefers-reduced-motion: reduce) {
    .stage::before {
      animation: none;
    }
  }
</style>
