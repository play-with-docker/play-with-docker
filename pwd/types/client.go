package types

type Client struct {
	Id       string
	ViewPort ViewPort
	Session  *Session
}

type ViewPort struct {
	Rows uint
	Cols uint
}
