package stat

type Dto struct {
	Key   string `json:"key" bson:"key"`
	Count int64  `json:"count" bson:"count"`
}
