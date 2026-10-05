package film

import (
	"encoding/json"
	"strings"

	"github.com/Hy0sh/demo-film/internal/scenario"
)

// cursorJS draws a visible cursor and a halo on every click. The page is
// otherwise filmed untouched. %HIDE% is the CSS injected during filming.
const cursorJS = `(() => {
  const add = () => {
    if (document.getElementById("__demo_cursor")) return;
    const st = document.createElement("style");
    st.textContent = %HIDE%;
    document.documentElement.appendChild(st);
    const c = document.createElement("div");
    c.id = "__demo_cursor";
    c.style.cssText = "position:fixed;z-index:2147483647;width:22px;height:22px;margin:-11px 0 0 -11px;border-radius:50%;background:rgba(36,87,166,.35);border:2px solid #2457a6;pointer-events:none;left:-50px;top:-50px;transition:left .08s linear,top .08s linear";
    document.documentElement.appendChild(c);
    addEventListener("mousemove", e => { c.style.left = e.clientX + "px"; c.style.top = e.clientY + "px"; }, true);
    addEventListener("mousedown", e => {
      const h = document.createElement("div");
      h.style.cssText = "position:fixed;z-index:2147483646;left:" + (e.clientX - 24) + "px;top:" + (e.clientY - 24) + "px;width:48px;height:48px;border-radius:50%;border:3px solid #e8a33a;pointer-events:none;transition:transform .5s,opacity .5s";
      document.documentElement.appendChild(h);
      requestAnimationFrame(() => { h.style.transform = "scale(1.8)"; h.style.opacity = "0"; });
      setTimeout(() => h.remove(), 600);
    }, true);
  };
  if (document.readyState === "loading") addEventListener("DOMContentLoaded", add); else add();
})();`

// InitScript is injected before any page script: the locale (i18next's
// storage key), then the cursor overlay and the hidden elements.
func InitScript(s *scenario.Scenario) string {
	var b strings.Builder
	if s.Locale != "" {
		loc, _ := json.Marshal(s.Locale)
		b.WriteString(`try { localStorage.setItem("i18nextLng", ` + string(loc) + `); } catch (e) {}` + "\n")
	}
	css := ""
	for _, sel := range s.Hide {
		css += sel + "{display:none!important}\n"
	}
	cssJS, _ := json.Marshal(css)
	b.WriteString(strings.Replace(cursorJS, "%HIDE%", string(cssJS), 1))
	return b.String()
}
