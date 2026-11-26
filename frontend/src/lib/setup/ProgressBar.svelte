<script lang="ts">
  interface Props {
    /** 0 to 100; unknown shows a sliver, as a download does before it knows its size. */
    percent: number | null;
    label: string;
  }
  let { percent, label }: Props = $props();
</script>

<div class="bar" role="progressbar" aria-valuemin="0" aria-valuemax="100" aria-valuenow={percent ?? undefined} aria-label={label}>
  <div class="fill" style:transform="scaleX({(percent ?? 4) / 100})"></div>
</div>

<style>
  .bar {
    height: 6px;
    border-radius: 3px;
    background: var(--lift);
    overflow: hidden;
  }
  .fill {
    height: 100%;
    border-radius: 3px;
    background: var(--chrome);
    transform-origin: left;
    transition: transform 0.4s var(--ease);
  }
</style>
