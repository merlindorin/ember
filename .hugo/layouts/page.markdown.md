{{- .Title | replaceRE "\n" " " | printf "# %s" }}

{{ with .Params.markdown }}{{ . }}{{ else }}{{ .RawContent }}{{ end }}
