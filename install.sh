#!/usr/bin/env sh
# clash-cli installer
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/boomyao/clash-cli/main/install.sh | sh
#
# Or with a specific version:
#   curl -fsSL https://raw.githubusercontent.com/boomyao/clash-cli/main/install.sh | sh -s -- v0.1.0

set -eu

REPO="boomyao/clash-cli"
MIHOMO_REPO="MetaCubeX/mihomo"
BIN_NAME="clashc"

# ---------- helpers ----------
red()    { printf "\033[31m%s\033[0m\n" "$*"; }
green()  { printf "\033[32m%s\033[0m\n" "$*"; }
yellow() { printf "\033[33m%s\033[0m\n" "$*"; }
blue()   { printf "\033[34m%s\033[0m\n" "$*"; }

info()  { blue   "▶ $*"; }
ok()    { green  "✓ $*"; }
warn()  { yellow "⚠ $*"; }
die()   { red    "✗ $*" >&2; exit 1; }

require() {
  command -v "$1" >/dev/null 2>&1 || die "$1 is required but not installed"
}

# ---------- detect platform ----------
detect_os() {
  case "$(uname -s)" in
    Darwin)  echo darwin ;;
    Linux)   echo linux ;;
    *)       die "unsupported OS: $(uname -s)" ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64)   echo amd64 ;;
    arm64|aarch64)  echo arm64 ;;
    *)              die "unsupported arch: $(uname -m)" ;;
  esac
}

# ---------- pick install dir ----------
pick_install_dir() {
  if [ "$(id -u)" = "0" ]; then
    echo "/usr/local/bin"
  else
    echo "$HOME/.local/bin"
  fi
}

# ---------- download a tarball/zip and extract a single binary ----------
fetch_binary() {
  url="$1"; out="$2"; binname="$3"
  tmpdir="$(mktemp -d)"
  trap 'rm -rf "$tmpdir"' EXIT INT TERM

  case "$url" in
    *.tar.gz|*.tgz)
      curl -fsSL "$url" | tar xz -C "$tmpdir"
      ;;
    *.zip)
      tmpzip="$tmpdir/dl.zip"
      curl -fsSL -o "$tmpzip" "$url"
      unzip -q "$tmpzip" -d "$tmpdir"
      ;;
    *)
      curl -fsSL -o "$out" "$url"
      chmod +x "$out"
      return 0
      ;;
  esac

  found="$(find "$tmpdir" -type f -name "$binname" -perm -u+x 2>/dev/null | head -n 1)"
  if [ -z "$found" ]; then
    found="$(find "$tmpdir" -type f -name "$binname" 2>/dev/null | head -n 1)"
  fi
  if [ -z "$found" ]; then
    die "could not find $binname inside downloaded archive"
  fi

  install -m 0755 "$found" "$out"
}

# ---------- get latest version tag from GitHub ----------
latest_tag() {
  repo="$1"
  curl -fsSL "https://api.github.com/repos/$repo/releases/latest" \
    | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' \
    | head -n 1
}

# ---------- install clashc ----------
install_clashc() {
  os="$1"; arch="$2"; dest_dir="$3"; version="$4"

  if [ -z "$version" ]; then
    info "Resolving latest clash-cli version..."
    version="$(latest_tag "$REPO")" || die "failed to query latest release"
    [ -z "$version" ] && die "no clash-cli releases found"
  fi
  ok "Using clash-cli $version"

  archive="clashc_${version#v}_${os}_${arch}.tar.gz"
  url="https://github.com/$REPO/releases/download/$version/$archive"

  info "Downloading $archive..."
  out="$dest_dir/$BIN_NAME"
  mkdir -p "$dest_dir"
  fetch_binary "$url" "$out" "$BIN_NAME"
  ok "Installed clashc → $out"
}

# ---------- install mihomo if missing ----------
# Echoes the absolute path of the mihomo binary on stdout so the caller
# can act on it (e.g. grant capabilities).
install_mihomo_if_missing() {
  os="$1"; arch="$2"; dest_dir="$3"

  if command -v mihomo >/dev/null 2>&1; then
    existing="$(command -v mihomo)"
    ok "mihomo already installed: $existing" >&2
    echo "$existing"
    return 0
  fi

  info "mihomo not found — installing latest from MetaCubeX..." >&2
  version="$(latest_tag "$MIHOMO_REPO")" || die "failed to query mihomo release"
  [ -z "$version" ] && die "no mihomo releases found"

  ver_no_v="${version#v}"
  archive="mihomo-${os}-${arch}-${version}.gz"
  url="https://github.com/$MIHOMO_REPO/releases/download/$version/$archive"

  tmpdir="$(mktemp -d)"
  out="$dest_dir/mihomo"
  mkdir -p "$dest_dir"

  info "Downloading $archive..." >&2
  if curl -fsSL -o "$tmpdir/mihomo.gz" "$url"; then
    gunzip -c "$tmpdir/mihomo.gz" > "$out"
    chmod +x "$out"
    ok "Installed mihomo $version → $out" >&2
    rm -rf "$tmpdir"
    echo "$out"
    return 0
  fi

  rm -rf "$tmpdir"
  warn "Could not download mihomo automatically (URL: $url)" >&2
  warn "Please install mihomo manually from https://github.com/$MIHOMO_REPO/releases" >&2
  return 0
}

# ---------- grant CAP_NET_ADMIN to mihomo so TUN mode works without root ----------
grant_mihomo_caps() {
  os="$1"; mihomo_path="$2"

  # Only meaningful on Linux. macOS TUN setup is a separate beast that
  # involves codesigning / system extensions; we don't touch it here.
  [ "$os" = "linux" ] || return 0
  [ -n "$mihomo_path" ] && [ -x "$mihomo_path" ] || return 0

  if ! command -v setcap >/dev/null 2>&1; then
    warn "'setcap' not found (install libcap2-bin) — TUN mode will need 'sudo clashc' or systemd"
    return 0
  fi

  # Already granted? Nothing to do.
  current="$(getcap "$mihomo_path" 2>/dev/null || true)"
  case "$current" in
    *cap_net_admin*) ok "mihomo already has cap_net_admin"; return 0 ;;
  esac

  info "Granting CAP_NET_ADMIN to $mihomo_path so TUN mode works without root..."

  # Already root? Just run setcap directly.
  if [ "$(id -u)" = "0" ]; then
    if setcap 'cap_net_admin,cap_net_bind_service=+ep' "$mihomo_path" 2>/dev/null; then
      ok "Granted CAP_NET_ADMIN"
      return 0
    fi
    warn "setcap failed; TUN mode will not work without 'sudo clashc'"
    return 0
  fi

  # Non-root: try sudo. Prompt explicitly so the user knows why their password is requested.
  if command -v sudo >/dev/null 2>&1; then
    echo "  → running: sudo setcap 'cap_net_admin,cap_net_bind_service=+ep' $mihomo_path"
    echo "  (you may be prompted for your password)"
    if sudo setcap 'cap_net_admin,cap_net_bind_service=+ep' "$mihomo_path"; then
      ok "Granted CAP_NET_ADMIN"
      return 0
    fi
    warn "sudo setcap failed or was declined."
    warn "TUN mode requires CAP_NET_ADMIN. To fix later, run:"
    warn "  sudo setcap 'cap_net_admin,cap_net_bind_service=+ep' $mihomo_path"
    return 0
  fi

  warn "sudo not available — please run as root:"
  warn "  setcap 'cap_net_admin,cap_net_bind_service=+ep' $mihomo_path"
}

# ---------- ensure dest_dir is on PATH ----------
ensure_path() {
  dir="$1"
  case ":$PATH:" in
    *":$dir:"*) return 0 ;;
  esac

  warn "$dir is not on your PATH"
  shell_name="$(basename "${SHELL:-sh}")"
  case "$shell_name" in
    bash) rc="$HOME/.bashrc" ;;
    zsh)  rc="$HOME/.zshrc" ;;
    fish) rc="$HOME/.config/fish/config.fish" ;;
    *)    rc="" ;;
  esac

  if [ -n "$rc" ]; then
    echo "  Add this line to $rc:"
    if [ "$shell_name" = "fish" ]; then
      echo "    set -gx PATH $dir \$PATH"
    else
      echo "    export PATH=\"$dir:\$PATH\""
    fi
  else
    echo "  Add $dir to your PATH manually."
  fi
}

# ---------- main ----------
main() {
  require curl
  require uname
  require tar

  os="$(detect_os)"
  arch="$(detect_arch)"
  dest_dir="$(pick_install_dir)"
  version="${1:-}"

  info "Platform: $os/$arch"
  info "Install dir: $dest_dir"
  echo

  install_clashc "$os" "$arch" "$dest_dir" "$version"
  mihomo_path="$(install_mihomo_if_missing "$os" "$arch" "$dest_dir")"
  echo
  grant_mihomo_caps "$os" "$mihomo_path"
  echo
  ensure_path "$dest_dir"
  echo
  ok "Done! Run: clashc"
  echo "  First run? Try: clashc 'https://your-subscription-url'"
}

main "$@"
