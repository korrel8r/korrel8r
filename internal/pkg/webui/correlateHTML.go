package webui

const correlateTemplate = `
{{define "body"}}
    <form>
        <input type="text" id="start" name="start" value="{{.Start}}">
        <label for="start">Start Console URL</label>
        <br>
        <input type="text" id="goal" name="goal" value="{{.Goal}}">
        <label for="goal">Goal Class</label>
        <br>
        <input type="submit" value="Submit">
    </form>

    <p>
	Start class {{.StartClass}} (found {{len .StartObjects}} objects):
        <a href={{.Start}} target="_blank">Console</a>/
        <a href="/stores/{{fullname .StartClass}}/{{.StartRef}}" target="_blank">Raw</a>
    </p>

    {{with .Err}}
	<p>
            Errors: <br>
            <pre>{{printf "%+v" .}}</pre>
	</p>
    {{end}}

    <p>
	{{$result := .Results.Get .GoalClass }}
	Goal class {{fullname .GoalClass}} (found {{len $result.Objects.List}})
	<ul>
	    {{range $result.References.List}}
		<li>
		    <a href="{{refToConsole $result.Class .}}" target="_blank">Console</a>/
		    <a href="/stores/{{fullname $result.Class}}/{{.}}" target="_blank">Raw</a>
		</li>
	    {{end}}
	</ul>
    </p>

    {{if .Diagram}}
	<p>
	    Rules traversed:
	    <object type="image/svg+xml" data="{{.Diagram}}"></object>
	</p>
    {{end}}

    {{with .FollowErr}}
	<p>
            Rule following errors: <br>
            <pre>{{printf "%+v" .}}</pre>
	</p>
    {{end}}

    {{if .Topo}}
	<p>
	    Topological sort:
	    <ul>
		{{range .Topo}}
		    <li> {{fullname .}}</li>
		{{end}}
	    </ul>
	</p>
    {{end}}
{{end}}
`
