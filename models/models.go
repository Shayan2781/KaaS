package models

type Environment struct {
	Key      string `json:"Key"`
	Value    string `json:"Value"`
	IsSecret bool   `json:"IsSecret"`
}

type Resource struct {
	CPU string `json:"CPU"`
	RAM string `json:"RAM"`
}
