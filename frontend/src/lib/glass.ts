import { Service as Glass } from "../../bindings/omni/internal/glass";

// The frosted glass behind a card. A page cannot blur what is behind its own window, so it tells the window where its card is
// (internal/glass) and the system blurs the desktop there; the page paints the tint (--glass) on top. Someone who asked the
// system for less transparency, or a system that cannot do it, gets the plate opaque (html[data-glass="off"] in theme.css).

const reduced = matchMedia("(prefers-reduced-transparency: reduce)");

function off() {
  document.documentElement.dataset.glass = "off";
}

/** A Svelte action: the glass follows the element's box, rounded by radius CSS pixels. */
export function glass(node: HTMLElement, radius: number) {
  let last = "";
  let frame = 0;
  const send = () => {
    frame = 0;
    if (reduced.matches) {
      off();
      return;
    }
    const box = node.getBoundingClientRect();
    const dark = getComputedStyle(document.documentElement).colorScheme !== "light";
    const key = `${box.left} ${box.top} ${box.width} ${box.height} ${radius} ${dark}`;
    if (key === last) return;
    last = key;
    Glass.Shape(box.left, box.top, box.width, box.height, radius, dark).catch((err) => {
      console.warn("glass:", err);
      off();
    });
  };
  // Once a frame at most: the bar's card changes height every frame while it grows.
  const soon = () => (frame ||= requestAnimationFrame(send));
  const observer = new ResizeObserver(soon);
  observer.observe(node);
  // The theme switches between dark and light glass.
  const themes = new MutationObserver(soon);
  themes.observe(document.documentElement, { attributes: true, attributeFilter: ["data-theme"] });
  addEventListener("resize", soon);
  const changed = () => {
    if (!reduced.matches) delete document.documentElement.dataset.glass;
    last = "";
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
      Glass.Shape(0, 0, 0, 0, 0, true).catch((err) => console.warn("glass:", err));
    },
  };
}
