# Claude Code Custom Commands

This directory contains custom slash commands for Claude Code to assist with Korrel8r development.

## Available Commands

### `/generate-rule` - Interactive Rule Generator

Generate new Korrel8r correlation rules with interactive guidance.

**Usage:**
```
/generate-rule [optional-domain]
```

**What it does:**
- Guides you through creating correlation rules between observability domains
- Asks about source domain, target domain, and field mappings
- Generates valid YAML rule files with Go template queries
- Helps place rules in the appropriate file in `etc/korrel8r/rules/`

**When to use:**
- Adding new correlations between k8s resources, logs, metrics, alerts, traces, etc.
- Creating rules for custom domains
- Learning the rule syntax and structure

**Example:**
```
User: /generate-rule
Agent: I'll help you create a new correlation rule. What is the source domain? (k8s, log, metric, alert, trace, netflow, incident)
User: k8s
Agent: What source classes? (e.g., Pod, Deployment.apps) or leave blank for all k8s classes
...
```

**See also:**
- [AGENTS.md](../../AGENTS.md) - Full developer guide with detailed command documentation
- [User Guide - Configuration](https://korrel8r.github.io/korrel8r/#_configuration) - Rule syntax and examples

## Creating New Commands

To create a new custom command:

1. Create a markdown file in this directory: `command-name.md`
2. Add frontmatter with description and argument hint:
   ```yaml
   ---
   description: Brief description of the command
   argument-hint: [optional-args]
   ---
   ```
3. Write the command prompt with context, examples, and instructions
4. Test the command: `/command-name`

The command will be automatically available in Claude Code sessions.
