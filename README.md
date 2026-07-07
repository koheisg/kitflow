# kitflow

A small GitFlow-style workflow helper written in Go.

## Install

```sh
go install github.com/koheisg/kitflow@latest
```

Or build from source:

```sh
go build -o kitflow .
```

## Usage

```sh
kitflow init
kitflow feature start <name>
kitflow feature finish [name]
kitflow release start <version>
kitflow release finish <version>
kitflow hotfix start <version>
kitflow hotfix finish <version>
kitflow status
```

## Commands

- `kitflow init` stores local kitflow config and creates `develop` if possible.
- `kitflow feature start <name> [<base>]` creates `feature/<name>` from `develop` or `<base>`.
- `kitflow feature finish [name]` merges the feature branch into `develop` and deletes it.
- `kitflow feature publish [name]` pushes the feature branch and sets upstream tracking.
- `kitflow feature track <name>` creates a local feature branch tracking the remote branch.
- `kitflow feature list` lists the current feature branches.
- `kitflow release start <version> [<base>]` creates `release/<version>` from `develop` or `<base>`.
- `kitflow release finish <version>` merges the release into `main` and `develop`, tags it, and deletes the release branch.
- `kitflow release publish [version]` pushes the release branch and sets upstream tracking.
- `kitflow release track <version>` creates a local release branch tracking the remote branch.
- `kitflow release list` lists the current release branches.
- `kitflow hotfix start <version> [<base>]` creates `hotfix/<version>` from `main` or `<base>`.
- `kitflow hotfix finish <version>` merges the hotfix into `main` and `develop`, tags it, and deletes the hotfix branch.
- `kitflow hotfix publish [version]` pushes the hotfix branch and sets upstream tracking.
- `kitflow hotfix list` lists the current hotfix branches.
- `kitflow status` prints the current kitflow config and branch.

## Options

- `--verbose` prints git commands before running them.
- `--dry-run` prints write operations without running them.

## Configuration

`kitflow init` writes local Git config values:

```sh
git config --local kitflow.branch.main main
git config --local kitflow.branch.develop develop
git config --local kitflow.prefix.feature feature/
git config --local kitflow.prefix.release release/
git config --local kitflow.prefix.hotfix hotfix/
```

Remote commands use `origin` by default. Override it with:

```sh
git config --local kitflow.remote upstream
```

## AI agent skill

This repository includes a reusable skill for AI coding agents:

```text
skills/kitflow/SKILL.md
```

Use it when an agent needs to operate GitFlow-style workflows with `kitflow`.
