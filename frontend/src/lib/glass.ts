import { isMac } from "./platform";
import { Service as Glass } from "../../bindings/omni/internal/glass";

// The frosted glass behind a card, while the Glass theme is chosen on Windows. A page cannot blur what is behind its own window, so it tells the window where its card is
// (internal/glass) and the system blurs the desktop there; the page paints the tint (--glass) on top. The window says what the
// glass is: while Windows' energy saver stops the blur the plate is nearly opaque (html[data-glass="tint"]); someone who asked the
// system for less transparency, or a system that cannot do it, gets the plate opaque (html[data-glass="off"]).

const reduced = matchMedia("(prefers-reduced-transparency: reduce)");

// A Mac has the launcher's Liquid Glass instead (panel.Glass, and the card windows' own), so this blur is Windows' only.
const wanted = () => !isMac && document.documentElement.dataset.theme === "glass";

// Panes drawn as the same glass beside a card (the bar's quick-open tiles), each round: the card's glass is asked for with them.
const panes = new Set<HTMLElement>();
const resend = new Set<() => void>();
const sendAll = () => resend.forEach((soon) => soon());

/** A Svelte action: the element is a round pane of the same glass as the card on its page. */
export function glassPane(node: HTMLElement) {
  panes.add(node);
  const observer = new ResizeObserver(sendAll);
  observer.observe(node);
  sendAll();
  return {
    destroy() {
      observer.disconnect();
      panes.delete(node);
      sendAll();
    },
  };
}

function off() {
  document.documentElement.dataset.glass = "off";
}

/** A Svelte action: the glass follows the element's box, rounded by radius CSS pixels. */
export function glass(node: HTMLElement, radius: number) {
  let last = ""; // what the window was last told; "none" for no glass
  let frame = 0;
  const send = () => {
    frame = 0;
    if (!wanted() || reduced.matches) {
      if (reduced.matches) off();
      // Another theme, or no transparency wanted: take away any blur this card had.
      if (last !== "none" && last !== "") Glass.Shape([], true).catch((err) => console.warn("glass:", err));
      last = "none"; // a window that never had glass is not asked to remove it
      return;
    }
    const box = node.getBoundingClientRect();
    const beside = [...panes].map((pane) => ({ pane, box: pane.getBoundingClientRect() }));
    const shapes = [
      { X: box.left, Y: box.top, W: box.width, H: box.height, R: radius },
      ...beside.map(({ box }) => ({ X: box.left, Y: box.top, W: box.width, H: box.height, R: Math.min(box.width, box.height) / 2 })),
    ];
    const dark = getComputedStyle(document.documentElement).colorScheme !== "light";
    const key = `${shapes.map((r) => `${r.X} ${r.Y} ${r.W} ${r.H} ${r.R}`).join(" ")} ${dark}`;
    if (key === last) return;
    last = key;
    Glass.Shape(shapes, dark).then(
      // A call that failed earlier (the window had no handle yet) does not keep the card opaque once one works.
      (look) => {
        if (look.Look === "blur") delete document.documentElement.dataset.glass;
        else document.documentElement.dataset.glass = look.Look;
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
  resend.add(soon);
  const observer = new ResizeObserver(soon);
  observer.observe(node);
  // The glass comes and goes with the Glass theme.
  const themeChanges = new MutationObserver(soon);
  themeChanges.observe(document.documentElement, { attributes: true, attributeFilter: ["data-theme"] });
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
      resend.delete(soon);
      observer.disconnect();
      themeChanges.disconnect();
      removeEventListener("resize", soon);
      removeEventListener("focus", shown);
      reduced.removeEventListener("change", changed);
      cancelAnimationFrame(frame);
      // The card is gone: no glass left behind on an empty window.
      if (last !== "none" && last !== "") Glass.Shape([], true).catch((err) => console.warn("glass:", err));
    },
  };
}
