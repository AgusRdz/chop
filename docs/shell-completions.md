# Shell Completions

chop provides tab-completion scripts for bash, zsh, fish, and PowerShell.

## bash

Add to `~/.bashrc`:

```bash
source <(chop completion bash)
```

Or install permanently (no subprocess on each shell start):

**Linux:**

```bash
chop completion bash > ~/.local/share/bash-completion/completions/chop
```

**macOS (Homebrew):**

```bash
chop completion bash > $(brew --prefix)/etc/bash_completion.d/chop
```

> macOS ships with bash 3.2 which doesn't support `source <(...)`. Install a newer bash via Homebrew (`brew install bash`) and use the permanent install method above, or switch to zsh.

## zsh

Add to `~/.zshrc`:

```zsh
source <(chop completion zsh)
```

Or install to a user completions directory:

```zsh
mkdir -p ~/.zsh/completions
chop completion zsh > ~/.zsh/completions/_chop
# ensure this directory is in your fpath (add to ~/.zshrc):
fpath=(~/.zsh/completions $fpath)
```

## fish

Add to your fish config (`~/.config/fish/config.fish`):

```fish
chop completion fish | source
```

Or install permanently:

```fish
chop completion fish > ~/.config/fish/completions/chop.fish
```

## PowerShell

Add to your `$PROFILE`:

```powershell
chop completion powershell | Invoke-Expression
```

To find your profile path: `echo $PROFILE`
