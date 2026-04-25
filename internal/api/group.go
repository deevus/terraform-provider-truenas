package api

// GroupResponse represents a group from the TrueNAS API.
type GroupResponse struct {
	ID                   int64    `json:"id"`
	GID                  int64    `json:"gid"`
	Name                 string   `json:"name"`
	Builtin              bool     `json:"builtin"`
	SMB                  bool     `json:"smb"`
	SudoCommands         []string `json:"sudo_commands"`
	SudoCommandsNopasswd []string `json:"sudo_commands_nopasswd"`
	Users                []int64  `json:"users"`
	Local                bool     `json:"local"`
	Immutable            bool     `json:"immutable"`
}
