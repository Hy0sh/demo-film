package film_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// appHTML is a tiny single-page app covering the whole vocabulary: a nav
// menu (button parent, link child), form controls, a table with icon-only
// buttons, a hover tooltip, a dialog with a select and a Next button, a
// confirm dialog, and a link opening a new tab.
const appHTML = `<!doctype html>
<html><head><meta charset="utf-8"><title>Demo shop</title>
<style>body{font:16px sans-serif;margin:20px} dialog,[role=dialog]{border:1px solid #888;padding:16px;background:#fff;margin:8px}</style></head>
<body>
<nav>
  <button id="parent" onclick="document.getElementById('sub').hidden=!document.getElementById('sub').hidden">Settings</button>
  <ul id="sub" hidden><li><a href="#profile" onclick="show('Profile page')">Profile</a></li></ul>
  <a href="#decoy" onclick="show('Decoy page')">Profile settings</a>
</nav>
<h1>Home</h1>
<p id="state"></p>
<label>Name <input id="name"></label>
<input placeholder="Search" onkeydown="if(event.key==='Enter')show('Submitted: '+this.value)">
<label>Secret <input type="password"></label>
<table>
  <tr><td>Ada</td><td><button onclick="show('Editing Ada')">&#9998;</button><button onclick="show('Removed Ada')">&#128465;</button></td></tr>
  <tr><td>Grace</td><td><button aria-label="Edit" onclick="show('Editing Grace')">&#9998;</button></td></tr>
</table>
<span id="help" onmouseenter="show('Help tooltip')">Help</span>
<button onclick="openWizard()">Open wizard</button>
<a href="/docs" target="_blank">Docs</a>
<div id="slot"></div>
<script>
const show = t => { document.getElementById('state').textContent = t; };
function openWizard() {
  const d = document.createElement('div'); d.setAttribute('role', 'dialog');
  d.innerHTML = '<h2>Wizard</h2><label>Colour <select><option>Red</option><option>Blue</option></select></label>' +
    '<p id="wiz">Pick a colour</p><button id="next">Next</button><button id="close">Close</button>';
  document.getElementById('slot').appendChild(d);
  d.querySelector('#next').onclick = () => { d.querySelector('#wiz').textContent = 'Step two: ' + d.querySelector('select').value; };
  d.querySelector('#close').onclick = () => {
    const c = document.createElement('div'); c.setAttribute('role', 'dialog');
    c.innerHTML = '<p>Abandon changes?</p><button>Keep</button><button>Discard</button>';
    document.getElementById('slot').appendChild(c);
    c.querySelectorAll('button')[1].onclick = () => { document.getElementById('slot').innerHTML = ''; show('Wizard closed'); };
  };
}
</script>
</body></html>`

// newApps serves the app and, on another origin, a mail catcher.
func newApps(t *testing.T) (app, mail *httptest.Server) {
	t.Helper()
	am := http.NewServeMux()
	am.HandleFunc("/docs", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "<h1>Documentation</h1>")
	})
	am.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, appHTML)
	})
	app = httptest.NewServer(am)
	mail = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, "<h1>Inbox</h1><p>Welcome mail</p>")
	}))
	t.Cleanup(func() { app.Close(); mail.Close() })
	return app, mail
}
