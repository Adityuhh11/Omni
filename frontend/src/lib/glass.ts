import { Service as Glass } from "../../bindings/omni/internal/glass";

// The frosted glass behind a card, while the Glass theme is chosen. A page cannot blur what is behind its own window, so it tells the window where its card is
// (internal/glass) and the system blurs the desktop there; the page paints the tint (--glass) on top. The window says what the
// glass is: while Windows' energy saver stops the blur the plate is nearly opaque, without one (html[data-glass="tint"]); someone
// who asked the system for less transparency, or a system that cannot do it, gets the plate opaque (html[data-glass="off"]).

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
      (look) => {
        if (look === "blur") delete document.documentElement.dataset.glass;
        else document.documentElement.dataset.glass = look;
      },
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
  // Energy saver and the transparency setting change while the window is hidden: ask again whenever it is shown.
  const shown = () => {
    if (last !== "none") last = "";
    soon();
  };
  addEventListener("focus", shown);
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
      removeEventListener("focus", shown);
      reduced.removeEventListener("change", changed);
      cancelAnimationFrame(frame);
      // The card is gone: no glass left behind on an empty window.
      if (last !== "none" && last !== "") Glass.Shape(0, 0, 0, 0, 0, true).catch((err) => console.warn("glass:", err));
    },
  };
}
