---
name: kitflow
description: "Use when the user wants to manage a GitFlow-style workflow with the kitflow CLI: initializing config, starting/finishing/listing feature, release, or hotfix branches, publishing branches, tracking remote branches, or checking kitflow status. Trigger on: 'kitflow', 'gitflow', 'git flow', 'feature start', 'release finish', 'hotfix'."
compatibility: "Requires the kitflow CLI installed and a Git work tree. Install with `go install github.com/koheisg/kitflow@latest`."
---

# kitflow Skill

## When to use

Use this skill when the user wants to perform GitFlow-style branch operations with `kitflow`.

## Safety checks

Before commands that change branches, merge, tag, delete branches, or push:

```sh
git status --short
git branch --show-current
git log --oneline -5
```

If the working tree is dirty, stop and ask the user how to proceed.

## Common commands

Initialize kitflow in a repository:

```sh
kitflow init
```

Start and finish a feature:

```sh
kitflow feature start <name>
kitflow feature finish [name]
```

Start and finish a release:

```sh
kitflow release start <version>
kitflow release finish <version>
```

Start and finish a hotfix:

```sh
kitflow hotfix start <version>
kitflow hotfix finish <version>
```

List branches:

```sh
kitflow feature list
kitflow release list
kitflow hotfix list
```

Publish a branch and set upstream tracking:

```sh
kitflow feature publish [name]
kitflow release publish [version]
kitflow hotfix publish [version]
```

Track a remote branch locally:

```sh
kitflow feature track <name>
kitflow release track <version>
```

Show current kitflow config and branch:

```sh
kitflow status
```

## Options

Use `--dry-run` when the user wants a preview:

```sh
kitflow --dry-run feature start <name>
```

Use `--verbose` when debugging:

```sh
kitflow --verbose release finish <version>
```

## Workflow guidance

- `feature` and `release` branches start from `develop` by default.
- `hotfix` branches start from `main` by default.
- `start` accepts an optional base branch: `kitflow feature start <name> <base>`.
- `finish` merges with `--no-ff`, deletes the completed local branch, and release/hotfix flows create annotated tags.
- Remote operations use `origin` by default. Override with `git config --local kitflow.remote <remote>`.

## Verification

After a branch-changing command, verify state with:

```sh
git status --short
git branch --show-current
kitflow status
```

After publish/track commands, verify upstream with:

```sh
git rev-parse --abbrev-ref HEAD@{upstream}
```
