package models

type AccessLevel string

const (
	AccessLevelRead  AccessLevel = "read"
	AccessLevelWrite AccessLevel = "write"
	AccessLevelAdmin AccessLevel = "admin"
)

type GPIOType string

const (
	GPIOTypeDigital GPIOType = "digital"
	GPIOTypeAnalog  GPIOType = "analog"
)
