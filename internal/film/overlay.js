// Injected before any page script, with `cfg` = {locale, hide} prepended by
// overlay.go. Sets the locale, hides the listed elements, and draws a visible
// cursor with a halo on every click. The page is otherwise filmed untouched.
(cfg => {
  if (cfg.locale) {
    try { localStorage.setItem("i18nextLng", cfg.locale); } catch (e) {}
  }

  const css = `
    #__demo_cursor {
      position: fixed; z-index: 2147483647; pointer-events: none;
      width: 22px; height: 22px; margin: -11px 0 0 -11px; left: -50px; top: -50px;
      border-radius: 50%; border: 2px solid #2457a6; background: rgba(36, 87, 166, .35);
      transition: left .08s linear, top .08s linear;
    }
    .__demo_halo {
      position: fixed; z-index: 2147483646; pointer-events: none;
      width: 48px; height: 48px; margin: -24px 0 0 -24px;
      border-radius: 50%; border: 3px solid #e8a33a;
      transition: transform .5s, opacity .5s;
    }
    .__demo_halo.__demo_fade { transform: scale(1.8); opacity: 0; }
  ` + cfg.hide.map(sel => `${sel} { display: none !important; }`).join("\n");

  const add = () => {
    if (document.getElementById("__demo_cursor")) return;
    const style = document.createElement("style");
    style.textContent = css;
    document.documentElement.appendChild(style);

    const cursor = document.createElement("div");
    cursor.id = "__demo_cursor";
    document.documentElement.appendChild(cursor);

    const at = (el, e) => { el.style.left = e.clientX + "px"; el.style.top = e.clientY + "px"; };
    addEventListener("mousemove", e => at(cursor, e), true);
    addEventListener("mousedown", e => {
      const halo = document.createElement("div");
      halo.className = "__demo_halo";
      at(halo, e);
      document.documentElement.appendChild(halo);
      requestAnimationFrame(() => halo.classList.add("__demo_fade"));
      setTimeout(() => halo.remove(), 600);
    }, true);
  };
  if (document.readyState === "loading") addEventListener("DOMContentLoaded", add); else add();
})
