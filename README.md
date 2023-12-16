<div align="center">
  <!-- <h1>🐙 Pullpo CLI 🐙</h1> -->
  <div align="center">
  <a href="https://pullpo.io">
    <img src="./readme/banner.png" />
  </a>
</div>
  <h3>Control Pullpo, GitHub and GitLab (soon) from the terminal. </h3>
</div>
<br>
<p align="center">
    <a href="https://pullpo.io"><img src="https://img.shields.io/badge/Pullpo-CLI-green.svg?style=flat-square" alt="codely.tv"/></a>
   <a href="https://github.com/pullpo-io/cli/releases"><img src="https://img.shields.io/github/v/release/pullpo-io/cli"></a>
    
    
</p>
<p align="center">
Pullpo CLI is a wrapper made in top of the GitHub and GitLab CLI, so that you can also control Pullpo from the terminal.

  <img src="./readme/demo.gif" alt="demo" />
</p>

## 🚀 Installation

### 💻 1. Install the CLI

#### For macOS and Linux

`pullpo` is available via [Homebrew][] and as a downloadable binary from the [releases page][].

```bash
brew install pullpo-io/cli/pullpo
```

#### Windows

`pullpo` is available via downloadable MSI on our [releases page][]

#### Other platforms

Download packaged binaries from the [releases page][].

[Homebrew]: https://brew.sh
[releases page]: https://github.com/pullpo-io/cli/releases/latest

### 🐙 2. Install [Pullpo](https://pullpo.io)

In order to have the Pullpo functionality available in the CLI, you'll need to install Pullpo in your GitHub/Gitlab and Slack workspace.

```
📌 Pullpo can only be installed in GitHub/GitLab orgs, not on personal accounts
```

**Go to [Pullpo.io](https://pullpo.io/app)** and follow the instructions to install Pullpo on GitHub/GitLab and Slack.

<p align="center">
  <img src="./readme/install-pullpo.gif" alt="install Pullpo" />
</p>

You can check the [GitHub](https://docs.pullpo.io/github-permissions) and [Slack](https://docs.pullpo.io/slack-permissions) permissions we ask for along with their reasons on our [docs page](https://docs.pullpo.io/).

## 🚶‍♂️ Getting started

First, you need to log in to your GitHub account using the CLI.

```bash
pullpo auth login
```

You can get started by creating a PR and an associated Slack channel with:
To get a full list and explanation of all the commands run:

```bash
man pullpo
```

Or to get info about a specific command:

```bash
man pullpo <command>
```

## 🤝 Contributing

If you want to implement a new feature, please, open an issue first.

## 💌 Contact us!

If you want to reach out, give feedback... email us at marco@pullpo.io

Thanks.
Pullpo team.
