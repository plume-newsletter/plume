<!-- Thanks for contributing to Plume! -->

## What & why

<!-- What does this change and why? Link any related issue: Closes #123 -->

## How tested

<!-- Commands run / scenarios checked. Backend: gofmt/vet/build/test. Frontend: build/test/oxlint. -->

## Checklist

- [ ] Backend green: `gofmt -l .` clean, `go vet ./...`, `go test ./...`
- [ ] Frontend green (if touched): `npm run build`, `npm test`, `npx oxlint`
- [ ] Generated code (`internal/store/gen/`) regenerated via `sqlc generate`, not hand-edited
- [ ] Commits signed off (DCO): `git commit -s`
