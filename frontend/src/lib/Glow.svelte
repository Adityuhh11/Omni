<!--
  Two soft lights drifting inside a glass card, in the accent's colour: they cross, run together into one brighter light and part
  again. Only in the Glass theme, while the system blurs (data-glass unset); it stops while the bar is closed and holds still for
  someone who asked for less motion. Two durations that never line up keep the paths from looking like a loop.
  The parent isolates its stacking, so the lights lie over its plate and under its content.
-->
<div class="glow" aria-hidden="true"><span class="orb a"></span><span class="orb b"></span></div>

<style>
  .glow {
    position: absolute;
    inset: 0;
    z-index: -1;
    border-radius: inherit;
    overflow: hidden;
    pointer-events: none;
    container-type: size;
  }
  :global(html:not([data-theme="glass"])) .glow,
  :global(html[data-glass]) .glow {
    display: none;
  }
  .orb {
    --size: max(200px, 40cqw);
    position: absolute;
    left: calc(50% - var(--size) / 2);
    top: calc(50% - var(--size) / 2);
    width: var(--size);
    height: var(--size);
    border-radius: 50%;
    background: radial-gradient(circle, color-mix(in srgb, var(--chrome) 30%, transparent), transparent 68%);
    mix-blend-mode: screen; /* where they meet they add up: one bigger, brighter light */
    will-change: transform;
    animation: a 27s ease-in-out infinite alternate;
  }
  .b {
    animation-name: b;
    animation-duration: 37s;
  }
  :global(html[data-active="false"]) .orb {
    animation-play-state: paused;
  }
  @media (prefers-reduced-motion: reduce) {
    .orb {
      animation: none;
    }
    .a {
      transform: translate(-22cqw, -10cqh);
    }
    .b {
      transform: translate(24cqw, 12cqh);
    }
  }
  @keyframes a {
    0% { transform: translate(-32cqw, -22cqh) scale(1); }
    22% { transform: translate(6cqw, 24cqh) scale(1.2); }
    45% { transform: translate(30cqw, -12cqh) scale(0.9); }
    70% { transform: translate(-4cqw, 6cqh) scale(1.3); }
    100% { transform: translate(-28cqw, 26cqh) scale(1); }
  }
  @keyframes b {
    0% { transform: translate(30cqw, 20cqh) scale(1.1); }
    30% { transform: translate(-10cqw, -24cqh) scale(0.9); }
    55% { transform: translate(2cqw, 8cqh) scale(1.3); }
    80% { transform: translate(-30cqw, 18cqh) scale(1); }
    100% { transform: translate(26cqw, -20cqh) scale(1.15); }
  }
</style>
