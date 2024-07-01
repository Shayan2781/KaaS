package models

type Environment struct {
	Key      string
	Value    string
	IsSecret bool
}

type Resource struct {
	CPU string
	RAM string
}
