package domain

type Test struct {
	ID     int64  `db:"id"`
	Filed1 string `db:"filed1"`
	Filed2 int    `db:"filed2"`
}
