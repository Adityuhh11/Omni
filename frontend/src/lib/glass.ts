import { Service as Glass } from "../../bindings/omni/internal/glass";

// The frosted glass behind a card, while the Glass theme is chosen. A page cannot blur what is behind its own window, so it tells the window where its card is
// (internal/glass) and the system blurs the desktop there; the page paints the tint (--glass) on top. Someone who asked the
// system for less transparency, or a system that cannot do it, gets the plate opaque (html[data-glass="off"] in theme.css).

const reduced = matchMedia("(prefers-reduced-transparency: reduce)");

function off() {
  document.documentElement.dataset.glass = "off";
}

/** A Svelte action: the glass follows the element's box, rounded by radius CSS pixels. */
export function glass(node: HTMLElement, radius: number) {
  let last = ""; // what the window was last told; "none" for no glass
  let frame = 0;
  const send = () => {
    frame = 0;
    if (document.documentElement.dataset.theme !== "glass" || reduced.matches) {
      if (reduced.matches) off();
      // Another theme, or no transparency wanted: take away any blur this card had.
      if (last !== "none" && last !== "") Glass.Shape(0, 0, 0, 0, 0, true).catch((err) => console.warn("glass:", err));
      last = "none"; // a window that never had glass is not asked to remove it
      return;
    }
    const box = node.getBoundingClientRect();
    const dark = getComputedStyle(document.documentElement).colorScheme !== "light";
    const key = `${box.left} ${box.top} ${box.width} ${box.height} ${radius} ${dark}`;
    if (key === last) return;
    last = key;
    Glass.Shape(box.left, box.top, box.width, box.height, radius, dark).then(
      // A call that failed earlier (the window had no handle yet) does not keep the card opaque once one works.
      () => delete document.documentElement.dataset.glass,
      (err) => {
        console.warn("glass:", err);
        off();
        last = ""; // tried again on the next change
      },
    );
  };
  // Once a frame at most: the bar's card changes height every frame while it grows.
  const soon = () => (frame ||= requestAnimationFrame(send));
  const observer = new ResizeObserver(soon);
  observer.observe(node);
  // The glass comes and goes with the Glass theme.
  const themes = new MutationObserver(soon);
  themes.observe(document.documentElement, { attributes: true, attributeFilter: ["data-theme"] });
  addEventListener("resize", soon);
  const changed = () => {
    // Turning it on takes the blur away (send sees what was shown); turning it off shows it again (the key no longer matches).
    if (!reduced.matches) delete document.documentElement.dataset.glass;
    soon();
  };
  reduced.addEventListener("change", changed);
  soon();
  return {
    update(next: number) {
      radius = next;
      soon();
    },
    destroy() {
      observer.disconnect();
      themes.disconnect();
      removeEventListener("resize", soon);
      reduced.removeEventListener("change", changed);
      cancelAnimationFrame(frame);
      // The card is gone: no glass left behind on an empty window.
      if (last !== "none" && last !== "") Glass.Shape(0, 0, 0, 0, 0, true).catch((err) => console.warn("glass:", err));
    },
  };
}
