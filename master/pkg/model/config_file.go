package model

// ConfigFile represents a row from the `config_files` table.
type ConfigFile struct {
	ID      int    `db:"id" json:"id"`
	Content []byte `db:"content"`
}
