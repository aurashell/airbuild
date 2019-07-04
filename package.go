package main

// Package - a package?
type Package struct {
	Name           string
	Wants          []string
	Source         map[string]string
	Tool           string
	Where          string
	SourceDir      string
	BuildDir       string
	PrecfgCommands []string
	GetSteps       []Step
	BuildSteps     []Step
	RebuildSteps   []Step
	NoTouch        bool
	ConfigureFlags string
	InstallCopy    map[string]string
}
