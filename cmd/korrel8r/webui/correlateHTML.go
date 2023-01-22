package webui

const correlateHTML = `
{{define "body"}}
    <form>
        <input type="text" id="start" name="start" value="{{.Start}}">
        <label for="start">Start URL</label>
        <br>
        <input type="text" id="goal" name="goal" value="{{.Goal}}">
        <label for="goal">Goal Class</label>
        <br>
        <input type="submit" value="Submit">
    </form>

    <p>
	Start <mark>{{fullname .StartClass}}</mark> ({{len .StartObjects}} objects):
        <a href={{.Start}} target="_blank">Console</a>/
        <a href="/stores/{{fullname .StartClass}}/{{.StartQuery}}" target="_blank">Raw</a>
    </p>

    {{with .Err}}
	<p>
            Errors: <br>
            <pre>{{printf "%+v" .}}</pre>
	</p>
    {{end}}

    <p>
	Goal {{template "result" .Results.Last}}
    </p>

    {{if .Diagram}}
	<p>
	    Rules traversed:
	    <object type="image/svg+xml" data="{{.Diagram}}" border="4"></object>
	</p>
    {{end}}

    <p>
	Full results ({{len .Results}} stages):
	<ul>
	    {{range .Results}}<li>{{template "result" .}}</li>{{end}}
	</ul>
    </p>
    <hr>
    <p><em>{{.Time}}</em></p>
{{end}}

{{define "result"}}
    <mark>{{.Class}}</mark> ({{if .Objects}}{{len .Objects.List}}{{else}}0{{end}} objects)
    <ul>
	{{range .Queries.List}}
	    <span style="white-space: nowrap;">
		<li><a href="{{queryToConsole .}}" target="_blank"><code>{{.}}</code></a></li>
	    </span>
	{{end}}
    </ul>
{{end}}
`
