{{template "common/application.html" .}}

{{define "body"}}
<div>
This is a test ???
<div>
</div>
{{if .TrueCond}}
true Condition
{{end}}
<div>
</div>
{{with .User}}
{{.Name}}; {{.Age}}; {{.Sex}}
{{end}}
</div>
<div>
{{range .Nums}}
{{.}}
{{end}}
</div>
{{end}}
