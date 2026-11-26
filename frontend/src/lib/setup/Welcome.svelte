<script lang="ts">
  import DemoBar from "./DemoBar.svelte";
</script>

<div class="hero">
  <h1>Meet Omni</h1>
  <p>One key opens anything on this computer. Here is what you can type.</p>
</div>

<DemoBar />

<p class="help privacy">
  Everything runs on this computer: no account, no telemetry. Omni uses the network only when you ask it to: the optional downloads (file search and
  Omni AI, about 1 GB each, once), a web search in your browser, and an update check if you switch it on.
</p>

<style>
  .hero {
    margin: 4px 0 26px;
    text-align: center;
  }
  h1 {
    margin: 0 0 6px;
    font: 640 32px/1.1 var(--display);
    letter-spacing: -0.025em;
  }
  .hero p {
    margin: 0;
    color: var(--text-2);
    font-size: 14.5px;
  }
  .privacy {
    margin-top: 22px;
    text-align: center;
    text-wrap: balance;
  }
</style>
