package webui

const correlateHTML = `
{{define "body"}}
    <form>
        <input type="text" id="start" name="start" value="{{.Start}}" accesskey="k">
        <label for="start">Start URL</label>
        <br>
        <input type="text" id="goal" name="goal" value="{{.Goal}}">
        <label for="goal">Goal Class</label>
        <br>
        <input type="submit" value="Submit">
    </form>

    <p>
	Start: <code>{{classname .StartClass}}</code> (found {{len .StartObjects}})
	<br>
	Goal: <code>{{classname .GoalClass}}</code>  (found {{with and .Results.Last .Results.Last.Objects}}{{len .List}}{{else}}0{{end}})
    </p>

    {{if .Diagram}}
	<p>
	    <object type="image/svg+xml" data="{{.Diagram}}" align="center"></object>
	</p>
    {{end}}

    {{with .Results}}
	<p>
	    Full results ({{len .}} stages):
	    <ul>
		{{range .}}<li>{{template "result" .}}</li>{{end}}
	    </ul>
	</p>
    {{end}}
    <hr>
    <p><em>{{.Time}}</em></p>
{{end}}

{{define "result"}}
    <code>{{classname .Class}}</code> (found {{with .Objects}}{{len .List}}{{else}}0{{end}})
    <ul>
	{{range .Queries.List}}
	    <span style="white-space: nowrap;">
		<li><a href="{{queryToConsole .}}" target="_blank"><code>{{.}}</code></a></li>
	    </span>
	{{end}}
    </ul>
{{end}}
`
