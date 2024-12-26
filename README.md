- [Introduction](#introduction)
  - [Why?](#why)
  - [Getting started](#getting-started)
  - [Usage](#usage)
    - [Setup](#setup)
    - [Repository management](#repository-management)
    - [Workspace management](#workspace-management)
    - [Open a workspace](#open-a-workspace)
    - [Tmux](#tmux)
    - [Scripts](#scripts)
      - [Create a script](#create-a-script)
      - [Adding a Repo post install script](#adding-a-repo-post-install-script)

# Introduction

`qail` pronounded `kyle`

Manage your workspace in style with qail

## Why?

When dealing with micro services or services which require multiple repos, it is handy to create a workspace that encapsulates all related project repos.

## Getting started

- Build the project with

```sh
make build
mkdir ~/.qail/bin
cp bin/qail ~/.qail/bin
```

- Add the following to your `.bashrc` or `.zshrc`

```sh
export QAILPATH="~/.qail/bin"
export PATH=$QAILPATH:$PATH
```

## Usage

### Setup

- Set your workspace location with

```sh
qail init
```

- Add a text editor with

```sh
# This will set vscode on macos
qail config editor "/Applications/Visual Studio Code.app/Contents/Resources/app/bin/code"
```

- View your config settings with

```sh
qail config ls
```

### Repository management

Before you create a workspace add a git repo to your qail config with

```sh
qail repo add
# short

qail r a
```

follow the steps to include your git repo

you can manage your repos with the `repo` command.

```sh
qail repo --help
```

### Workspace management

Create a workspace with

```sh
qail workspace add

# short

qail ws a
```

you can manage your repos with the `workspace` command.

```sh
qail workspace --help
```

### Open a workspace

Open a workspace with

```sh
qail open
```

### Tmux

- Install tmux

Open a workspace with

```sh
qail mux o
```

### Scripts

Run a bash script before or after a workspace is created or a repo is installed.

#### Create a script

```sh
qail scripts a
```

Edit your script

```sh
qail scripts o
```

The default script will be

```sh
#!/bin/bash

# Add your custom logic here
ls -l
```

#### Adding a Repo post install script

```sh
qail repo p
```
