<script lang="ts">
  import type { View } from "../../../bindings/omni/internal/settingsui";
  import LookPicker from "../settings/LookPicker.svelte";
  import DemoBar from "./DemoBar.svelte";
  import { SCENES, type Scene } from "./demo";

  let { view }: { view: View } = $props();

  // svelte-ignore state_referenced_locally
  let theme = $state(view.Theme);
  // svelte-ignore state_referenced_locally
  let accent = $state(view.Accent);

  // The bar the choice is for, one row tall, typing a word and left as it is.
  const first = SCENES[0];
  const scene: Scene = { ...first, items: first.items?.slice(0, 1) };
</script>

<h2>Pick your look</h2>
<p class="lede">The bar and this window follow your choice. Settings, General changes it later.</p>

<div class="preview">
  <DemoBar scenes={[scene]} autoplay={false} chips={false} height={130} />
</div>

<div class="group">
  <LookPicker bind:theme bind:accent />
</div>

<style>
  .preview {
    margin: 0 0 18px;
  }
</style>
