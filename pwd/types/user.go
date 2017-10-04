package types

type User struct {
	Id             string `json:"id" bson:"id"`
	Name           string `json:"name" bson:"name"`
	ProviderUserId string `json:"provider_user_id" bson:"provider_user_id"`
	Avatar         string `json:"avatar" bson:"avatar"`
	Provider       string `json:"provider" bson:"provider"`
	Email          string `json:"email" bson:"email"`
}

type LoginRequest struct {
	Id       string `json:"id" bson:"id"`
	Provider string `json:"provider" bson:"provider"`
}
