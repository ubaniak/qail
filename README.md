# Introduction

A workspace management tool. 

`qail` pronounded `kyle`

## Why?

When dealing with micro services or services which require multiple repos, it is handy to create a workspace that encapsulates all related project repos.


## Getting started

* Build the project with

```sh
make build
mkdir ~/.qail/bin
cp bin/qail ~/.qail/bin
```

* Add the following to your `.bashrc` or `.zshrc`

```sh
export QAILPATH="~/.qail/bin"
export PATH=$QAILPATH:$PATH
```

## Usage

### Setup

* Set your workspace location with

```sh
qail init
```

* Add a text editor with

```sh
# This will set vscode on macos
qail config editor "/Applications/Visual Studio Code.app/Contents/Resources/app/bin/code"
```

* View your config settings with

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

### Workspace managemet


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

* Install tmux

Open a workspace with 


``` sh
qail mux o
```
