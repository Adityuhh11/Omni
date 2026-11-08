import { Events } from "@wailsio/runtime";
import { Service as Glass, type Glass as Look } from "../../bindings/omni/internal/glass";

// The frosted glass behind a card, while the Glass theme is chosen. A page cannot blur what is behind its own window, so it tells the window where its card is
// (internal/glass) and the system blurs the desktop there; the page paints the tint (--glass) on top. The window says what the
// glass is: while Windows' energy saver stops the blur, the wallpaper under the window, blurred by the window, is laid in the
// card under the tint (html[data-glass="wallpaper"]), or without one the plate is nearly opaque (html[data-glass="tint"]); someone
// who asked the system for less transparency, or a system that cannot do it, gets the plate opaque (html[data-glass="off"]).

const reduced = matchMedia("(prefers-reduced-transparency: reduce)");

function off() {
  document.documentElement.dataset.glass = "off";
}

/**
 * Lays the blurred wallpaper of the window's screen in the card, under a layer of the tint, lined up with where the screen is;
 * or takes it away. The card's rounded corners cut it.
 */
function paint(node: HTMLElement, look: Look | null, box: DOMRect) {
  const style = node.style;
  if (look?.Look !== "wallpaper") {
    style.backgroundImage = style.backgroundSize = style.backgroundPosition = style.backgroundRepeat = style.backgroundOrigin = "";
    return;
  }
  const image = `linear-gradient(var(--glass), var(--glass)), url("${look.Image}")`;
  if (style.backgroundImage !== image) style.backgroundImage = image; // the same picture is not taken in again on every move
  style.backgroundSize = `auto, ${look.Width}px ${look.Height}px`;
  style.backgroundPosition = `0 0, ${look.Left - box.left}px ${look.Top - box.top}px`;
  style.backgroundRepeat = "no-repeat";
  style.backgroundOrigin = "border-box";
}

/** A Svelte action: the glass follows the element's box, rounded by radius CSS pixels. */
export function glass(node: HTMLElement, radius: number) {
  let last = ""; // what the window was last told; "none" for no glass
  let frame = 0;
  const send = () => {
    frame = 0;
    if (document.documentElement.dataset.theme !== "glass" || reduced.matches) {
      if (reduced.matches) off();
      paint(node, null, node.getBoundingClientRect());
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
        if (look.Look === "blur") delete document.documentElement.dataset.glass;
        else document.documentElement.dataset.glass = look.Look;
        paint(node, look, box);
      },
      (err) => {
        console.warn("glass:", err);
        off();
        paint(node, null, box);
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
  // The wallpaper is lined up with the screen, so it follows the window. ponytail: Windows sends the move once the window stops
  // (debounced 50 ms), so during a drag the picture travels with the card and settles after; a frame loop would cost the power
  // energy saver is saving.
  const stopMoves = Events.On(Events.Types.Windows.WindowDidMove, () => {
    if (document.documentElement.dataset.glass === "wallpaper") shown();
  });
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
      stopMoves();
      reduced.removeEventListener("change", changed);
      cancelAnimationFrame(frame);
      // The card is gone: no glass left behind on an empty window.
      if (last !== "none" && last !== "") Glass.Shape(0, 0, 0, 0, 0, true).catch((err) => console.warn("glass:", err));
    },
  };
}
