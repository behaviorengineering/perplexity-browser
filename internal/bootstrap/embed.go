package bootstrap

import _ "embed"

//go:embed templates/workflow.md.tpl
var workflowTemplate string

//go:embed templates/pack.md.tpl
var packTemplate string
