# Contributing to Korrel8r

Thank you for your interest in contributing to korrel8r! This guide covers how to get started, submit changes, and what to expect during review.

> **New to korrel8r?** Read the [README](README.md) for a project overview and the [User Guide](https://korrel8r.github.io/korrel8r) to understand how korrel8r works from a user perspective.

## Ways to Contribute

- **Bug reports** - File an [issue](https://github.com/korrel8r/korrel8r/issues) with steps to reproduce.
- **Feature requests** - Open an [issue](https://github.com/korrel8r/korrel8r/issues) or start a [discussion](https://github.com/korrel8r/korrel8r/discussions).
- **Correlation rules** - Add new rules in `etc/korrel8r/rules/` without changing Go code. See [Writing Rules](https://korrel8r.github.io/korrel8r/docs/writing-rules/).
- **Code contributions** - Bug fixes, new features, new domains, tests, and documentation.

## Getting Started

### Prerequisites

- **Go 1.26+** - [Installation guide](https://golang.org/doc/install)
- **Make** - Standard build tool
- **Container runtime** - Docker or Podman (for image builds)
- **OpenShift/Kubernetes cluster** - Only needed for cluster tests

### Fork, Clone, and Build

```bash
# Fork the repository on GitHub, then:
git clone https://github.com/YOUR_USERNAME/korrel8r.git
cd korrel8r
make install        # Build and install to $GOPATH/bin
```

## Making Changes

### Development Loop

```bash
make help              # Show available targets and variables
```

1. Create a feature branch from `main`.
2. Make your changes.
3. Run tests:
   ```bash
   make test              # All tests (requires cluster)
   make test-no-cluster   # Tests that don't require a cluster
   ```
4. Run the full lint and test suite before committing:
   ```bash
   make all
   ```

### Coding Guidelines

- Follow standard Go formatting (`make lint` enforces this).
- Follow existing patterns in the codebase.
- Add tests for new functionality.
- See [DEVELOPER.md](DEVELOPER.md) for architecture details, logging levels, debugging, and advanced workflows.

## Submitting a Pull Request

1. Push your branch to your fork.
2. Open a pull request against `main` on [korrel8r/korrel8r](https://github.com/korrel8r/korrel8r).
3. Write a clear description of the change and why it's needed.
4. Ensure CI passes - the PR must pass `make all` (lint + tests).
5. Respond to review feedback.

### What Makes a Good PR

- **Focused scope** - One logical change per PR. Separate refactoring from feature work.
- **Tests included** - New features and bug fixes should include tests. Cluster tests should have "Cluster" in the test name.
- **Clear commit messages** - Describe what changed and why.

### Review Process

A maintainer will review your PR. Expect feedback on code style, test coverage, and design. Reviews aim to be constructive - don't hesitate to ask questions or push back.

## Reporting Issues

When filing a bug report, include:

- Korrel8r version (`korrel8r version`)
- Steps to reproduce the issue
- Expected vs. actual behavior
- Relevant logs (use `korrel8r -v3` for verbose output)

## Project Resources

- **[User Guide](https://korrel8r.github.io/korrel8r)** - Complete user documentation
- **[Developer Guide](DEVELOPER.md)** - Architecture, debugging, and advanced development workflows
- **[Project Board](https://github.com/orgs/korrel8r/projects/3)** - Current work and priorities
- **[Discussions](https://github.com/korrel8r/korrel8r/discussions)** - Questions, ideas, and community support

## License

By contributing, you agree that your contributions will be licensed under the [Apache License 2.0](LICENSE).
