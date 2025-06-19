Keep comments to a minimum - only add them if they are absolutely necessary for understanding of the code, otherwise do not add them
Remove superfluous comments in the code
Always aim to write clean, minimal, code that is easy to understand
Always consider alternatives before proceeding and pick the best one
Aim to avoid hard coding numbers and strings as much as possible; isolate them to a separate constants file if needed
Perform security scannning with `golang.org/x/vuln/cmd/govulncheck@latest` (e.g. `govulncheck ./...`, `govulncheck -show verbose ./...`, `govulncheck -mode=module ./...`),`github.com/securego/gosec/v2/cmd/gosec@latest`(e.g. `gosec ./...`, `gosec -fmt=json ./... | jq .`)
Check for outdated dependencies (e.g. `go list -u -m all` and try to update them (e.g. `go get -u ./...;go mod tidy`)
