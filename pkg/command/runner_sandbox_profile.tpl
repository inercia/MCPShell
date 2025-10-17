{{ if .CustomProfile }}
{{ .CustomProfile }}
{{ else }}
(version 1)

(allow default)

;; Protect system directories from writes
(deny file-write* (subpath "/bin"))
(deny file-write* (subpath "/sbin"))
(deny file-write* (subpath "/usr/bin"))
(deny file-write* (subpath "/usr/sbin"))
(deny file-write* (subpath "/usr/local/bin"))
(deny file-write* (subpath "/usr/local/sbin"))
(deny file-write* (subpath "/etc"))
(deny file-write* (subpath "/System"))
(deny file-write* (subpath "/Library"))
(deny file-write* (literal "/var/root"))
(deny file-write* (subpath "/var/db"))
(deny file-write* (subpath "/private/etc"))
(deny file-write* (subpath "/private/var/db"))

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

{{ range .AllowReadFiles }}
(allow file-read* (literal "{{ . }}"))
{{ end }}

{{ range .AllowWriteFolders }}
(allow file-write* (subpath "{{ . }}"))
{{ end }}

{{ range .AllowWriteFiles }}
(allow file-write* (literal "{{ . }}"))
{{ end }}

{{ end }}