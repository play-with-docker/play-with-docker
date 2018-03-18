package types

type Client struct {
	Id        string   `json:"id" bson:"id"`
	SessionId string   `json:"session_id" bson:"session_id"`
	ViewPort  ViewPort `json:"viewport"`
}

type ViewPort struct {
	Rows uint `json:"rows"`
	Cols uint `json:"cols"`
}
