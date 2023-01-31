package webui

const correlateHTML = `
{{define "body"}}
    <form>
        <input type="text" id="start" name="start" value="{{.Start}}" accesskey="k"><label for="start">Start URL</label>
	{{with .StartClass}}<code> {{classname .}}</code>{{end}}
        <br>

	<input type="radio" id="logs" name="goals" value="logs" {{if eq .Goals "logs" }}checked="true"{{end}}><label for="logs">Logs</label><br>

	<input type="radio" id="events" name="goals" value="events" {{if eq .Goals "events"}}checked="true"{{end}}><label for="events">Events</label><br>

	<input type="radio" id="metrics" name="goals" value="metrics" {{if eq .Goals "metrics"}}checked="true"{{end}}><label for="metrics">Metrics</label><br>

	<input type="radio" id="other" name="goals" value="other" {{if eq .Goals "other"}}checked="true"{{end}}><label for="events">Other</label><br>

        <input type="text" id="goal" name="goal" value="{{.Goal}}"><label for="goal">Goal Class</label>
	{{with .GoalClass}}<code> {{classname .}}</code>{{end}}
	<br>
        <input type="submit" value="Submit">
    </form>

    {{with .Diagram}}
        <p align="center">
            <object type="image/svg+xml" data="{{.}}"></object>
	    <a href="{{.}}" target="_blank">open</a>
        </p>
    {{end}}

    <hr>
    <p><em>{{.Time}}</em></p>

    <hr>
    {{with .Results}}
        <p>
            Detailed Results:
            <ul>
                {{range .}}
		    {{if .Queries.List}}
			<li>{{template "result" .}}</li>
		    {{end}}
		{{end}}
            </ul>
        </p>
    {{end}}

    {{- with .Err -}}
	<p style="white-space: pre-line; border-width:2px; border-style:solid; border-color:red; padding:1em">
            {{- printf "%+v" . -}}
	</p>
    {{- end -}}

{{end}}

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
		    {{end}}
		</ul>
	    </li>
	{{end}}
	{{with .Rules}}
	    <li>Rules
		<ul> {{range .}}<li><code>{{rulename .}}</code></li>{{end}} </ul>
	    </li>
	{{end}}
	{{with .Errors.List}}
	    <li>Errors
		<ul> {{range .}}<li><code>{{.}}</code></li>{{end}} </ul>
	    </li>
	{{end}}
    </ul>
{{end}}    `
