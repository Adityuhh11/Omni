<script lang="ts">
  import DemoBar from "./DemoBar.svelte";
</script>

<div class="hero">
  <h1>Meet Omni</h1>
  <p>One key opens anything. Here is what you can type.</p>
</div>

<DemoBar />

<p class="help privacy">Everything stays on this computer: no account, no telemetry. The network is used only for what you start or switch on.</p>

<style>
  .hero {
    margin: 0 0 20px;
    text-align: center;
  }
  h1 {
    margin: 0 0 4px;
    font: 640 28px/1.1 var(--display);
    letter-spacing: -0.025em;
  }
  .hero p {
    margin: 0;
    color: var(--text-2);
    font-size: 14px;
  }
  .privacy {
    margin-top: 18px;
    text-align: center;
    text-wrap: balance;
  }
</style>
