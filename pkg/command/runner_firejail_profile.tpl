{{ if .CustomProfile }}
{{ .CustomProfile }}
{{ else }}
# Basic profile for firejail
# Applied restrictions based on provided options

# Network restrictions
{{ if .AllowNetworking }}
# Allow networking
{{ else }}
# Disable networking
net none
{{ end }}

# File system restrictions
{{ if .AllowUserFolders }}
# Allow access to user folders
{{ else }}
# Deny access to user folders (except home directory structure)
blacklist ${HOME}/Documents
blacklist ${HOME}/Desktop
blacklist ${HOME}/Downloads
blacklist ${HOME}/Pictures
blacklist ${HOME}/Videos
blacklist ${HOME}/Music
{{ end }}

# Allow specific read folders
{{ range .AllowReadFolders }}
whitelist {{ . }}
read-only {{ . }}
{{ end }}

# Allow specific read files
{{ range .AllowReadFiles }}
whitelist {{ . }}
read-only {{ . }}
{{ end }}

# Allow specific write folders
{{ range .AllowWriteFolders }}
whitelist {{ . }}
{{ end }}

# Allow specific write files
{{ range .AllowWriteFiles }}
whitelist {{ . }}
{{ end }}

# Always apply basic security features
seccomp
caps.drop all
noroot
{{ end }} 