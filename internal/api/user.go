package api

// UserGroupResponse represents the nested group object in a user response.
type UserGroupResponse struct {
	ID        int64  `json:"id"`
	GID       int64  `json:"bsdgrp_gid"`
	GroupName string `json:"bsdgrp_group"`
}

// UserResponse represents a user from the TrueNAS API.
type UserResponse struct {
	ID                   int64             `json:"id"`
	UID                  int64             `json:"uid"`
	Username             string            `json:"username"`
	FullName             string            `json:"full_name"`
	Email                *string           `json:"email"`
	Home                 string            `json:"home"`
	Shell                string            `json:"shell"`
	HomeMode             string            `json:"home_mode"`
	Group                UserGroupResponse `json:"group"`
	Groups               []int64           `json:"groups"`
	SMB                  bool              `json:"smb"`
	PasswordDisabled     bool              `json:"password_disabled"`
	SSHPasswordEnabled   bool              `json:"ssh_password_enabled"`
	SSHPubKey            *string           `json:"sshpubkey"`
	Locked               bool              `json:"locked"`
	SudoCommands         []string          `json:"sudo_commands"`
	SudoCommandsNopasswd []string          `json:"sudo_commands_nopasswd"`
	Builtin              bool              `json:"builtin"`
	Local                bool              `json:"local"`
	Immutable            bool              `json:"immutable"`
}
