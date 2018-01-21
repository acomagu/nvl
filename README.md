# nvl: NeoVim as less

nvl enables to use NeoVim as less command.

## Installation

```bash
$ go get -u github.com/acomagu/nvl
```

## Usage

```bash
$ nvl file
```

or,

```bash
$ cat file | nvl
```

If `NVIM_LISTEN_ADDRESS` environment variable is set, `nvl` use it to view in existing NeoVim instance. If not, starts new NeoVim instance.

## Attributes

- **Asyncronous.** Unlike `view` command, it can start showing before the input stream ends.
- **Fast like less.**

## Author

[acomagu](https://github.com/acomagu)
