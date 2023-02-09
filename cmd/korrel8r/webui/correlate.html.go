//-*-web-*-

package webui

const correlateHTML = `
{{define "body"}}
  <h1>Correlation</h1>
  <p>Hover for hints, for background see <a href="https://github.com/korrel8r/korrel8r">korrel8r project documentation</a></p>
  <form onsubmit="return validate()">
    <hr>
    <div title="Choose how to find start objects for correlation">
      <b>Start</b>: {{template "choice" mkslice "startis" .StartIs " " "Openshift Console" "korrel8r Query"}}
    </div>
    <ul>
      {{if eq .StartIs "Openshift Console"}}
        <li>Browse to the the resource of interest in the <a href="{{.ConsoleURL}}">Openshift console</a>, copy the URL, and paste it here</li>
        <li>
          <label for="start" title=>Console URL</label>
          <input type="text" id="start" name="start" value="{{.Start}}" size="80" accesskey="k">
        </li>
      {{else}}
        <li>
          <span title="Enter a korrel8r domain name and query string (TODOO query docs)">
            <label for="domain">Domain</label>
            <input type="text" id="domain" name="domain" value="{{.StartDomain}}">
            <label for="start">Query</label>
            <input type="text" id="start" name="start" value="{{.Start}}" accesskey="k">
          </span>
        </li>
      {{end}}
    </ul>

    <div title="Choose the goal class for correlation">
      <b>Goal</b>: {{template "choice" mkslice "goals" .Goals " " "logs" "events" "metrics" "other"}}
    </div>
    <ul>
      {{if eq .Goals "other"}}
        <li>
          <label for="goal">Goal class domain/name</label>
          <input type="text" id="goal" name="goal" value="{{.Goal}}">
        </li>
      {{end}}
    </ul>

    <p>
      <b>Options</b>:
      <input type="checkbox" id="full" name="full" value="true" {{if .Full}}checked{{end}} />
      <span title="Show the full search graph, including empty nodes/links"><label for="full">full graph</label></span>
      <input type="checkbox" id="all" name="all" value="true" {{if .All}}checked{{end}} />
      <span title="Experimenta: search all possible paths, not just shortest paths" ><label for="all">all paths</label></span>
    </p>
    <p>
      <input type="submit" id="submitID">
      <span id="waiting" style="display:none;"><img src="static/gears.gif" id="loading"></span>
    </p>
  </form>
  <script type="text/javascript">
   <!-- Show spinner while waiting -->
   function validate(form) {
     document.getElementById("submitID").style.display="none";
     document.getElementById("waiting").style.display="";
     return true;
   }
  </script>
  <hr>

  <p><code>{{with .StartClass -}}{{classname .}}{{end}}{{with .GoalClass}} -> {{classname .}}{{end}}</code></p>
  {{with .Diagram}}
    <p align="center">
      <object type="image/svg+xml" data="{{.}}"></object>
    </p>
  {{end}}
  <p><em>{{.Time}}</em></p>

  <hr>
  {{with .Results}}
    <p>
      Detailed Results:
      <ul>
        {{range . -}}
          {{if .Queries.List -}}
            <li>{{template "result" . -}}</li>
          {{end -}}
        {{end -}}
      </ul>
    </p>
  {{end -}}
  {{- with .Err -}}
    <p style="white-space: pre-line; border-width:2px; border-style:solid; border-color:red; padding:1em">
      {{- printf "%+v" . -}}
    </p>
  {{- end -}}
{{end -}}

{{define "result"}}
  <code>{{classname .Class}}</code> (found {{.Objects}})
  {{with .Queries.List}}
    <ul>
      {{range .}}
        <li>
          <a href="{{queryToConsole .}}" target="_blank">Console</a> /
          <a href="/stores/{{.Class.Domain}}?query={{json . | urlquery}}" target="_blank">Data</a>
        </li>
      {{end -}}
    </ul>
  {{end -}}
{{end -}}

{{/* choice generates radio buttons. Dot is a slice of: [name, checked, separator, values...] */}}
{{define "choice"}}
  {{$name := index . 0}}
  {{$checked := index . 1}}
  {{$separator := index . 2}}
  {{$choices := slice . 3}}
  {{range $i, $c := $choices}}
    {{if $i}}{{asHTML $separator}}{{end -}}
    <input onchange="this.form.submit();" type="radio" name="{{$name}}" value="{{$c}}"  id="{{$c}}"  {{if eq $checked $c}}checked{{end}}>
    <label for="{{$c}}">{{$c}}</label>
  {{end -}}
{{end -}}
`
