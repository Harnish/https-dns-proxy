package main

const PageHTML = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>DNS lookup</title>
<style>
:root {
  color-scheme: light dark;
  --bg: #f6f8fa;
  --surface: #ffffff;
  --text: #1b1f24;
  --muted: #59636e;
  --border: #c9d1d9;
  --control-border: #848d97;
  --accent: #0b5fff;
  --on-accent: #ffffff;
  --accent-tint: #e6eeff;
  --danger: #b42318;
  --font: system-ui, -apple-system, "Segoe UI", Roboto, sans-serif;
  --mono: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
}
@media (prefers-color-scheme: dark) {
  :root {
    --bg: #0f1318;
    --surface: #161b22;
    --text: #e6edf3;
    --muted: #9aa4b0;
    --border: #3a424c;
    --control-border: #6e7781;
    --accent: #4c8dff;
    --on-accent: #06101f;
    --accent-tint: #172640;
    --danger: #ff8f87;
  }
}
* { box-sizing: border-box; }
body {
  margin: 0;
  background: var(--bg);
  color: var(--text);
  font: 1rem/1.5 var(--font);
}
main { max-width: 52rem; margin: 0 auto; padding: 1.5rem 1rem 3rem; }
h1 { font-size: 1.5rem; line-height: 1.25; margin: 0 0 .25rem; }
h2 { font-size: 1.125rem; line-height: 1.3; margin: 2rem 0 .5rem; }
.lede { margin: 0 0 1.5rem; color: var(--muted); }
code, pre { font-family: var(--mono); font-size: .875rem; }
form { display: grid; gap: 1rem; }
label.field { display: block; font-weight: 600; margin-bottom: .25rem; }
.row { display: flex; flex-wrap: wrap; gap: .5rem; }
input[type=text] {
  flex: 1 1 10rem;
  min-width: 0;
  min-height: 44px;
  padding: .5rem .75rem;
  font: inherit;
  color: inherit;
  background: var(--surface);
  border: 1px solid var(--control-border);
  border-radius: .375rem;
  caret-color: var(--accent);
}
button {
  min-height: 44px;
  padding: .5rem 1.25rem;
  font: inherit;
  font-weight: 600;
  color: var(--on-accent);
  background: var(--accent);
  border: 1px solid var(--accent);
  border-radius: .375rem;
  cursor: pointer;
  transition: filter 150ms ease-out;
}
button:hover { filter: brightness(1.08); }
button:active { filter: brightness(.95); }
button:disabled { opacity: .65; cursor: progress; filter: none; }
:focus-visible { outline: 2px solid var(--accent); outline-offset: 2px; }
fieldset { margin: 0; padding: 0; border: 0; min-width: 0; }
legend { font-weight: 600; padding: 0; margin-bottom: .25rem; }
.types { display: flex; flex-wrap: wrap; gap: .5rem; }
.types label {
  display: inline-flex;
  align-items: center;
  gap: .5rem;
  min-height: 44px;
  padding: .25rem .875rem;
  background: var(--surface);
  border: 1px solid var(--control-border);
  border-radius: .375rem;
  cursor: pointer;
}
.types label:has(input:checked) { background: var(--accent-tint); border-color: var(--accent); }
.types input { accent-color: var(--accent); width: 1.125rem; height: 1.125rem; margin: 0; }
#status { margin: 0 0 .75rem; min-height: 1.5rem; color: var(--muted); }
#status.err { color: var(--danger); font-weight: 600; }
pre {
  margin: 0;
  padding: 1rem;
  overflow: auto;
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: .375rem;
  scrollbar-color: var(--border) transparent;
}
pre:empty { display: none; }
.shortcut { margin: 1rem 0 0; overflow-wrap: anywhere; }
.shortcut a { color: var(--accent); text-underline-offset: .2em; }
::selection { background: var(--accent-tint); color: var(--text); }
@media (prefers-reduced-motion: reduce) { button { transition: none; } }
</style>
</head>
<body>
<main>
<h1>DNS lookup</h1>
<p class="lede">Resolve a name through this server and see the JSON that <code>/resolve</code> returns.</p>

<form id="f" action="/query" method="get">
  <div>
    <label class="field" for="name">Name</label>
    <div class="row">
      <input type="text" id="name" name="name" required placeholder="example.com" autocomplete="off" autocapitalize="none" spellcheck="false" inputmode="url">
      <button type="submit" id="go">Resolve</button>
    </div>
  </div>
  <fieldset>
    <legend>Record type</legend>
    <div class="types">
      <label><input type="radio" name="type" value="1"> A</label>
      <label><input type="radio" name="type" value="28"> AAAA</label>
      <label><input type="radio" name="type" value="5"> CNAME</label>
      <label><input type="radio" name="type" value="15"> MX</label>
      <label><input type="radio" name="type" value="255" checked> ANY</label>
    </div>
  </fieldset>
</form>

<h2 id="rh">Result</h2>
<p id="status" role="status">Enter a name and choose a record type.</p>
<pre id="json" role="region" tabindex="0" aria-labelledby="rh"></pre>
<p class="shortcut" id="shortcut" hidden>Shortcut URL: <a id="directurl"></a></p>
</main>
<script>
(function () {
  var form = document.getElementById('f');
  var nameEl = document.getElementById('name');
  var go = document.getElementById('go');
  var statusEl = document.getElementById('status');
  var out = document.getElementById('json');
  var shortcut = document.getElementById('shortcut');
  var link = document.getElementById('directurl');
  var ctl = null;

  function selectedType() {
    var r = form.querySelector('input[name=type]:checked');
    return r ? r.value : '255';
  }
  function setStatus(msg, isErr) {
    statusEl.textContent = msg;
    statusEl.className = isErr ? 'err' : '';
  }
  var explain = {
    'NXDOMAIN': 'NXDOMAIN: that name does not exist.',
    'SERVFAIL': 'SERVFAIL: the upstream resolver could not answer. Try again.',
    'REFUSED': 'REFUSED: the upstream resolver declined this query.',
    'upstream DNS error': 'The upstream DNS server could not be reached. Try again.',
    'missing name': 'Enter a name to look up.',
    'invalid type': 'That record type is not valid. Choose one from the list.'
  };
  function summarize(data, code) {
    if (!data) return { msg: 'Server error (HTTP ' + code + '). Try again.', err: true };
    if (data.Comment) return { msg: explain[data.Comment] || data.Comment, err: true };
    var n = (data.Answer || []).length;
    if (n === 0) return { msg: 'No records of this type for this name.', err: false };
    return { msg: n + (n === 1 ? ' answer' : ' answers'), err: false };
  }
  function done() {
    go.disabled = false;
    go.textContent = 'Resolve';
  }
  function lookup(name, type) {
    if (ctl) ctl.abort();
    var mine = ctl = new AbortController();
    var q = '/resolve?name=' + encodeURIComponent(name) + '&type=' + encodeURIComponent(type);
    link.href = q;
    link.textContent = location.origin + q;
    shortcut.hidden = false;
    go.disabled = true;
    go.textContent = 'Resolving…';
    setStatus('Looking up ' + name + '…', false);
    fetch(q, { signal: mine.signal })
      .then(function (r) {
        return r.json().then(
          function (d) { return [r.status, d]; },
          function () { return [r.status, null]; }
        );
      })
      .then(function (a) {
        if (mine !== ctl) return;
        var s = summarize(a[1], a[0]);
        setStatus(s.msg, s.err);
        out.textContent = a[1] ? JSON.stringify(a[1], undefined, 2) : '';
        done();
      })
      .catch(function (e) {
        if (mine !== ctl) return;
        setStatus('Could not reach the server. Check your connection and try again.', true);
        out.textContent = '';
        done();
      });
  }

  form.addEventListener('submit', function (e) {
    e.preventDefault();
    var name = nameEl.value.trim();
    if (!name) { nameEl.focus(); return; }
    var type = selectedType();
    history.replaceState(null, '', '/query?name=' + encodeURIComponent(name) + '&type=' + encodeURIComponent(type));
    lookup(name, type);
  });

  var params = new URLSearchParams(location.search);
  var qname = params.get('name');
  var qtype = params.get('type') || '255';
  var radios = form.querySelectorAll('input[name=type]');
  for (var i = 0; i < radios.length; i++) radios[i].checked = radios[i].value === qtype;
  if (qname) {
    nameEl.value = qname;
    lookup(qname, qtype);
  }
})();
</script>
</body>
</html>
`
