package main

const PageHTML = `
<html lang=en>
<head>
  <meta http-equiv="Content-Type" content="text/html; charset=utf-8">
  <title>HTTP(s) DNS lookup</title>
  <script language="JavaScript">
  // http://stackoverflow.com/questions/12460378/how-to-get-json-from-url-in-javascript
  var getJSON = function(url, callback) {
    var xhr = new XMLHttpRequest();
    xhr.open('GET', url, true);
    xhr.responseType = 'json';
    xhr.onload = function() {
      var status = xhr.status;
      if (status == 200) {
        callback(null, xhr.response);
      } else {
        callback(status);
      }
    };
    xhr.send();
   };
   // http://stackoverflow.com/questions/901115/how-can-i-get-query-string-values-in-javascript
   function getParameterByName(name, url) {
    if (!url) url = window.location.href;
    name = name.replace(/[\[\]]/g, "\\$&");
    var regex = new RegExp("[?&]" + name + "(=([^&#]*)|&|#|$)"),
        results = regex.exec(url);
    if (!results) return null;
    if (!results[2]) return '';
    return decodeURIComponent(results[2].replace(/\+/g, " "));
}

   function lookup(name, type) {
		var q = '/resolve?name=' + encodeURIComponent(name) + '&type=' + encodeURIComponent(type);
		var url = window.location.origin + q;
		var a = document.createElement('a');
		a.href = url;
		a.textContent = url;
		var d = document.getElementById("directurl");
		d.textContent = '';
		d.appendChild(a);
		getJSON(q, function(err, data) {
			if (err != null) {
				alert('Something went wrong: ' + err);
			} else {
				document.getElementById("json").textContent = JSON.stringify(data, undefined, 2);
			}
		});
	}
	function ResolveName() {
		var name = document.getElementById('name').value;
		var types = document.getElementsByName('type');
		var mytype = 255;
		for (var i = 0; i < types.length; i++) {
			if (types[i].checked) {
				mytype = types[i].value;
			}
		}
		lookup(name, mytype);
	}
	function loadURL() {
		var myname = getParameterByName("name");
		var mytype = getParameterByName("type") || "255";
		var radio = document.getElementById("type-" + mytype);
		if (radio) radio.checked = true;
		if (myname) {
			document.getElementById("name").value = myname;
			lookup(myname, mytype);
		}
	}

  </script>
</head>
<body onload="loadURL()">
<form>
<table width=100% bgcolor="#808080">
<tr><td>DNS Lookup</td><td><input type="text" name="name" id="name"></td><td><input type="button" name="lookup" value="Resolve" OnClick=ResolveName()></td></tr>
</table>
<table>
  <tr>
    <td><input type="radio" name="type" id="type-1" value="1"> A </td>
	<td><input type="radio" name="type" id="type-28" value="28"> AAAA </td>
	<td><input type="radio" name="type" id="type-5" value="5"> CNAME </td>
	<td><input type="radio" name="type" id="type-15" value="15"> MX </td>
	<td><input type="radio" name="type" id="type-255" value="255" checked> ANY </td>
   </tr>
</table>
</form>
<hr>
Results:<br>
<pre id="json"></pre>
Short cut URL: <p id="directurl"></p>
</body>
</html>
`
