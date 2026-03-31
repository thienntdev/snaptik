package templates

import "embed"

//go:embed *
//go:embed layouts/*
//go:embed seo/*
var FS embed.FS
