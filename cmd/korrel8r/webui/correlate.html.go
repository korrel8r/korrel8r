//-*-web-*-
package webui
const correlateHTML = `
{{define "body"}}
  <h1>Correlation Graph</h1>
  <hr>
  <form onsubmit="return validate()">
    <div style="white-space: nowrap">
      <label for="start" title="Consule URL or korrel8r query"><b>Start: </b></label>
      <input type="text" id="start" name="start" value="{{.Start}}" size="50" accesskey="k">
      <label for="domain" title="Only required for korrel8r query"><small>Domain</small></label>
      <input type="text" id="domain" name="domain" value="{{.StartDomain}}">
    </div>
    <div style="white-space: nowrap">
      <b>Goal: </b>
      <br>
      {{range .Goals}}
        <input type="radio" name="goal" id="{{.Label}}" value="{{.Value}}" {{if eq $.Goal .Value}}checked{{end}}>
        <lable for="{{.Label}}">{{.Label}}</lable>
      {{end}}
      <input type="radio" name="goal" id="other" value="other" {{if eq .Goal "other"}}checked{{end}}>
      <label for="other"> Other
        <input type="text" name="other" id="otherText" value="{{.Other}}">
      </label>
      <br>
      <input type="radio" name="goal" id="neighbours" value="neighbours" {{if eq .Goal "neighbours"}}checked{{end}}>
      <label for="neighbours"> <b>Neighbours</b>
        <input type="text" name="neighbours" id="neighboursText" value="{{.Neighbours}}" size="4">
      </label>
    </p>
    <p>
      <b>Options:</b>
      <input type="checkbox" name="short" id="short" value="true" {{if .ShortPaths}}checked{{end}}/>
      <label for="short" title="Follow shortest paths, instead of all paths.">Shortest paths</label>
      <input type="checkbox" name="rules" id="rules" value="true" {{if .RuleGraph}}checked{{end}}/>
      <label for="rules" title="Graph rules without getting results.">Rules</label>
    </p>
    <p>
      <input type="submit" id="submit" value="Update Graph">
      <span id="waiting" style="display:none;"><img src="static/gears.gif" id="loading"></span>
    </p>
  </form>

  <script type="text/javascript">
   <!-- Show spinner while waiting -->
   function validate(form) {
     document.getElementById("submit").style.display="none";
     document.getElementById("waiting").style.display="";
     return true;
   }
  </script>

  {{with .Err}}
    <hr>
    <h3>Errors</h3>
    <div style="white-space: pre-line; border-width:2px; border-style:solid; border-color:red"> {{printf "%+v" .}}</div>
  {{end}}

  {{if .Diagram}}
    <hr>
    <h3>Diagram</h3>
    <p align="center">
      <object type="image/svg+xml" data="{{.Diagram}}"></object><br>
      <a href="{{.DiagramImg}}" target="_blank">Image</a>
      <a href="{{.DiagramTxt}}" target="_blank">Source</a>
    </p>
  {{end}}

  <hr>
  <h3> Detailed Results </h3>
  <p>
    <ul>
      {{with .StartClass}}<li>Start: {{classname .}}</li>{{end}}
      {{with .Depth}}<li>Depth: {{.}}</li>{{end}}
      {{with .GoalClass}}<li>Goal: {{classname .}}</li>{{end}}
    </ul>
    <ul>
      {{range $node := (and .Graph .Graph.AllNodes)}}
        {{if $node.Result.List}}
          <li><code><b>{{classname $node.Class}}</b> ({{len $node.Result.List}})</code>
            <ul>
              {{range ($.Graph.LinesTo .)}}
                {{if .QueryCounts}}
                  <li><code>{{rulename .Rule}}</code></li>
                  <ul>
                    {{range $qc := .QueryCounts.Sort}}
                      <li>
                        <a href="{{queryToConsole $qc.Query}}" target="_blank">Console</a> /
                        <a href="/stores/{{$node.Class}}?query={{json $qc.Query | urlquery}}" target="_blank">Data</a>
                        ({{$qc.Count}})
                        <pre>{{$qc.Query | json}}</pre>
                      </li>
                    {{end}}
                  </ul>
                {{end}}
              {{end}}
            </ul>
          </li>
        {{end}}
      {{end}}
    </ul>
  </p>
{{end}}
`
