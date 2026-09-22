# Source this file from your shell rc to export ohk results into variables.
#
#   source /path/to/ohk.sh
#   some-command | ohk          # interact, press ENTER
#   ohke FOO                    # exports FOO with the last result
#
# ohk only writes a session result when OHK_SESSION is set, which the `ohk`
# wrapper below does for this shell. `ohke` reads it back and removes it.

ohk() { OHK_SESSION=$$ command ohk "$@"; }

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