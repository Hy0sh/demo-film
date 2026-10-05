// Lists the accessible names of the visible elements a scenario can aim at,
// in document order, without duplicates: what a failed lookup suggests.
(() => {
  const sel = "a, button, select, textarea, input:not([type=hidden]), label, summary," +
    " [role=button], [role=link], [role=tab], [role=menuitem], [role=option], [role=checkbox], [role=radio]";
  const visible = el => {
    const r = el.getBoundingClientRect();
    return r.width > 0 && r.height > 0 && getComputedStyle(el).visibility !== "hidden";
  };
  const name = el =>
    (el.getAttribute("aria-label") || el.innerText || el.getAttribute("placeholder") || el.value || "")
      .replace(/\s+/g, " ").trim();
  const seen = new Set();
  for (const el of document.querySelectorAll(sel)) {
    if (el.id === "__demo_cursor" || !visible(el)) continue;
    const n = name(el);
    if (n && n.length <= 60) seen.add(n);
  }
  return [...seen];
})()
