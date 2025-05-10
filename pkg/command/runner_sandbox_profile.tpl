{{ if .CustomProfile }}
{{ .CustomProfile }}
{{ else }}
(version 1)

(allow default)

{{ if .AllowNetworking }}
(allow network*)
{{ else }}
(deny network*)
{{ end }}

{{ if .AllowUserFolders }}
(deny file-read* (subpath "/Users"))
{{ else }}
(deny file-read-data (regex "^/Users/.*/(Documents|Desktop|Downloads|Pictures|Movies|Music)"))
{{ end }}

{{ range .AllowReadFolders }}
(allow file-read* (subpath "{{ . }}"))
{{ end }}
{{ end }}