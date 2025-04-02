package app

import (
	"embed"

	"gorm.io/gorm"
)

//go:embed embed/*
var Embed embed.FS

var Flag *TsFlag = &TsFlag{}

var DB *gorm.DB
