#!/bin/sh

set -e

if ! command -v tar >/dev/null; then
	echo "Error: tar is required to install jsctl" 1>&2
	exit 1
fi

if ! command -v curl >/dev/null; then
    echo "Error: curl is required to install jsctl" 1>&2
    exit 1
fi

if [ "$OS" = "Windows_NT" ]; then
    if [ -z "$PROCESSOR_ARCHITEW6432" ]; then
        arch="$PROCESSOR_ARCHITECTURE"
    else
        arch="$PROCESSOR_ARCHITEW6432"
    fi

    case $arch in
    "AMD64") target="jsctl-windows-amd64" ;;
    "ARM64") target="jsctl-windows-arm64" ;;
    *)
        echo "Error: Unsupported windows achitecture: ${arch}" 1>&2
        exit 1 ;;
    esac
    target_file="jsctl.exe"
else
	case $(uname -sm) in
	"Darwin x86_64") target="jsctl-darwin-amd64" ;;
	"Darwin arm64")  target="jsctl-darwin-arm64" ;;
    "Linux x86_64")  target="jsctl-linux-amd64" ;;
	"Linux aarch64") target="jsctl-linux-arm64" ;;
    *)
        echo "Error: Unsupported operating system or architecture: $(uname -sm)" 1>&2
        exit 1 ;;
	esac
    target_file="jsctl"
fi

jsctl_uri="https://github.com/jetstack/jsctl/releases/latest/download/${target}.tar.gz"

jsctl_install="${JSCTL_INSTALL:-$HOME/.jsctl}"
bin_dir="$jsctl_install/bin"
bin="$bin_dir/$target_file"

if [ ! -d "$bin_dir" ]; then
	mkdir -p "$bin_dir"
fi

curl --fail --location --progress-bar --output "$bin.tar.gz" "$jsctl_uri"
tar xfO "$bin.tar.gz" "$target/$target_file" > "$bin"
chmod +x "$bin"
rm "$bin.tar.gz"

echo "jsctl was installed successfully to $bin"
if command -v jsctl >/dev/null; then
	echo "Run 'jsctl --help' to get started"
else
	case $SHELL in
	/bin/zsh) shell_profile=".zshrc" ;;
	*) shell_profile=".bashrc" ;;
	esac
    echo
	echo "Manually add the directory to your \$HOME/$shell_profile (or similar)"
	echo "  export JSCTL_INSTALL=\"$jsctl_install\""
	echo "  export PATH=\"\$JSCTL_INSTALL/bin:\$PATH\""
    echo
	echo "And run \"source $HOME/.bashrc\" to update your current shell"
    echo
	echo "Run '$bin --help' to get started"
fi
echo
echo "Checkout https://platform.jetstack.io/documentation for more information"
