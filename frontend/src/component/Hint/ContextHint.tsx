// One shared right-click hint for a whole subtree. Right-clicking any
// descendant with a data-hint attribute shows that text next to it
// straight away; it hides when the pointer leaves the element, on any
// click, scroll or Escape. One div serves the whole list, and showing it
// never re-renders React (the hint is updated through a ref).

import { useEffect, useRef } from "react";
import type { MouseEvent, ReactNode } from "react";

const GAP = 6;

export const ContextHint = ({ children }: { children: ReactNode }) => {
  const tip = useRef<HTMLDivElement>(null);
  const current = useRef<HTMLElement | null>(null);

  const hide = () => {
    current.current = null;
    if (tip.current) tip.current.style.visibility = "hidden";
  };

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && hide();
    // Capture so scrolls inside the list's own scroll container count.
    window.addEventListener("scroll", hide, true);
    window.addEventListener("blur", hide);
    window.addEventListener("keydown", onKey);
    return () => {
      window.removeEventListener("scroll", hide, true);
      window.removeEventListener("blur", hide);
      window.removeEventListener("keydown", onKey);
    };
  }, []);

  const show = (el: HTMLElement) => {
    const t = tip.current;
    if (!t) return;
    current.current = el;
    t.textContent = el.dataset.hint ?? "";
    t.style.visibility = "hidden";
    t.style.left = "0px";
    t.style.top = "0px";
    const r = el.getBoundingClientRect();
    const { width, height } = t.getBoundingClientRect();
    const left = Math.max(
      GAP,
      Math.min(r.left + r.width / 2 - width / 2, window.innerWidth - width - GAP)
    );
    // Below the element so the row's name stays visible; above it when
    // there's no room.
    const top =
      r.bottom + GAP + height <= window.innerHeight
        ? r.bottom + GAP
        : r.top - height - GAP;
    t.style.left = `${left}px`;
    t.style.top = `${top}px`;
    t.style.visibility = "visible";
  };

  const onContextMenu = (e: MouseEvent) => {
    const el = (e.target as Element).closest<HTMLElement>("[data-hint]");
    if (!el) return;
    e.preventDefault();
    show(el);
  };

  const onMouseOut = (e: MouseEvent) => {
    const el = current.current;
    const to = e.relatedTarget as Node | null;
    if (el && !(to && el.contains(to))) hide();
  };

  return (
    <div onContextMenu={onContextMenu} onMouseOut={onMouseOut} onMouseDown={hide}>
      {children}
      <div
        ref={tip}
        role="tooltip"
        className="fixed z-[1100] pointer-events-none whitespace-nowrap rounded-md bg-zinc-800 px-2 py-1 text-xs text-zinc-100 shadow-lg"
        style={{ visibility: "hidden" }}
      />
    </div>
  );
};

export default ContextHint;
