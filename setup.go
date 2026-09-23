package main

const setupZsh = `ohk() { OHK_SESSION=$$ command ohk "$@"; }

ohke() {
  local _ohk_name=${1:?usage: ohke VARNAME}
  local _ohk_dir=${XDG_STATE_HOME:-$HOME/.local/state}/ohk
  local _ohk_file=$_ohk_dir/$$.stdout
  if [ ! -f "$_ohk_file" ]; then
    printf '%s\n' "ohke: no ohk result for this shell (run ohk first)" >&2
    return 1
  fi
  local _ohk_val
  _ohk_val=$(command cat -- "$_ohk_file")
  export "$_ohk_name=$_ohk_val"
  command rm -f -- "$_ohk_file"
}
`

const setupBash = `ohk() { OHK_SESSION=$$ command ohk "$@"; }

ohke() {
  local _ohk_name=${1:?usage: ohke VARNAME}
  local _ohk_dir=${XDG_STATE_HOME:-$HOME/.local/state}/ohk
  local _ohk_file=$_ohk_dir/$$.stdout
  if [ ! -f "$_ohk_file" ]; then
    printf '%s\n' "ohke: no ohk result for this shell (run ohk first)" >&2
    return 1
  fi
  local _ohk_val
  _ohk_val=$(command cat -- "$_ohk_file")
  export "$_ohk_name=$_ohk_val"
  command rm -f -- "$_ohk_file"
}
`

const setupSh = `ohk() { OHK_SESSION=$$ command ohk "$@"; }

ohke() {
  _ohk_name=${1:?usage: ohke VARNAME}
  _ohk_dir=${XDG_STATE_HOME:-$HOME/.local/state}/ohk
  _ohk_file=$_ohk_dir/$$.stdout
  if [ ! -f "$_ohk_file" ]; then
    printf '%s\n' "ohke: no ohk result for this shell (run ohk first)" >&2
    return 1
  fi
  _ohk_val=$(command cat -- "$_ohk_file")
  export "$_ohk_name=$_ohk_val"
  command rm -f -- "$_ohk_file"
}
`

func setupSnippet(shell string) (string, bool) {
	switch shell {
	case "zsh":
		return setupZsh, true
	case "bash":
		return setupBash, true
	case "sh":
		return setupSh, true
	}
	return "", false
}
