# qail

`qail` (pronounced *kyle*) is a CLI workspace manager for multi-repo projects.

Group git repos into a workspace, clone them together, and open the whole set in your editor or a tmux session. Run post-install scripts per repo or per workspace.

## Table of Contents

- [Why?](#why)
- [Install](#install)
- [Quick start](#quick-start)
- [Example: spinning up a microservice workspace](#example-spinning-up-a-microservice-workspace)
- [Commands](#commands)
  - [Config](#config)
  - [Repos](#repos)
  - [Workspaces](#workspaces)
  - [Scripts](#scripts)
  - [Tmux](#tmux)
- [Data location](#data-location)

## Why?

When working on microservices or any feature that touches multiple repos, you usually want them cloned side-by-side, opened together, and bootstrapped the same way each time. `qail` encapsulates that as a *workspace*.

## Install

Build and put the binary on your `PATH`:

```sh
make build
mkdir -p ~/.qail/bin
cp bin/qail ~/.qail/bin
```

Add to `.bashrc` / `.zshrc`:

```sh
export QAILPATH="$HOME/.qail/bin"
export PATH="$QAILPATH:$PATH"
```

## Quick start

```sh
qail init                              # set workspace root dir
qail config editor "/Applications/Visual Studio Code.app/Contents/Resources/app/bin/code"
qail repo add                          # register a git repo
qail workspace add                     # create a workspace from registered repos
qail open                              # open workspace in editor
```

## Example: spinning up a microservice workspace

Say you work on a payments product split across `payments-api`, `payments-worker`, and `payments-web`. You want one command to clone all three and run `npm install` in the web repo.

```sh
# 1. one-time setup
qail init
qail config editor "/Applications/Visual Studio Code.app/Contents/Resources/app/bin/code"

# 2. register each repo
qail r a   # → git@github.com:acme/payments-api.git
qail r a   # → git@github.com:acme/payments-worker.git
qail r a   # → git@github.com:acme/payments-web.git

# 3. create a post-install script for the web repo
qail scripts a    # name it: web-install
qail scripts o    # edit; replace body with: npm install

# 4. attach script to the web repo
qail repo p       # pick payments-web → web-install

# 5. create the workspace
qail workspace add
# pick name: payments
# select all 3 repos

# 6. open it
qail open         # pick payments → editor launches with all 3 repos
# or in tmux:
qail mux o        # pick payments → tmux session w/ a window per repo
```

Later, jump back in:

```sh
qail ws ls        # list workspaces
qail open         # reopen
qail ws cd        # print path; pair with shell alias to cd into it
```

Tear down a stale clone but keep the workspace definition:

```sh
qail ws clean
```

## Commands

Most commands have short aliases: `workspace` → `ws`, `repo` → `r`, `add` → `a`, `list` → `ls`, `remove` → `rm`, `open` → `o`, `create` → `c`.

### Config

```sh
qail init                  # set workspace root
qail config ls             # show current config
qail config editor <path>  # set editor binary
qail config root <path>    # set workspace root
```

### Repos

```sh
qail repo add              # register a git repo (alias: r a)
qail repo list             # list repos
qail repo remove           # unregister
qail repo p                # attach a post-install script
```

### Workspaces

```sh
qail workspace add         # create workspace (alias: ws a)
qail workspace list
qail workspace edit
qail workspace remove
qail workspace clone       # re-clone repos
qail workspace clean       # remove orphaned dirs
qail workspace explore     # browse workspace contents
qail workspace cd          # print path
qail open                  # open in editor
qail mux o                 # open in tmux
```

### Scripts

Bash scripts stored in `~/.qail/scripts/`, runnable as repo or workspace post-install hooks.

```sh
qail scripts add           # create (alias: s a)
qail scripts open          # edit
qail scripts list
qail scripts remove
```

Default template:

```sh
#!/bin/bash

# Add your custom logic here
ls -l
```

### Tmux

```sh
qail mux o                 # open workspace as tmux session
qail mux ls                # list sessions
qail mux rm                # kill session
```

Requires `tmux` installed.

## Data location

Everything lives under `~/.qail/`:

- `qail.db` — SQLite database (config, repos, workspaces)
- `scripts/` — user scripts
- `bin/` — the binary (if installed via the steps above)
