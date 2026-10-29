# Agent Entry Point

Keep changes focused and preserve unrelated working-tree changes. Before editing, inspect the relevant implementation and nearby tests; follow local package conventions.

Load the guidance relevant to your task rather than reading everything up front:

- [DEVELOPER.md](DEVELOPER.md): contributor workflow, coding conventions, generation, and validation.
- [README.md](README.md): product and usage overview.
- [CONTRIBUTING.md](CONTRIBUTING.md): contribution requirements.
- [doc/dev/README.md](doc/dev/README.md): technical development procedures and measurement workflows.
- `make help`: available build targets and variables.

For rules, see `pkg/rules/quickrules/doc.go` for compiled quickrules, and the runtime rule documentation in `etc/korrel8r/rules/` and the user guide for configuration rules. For generated code, edit its source and use `make generate`; do not hand-edit generated files.

Run the narrowest relevant tests first. Check `git diff` and `git status` before finishing; report commands run and any skipped cluster-dependent checks. See [DEVELOPER.md](DEVELOPER.md) for validation details.
