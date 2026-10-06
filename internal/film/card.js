// The transition card that covers a cut. Called with a text, it draws the
// card (a spinner over the whole page) or updates its line; an empty text
// shows the spinner alone. Called with null, it fades the card away.
text => {
  let c = document.getElementById("__demo_card");
  if (text === null || text === undefined) {
    if (c) {
      c.style.opacity = 0;
      setTimeout(() => c.remove(), 400);
    }
    return;
  }
  if (!c) {
    c = document.createElement("div");
    c.id = "__demo_card";
    c.style.cssText = "position:fixed;inset:0;z-index:2147483647;display:flex;flex-direction:column;gap:24px;" +
      "align-items:center;justify-content:center;background:#11161b;color:#fff;" +
      "font:600 34px system-ui,sans-serif;pointer-events:none;transition:opacity .4s";

    const style = document.createElement("style");
    style.textContent = "@keyframes __demo_spin { to { transform: rotate(360deg) } }";
    const spinner = document.createElement("div");
    spinner.style.cssText = "width:56px;height:56px;border:6px solid #2457a6;border-top-color:#7fa8e8;" +
      "border-radius:50%;animation:__demo_spin 1s linear infinite";
    c.append(style, spinner, document.createElement("div"));
    document.documentElement.appendChild(c);
  }
  c.lastChild.textContent = text;
}
