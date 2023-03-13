//-*-web-*-

package webui

const correlateHTML = `
{{define "body"}}
  <h1>Correlation View</h1>
  <form onsubmit="return validate()">
    <hr>
    <div title="Choose how to find start objects for correlation">
      <b>Start</b>: {{template "choice" mkslice "startis" .StartIs " " "Openshift Console URL" "korrel8r Query"}}
    </div>
    <ul>
      {{if eq .StartIs "Openshift Console"}}
        <p  title="Openshift Console Resouce URL">
          <label for="start">URL</label>
          <input type="text" id="start" name="start" value="{{.Start}}" size="80" accesskey="k">
        </p>
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
      <b>Goal</b>: {{template "choice" mkslice "goalChoice" .GoalChoice " " "logs/infrastructure" "k8s/Event" "metric/metric" "other" "neighbours"}}
    </div>
    <ul>
      {{if eq .GoalChoice "other"}}
        <li>
          <label for="goal">Goal class domain/name</label>
          <input type="text" id="goal" name="goal" value="{{.Goal}}">
        </li>
      {{end}}
      {{if eq .GoalChoice "neighbours"}}
        <li>
          <label for="goal">Depth:</label>
          <input type="text" id="goal" name="goal" value="{{.Depth}}">
        </li>
      {{end}}
    </ul>

    <p>
      <b>Options</b>:
      {{template "check" mkslice "short" .ShortPaths}}
      <span title="Follow only the shortest paths, instead of all paths."><label for="short">Shortest paths</label></span>
      {{template "check" mkslice "noresult" .NoResult}}
      <span title="Graph rules without applying them to get results."><label for="noresult">No Results</label></span>
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

  {{- with .Err -}}
    Errors
    <p style="white-space: pre-line; border-width:2px; border-style:solid; border-color:red; padding:1em">
      {{- printf "%+v" . -}}
    </p>
  {{- end -}}

  {{if .Diagram}}
    <p align="center">
      <object type="image/svg+xml" data="{{.Diagram}}"></object><br>
      <a href="{{.DiagramImg}}" target="_blank">Image</a>
      <a href="{{.DiagramTxt}}" target="_blank">Source</a>
    </p>
  {{end}}
  <p><em>{{.Time}}</em></p>

  <hr>
  <p>
    Detailed Results:
    <ul>
      {{range and .Graph .Graph.AllNodes}}
        {{$node := .}}
        {{range ($.Graph.LinesTo .)}}
          {{ $rule := .Rule }}
          {{if .QueryCounts}}
            <li><code>{{$rule}} {{classname $rule.Start}} -> <b>{{classname $rule.Goal}}</b></code>
              {{ range $qc := .QueryCounts.Sort }}
                <ul>
                  <li>
                    <a href="{{queryToConsole $qc.Query}}" target="_blank">Console</a> /
                    <a href="/stores/{{$rule.Goal.Domain}}?query={{json $qc.Query | urlquery}}" target="_blank">Data</a>
                    ({{$qc.Count}})
                    <pre>{{$qc.Query | json}}</pre>
                  </li>
                </ul>
              {{end}}
            </li>
          {{end}}
        {{end}}
      {{end}}
    </ul>
  </p>

{{end -}}

{{define "result"}}
  {{with .Queries.List}}
  {{end -}}
{{end -}}

{{/* choice generates radio buttons. Dot is a slice of: [name, var, separator, values...] */}}
{{define "choice"}}
  {{$name := index . 0}}
  {{$var := index . 1}}
  {{$separator := index . 2}}
  {{$choices := slice . 3}}
  {{range $i, $c := $choices}}
    {{if $i}}{{asHTML $separator}}{{end -}}
    <input type="radio" name="{{$name}}"  id="{{$c}}"  value="{{$c}}" {{if eq $var $c}}checked{{end}}/>
    <label for="{{$c}}">{{$c}}</label>
  {{end -}}
{{end -}}

{{/* check generates a check box. Dot is a slice of: [name, var] */}}
{{define "check"}}
  {{$name := index . 0}}
  {{$var := index . 1}}
  <input type="checkbox" name="{{$name}}" id="{{$name}}" value="true" {{if $var}}checked{{end}}/>
{{end -}}
`
