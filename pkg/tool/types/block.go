package types

type Block struct {
	Key     string `db:"key"`
	Content string `db:"content"`
}
