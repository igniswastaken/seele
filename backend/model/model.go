package model

type SetRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type GetRequest struct {
	Key string `json:"key"`
}

type SetResponse struct {
	Success bool `json:"success"`
}

type GetResponse struct {
	Value string `json:"value"`
}

type DeleteRequest struct {
	Key string `json:"key"`
}

type DeleteResponse struct {
	Success bool `json:"success"`
}
