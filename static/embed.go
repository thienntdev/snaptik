package static

import "embed"

//go:embed *
//go:embed css/*
//go:embed js/*
var FS embed.FS
