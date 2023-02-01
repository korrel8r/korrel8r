//-*-web-*-

package webui

const correlateHTML = `
{{define "body"}}
  <form onsubmit="return validate()">
    <p>
      Start: {{template "choice" mkslice "startis" .StartIs " " "console" "query"}}<br>
      {{if eq .StartIs "console"}}
        <input type="text" id="start" name="start" value="{{.Start}}" accesskey="k">
        <label for="start">Console URL</label>
      {{else}}
        <label for="domain">Domain</label>
        <input type="text" id="domain" name="domain" value="{{.StartDomain}}">
        <label for="start">Query</label>
        <input type="text" id="start" name="start" value="{{.Start}}" accesskey="k">
      {{end}}
    </p>
    <p>
      Goal: {{template "choice" mkslice "goals" .Goals " " "logs" "events" "metrics" "other"}}<br>
      {{if eq .Goals "other"}}
        <label for="goal">Goal Class</label>
        <input type="text" id="goal" name="goal" value="{{.Goal}}">
      {{end}}
    </p>
    <input type="submit" id="submit" >
    <span id="waiting" style="display:none;"><img src="static/gears.gif" id="loading"></span>
  </form>
  <!-- Submit the form implicitly on radio buttons -->
  <script type="text/javascript">
   const form = document.querySelector("form")
   form.querySelectorAll("input[type=radio]").forEach((i) => {
     i.addEventListener("change", () => { form.submit(); });
   });
   function validate(form) {
     document.getElementById("submit").style.display="none";
     document.getElementById("waiting").style.display="";
     return true;
   }
  </script>
  <hr>

  <p><code>{{with .StartClass -}}{{classname .}}{{end}}{{with .GoalClass}} -> {{classname .}}{{end}}</code></p>
  {{with .Diagram}}
    <p align="center">
      <object type="image/svg+xml" data="{{.}}"></object>
      <a href="{{.}}" target="_blank">open</a>
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
  Goal <code>{{classname .Class}}</code> (found {{len .Objects.List}})
  <ul>
    {{with .Queries.List}}
      <li>Queries
        <ul>
          {{range .}}
            <li>
              <a href="{{queryToConsole .}}" target="_blank">Console</a> /
              <a href="/stores/{{.Class.Domain}}?query={{json . | urlquery}}" target="_blank">Data</a>
            </li>
          {{end -}}
        </ul>
      </li>
    {{end -}}
    {{with .Rules}}
      <li>Rules
        <ul> {{range .}}<li><code>{{rulename .}}</code></li>{{end -}} </ul>
      </li>
    {{end -}}
    {{with .Errors.List}}
      <li>Errors
        <ul> {{range .}}<li><code>{{.}}</code></li>{{end -}} </ul>
      </li>
    {{end -}}
  </ul>
{{end -}}

{{/* choice generates radio buttons. Dot is a slice of: [name, checked, separator, values...] */}}
{{define "choice"}}
  {{$name := index . 0}}
  {{$checked := index . 1}}
  {{$separator := index . 2}}
  {{$choices := slice . 3}}
  {{range $i, $c := $choices}}
    {{if $i}}{{asHTML $separator}}{{end -}}
    <input type="radio" name="{{$name}}" value="{{$c}}"  id="{{$c}}"  {{if eq $checked $c}}checked{{end}}>
    <label for="{{$c}}">{{$c}}</label>
  {{end -}}
{{end -}}
`
