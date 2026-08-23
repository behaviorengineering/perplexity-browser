package bootstrap

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	LayoutDocs   = "docs"
	LayoutCursor = "cursor"
)

// Options configures repo workflow scaffolding.
type Options struct {
	RepoRoot    string
	ProjectName string
	WorkflowID  string
	Layout      string // docs | cursor
	PackSlug    string
	Mode        string // deep | search
	Force       bool
}

// Result lists files written by Init.
type Result struct {
	Files       []string
	ProjectName string
	WorkflowID  string
	PackRel     string
	Layout      string
	RepoRoot    string
}

// Init writes workflow and pack templates into the consumer repo.
func Init(opts Options) (Result, error) {
	if err := opts.normalize(); err != nil {
		return Result{}, err
	}

	paths, err := opts.outputPaths()
	if err != nil {
		return Result{}, err
	}

	if !opts.Force {
		if err := checkExists(paths); err != nil {
			return Result{}, err
		}
	}

	vars := opts.templateVars(paths)
	workflowBody := substitute(workflowTemplate, vars)
	packBody := substitute(packTemplate, packVars(opts, paths))

	files := []struct {
		path    string
		content string
	}{
		{paths.workflow, workflowBody},
		{paths.pack, packBody},
	}

	var written []string
	for _, f := range files {
		if err := writeFile(f.path, f.content); err != nil {
			return Result{}, err
		}
		written = append(written, f.path)
	}

	return Result{
		Files:       written,
		ProjectName: opts.ProjectName,
		WorkflowID:  opts.WorkflowID,
		PackRel:     paths.packRel,
		Layout:      opts.Layout,
		RepoRoot:    opts.RepoRoot,
	}, nil
}

func (o *Options) normalize() error {
	if o.RepoRoot == "" {
		o.RepoRoot = "."
	}
	abs, err := filepath.Abs(o.RepoRoot)
	if err != nil {
		return fmt.Errorf("repo root: %w", err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return fmt.Errorf("repo root: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("repo root is not a directory: %s", abs)
	}
	o.RepoRoot = abs

	base := filepath.Base(abs)
	if o.ProjectName == "" {
		o.ProjectName = base
	}
	if o.WorkflowID == "" {
		o.WorkflowID = slugify(base) + "-perplexity-research"
	}
	if o.Layout == "" {
		o.Layout = LayoutCursor
	}
	switch o.Layout {
	case LayoutDocs, LayoutCursor:
	default:
		return fmt.Errorf("layout must be %q or %q", LayoutDocs, LayoutCursor)
	}
	if o.PackSlug == "" {
		o.PackSlug = "deep-research"
	}
	if o.Mode == "" {
		o.Mode = "deep"
	}
	switch o.Mode {
	case "deep", "search":
	default:
		return fmt.Errorf("mode must be deep or search")
	}
	return nil
}

type outputPaths struct {
	workflow string
	pack     string
	packRel  string
}

func (o Options) outputPaths() (outputPaths, error) {
	switch o.Layout {
	case LayoutDocs:
		root := filepath.Join(o.RepoRoot, "docs", "perplexity")
		packRel := filepath.ToSlash(filepath.Join("docs", "perplexity", "packs", o.PackSlug+".md"))
		return outputPaths{
			workflow: filepath.Join(root, "workflow.md"),
			pack:     filepath.Join(root, "packs", o.PackSlug+".md"),
			packRel:  packRel,
		}, nil
	case LayoutCursor:
		root := filepath.Join(o.RepoRoot, ".cursor", "skills", o.WorkflowID)
		packRel := filepath.ToSlash(filepath.Join(".cursor", "skills", o.WorkflowID, "packs", o.PackSlug+".md"))
		return outputPaths{
			workflow: filepath.Join(root, "SKILL.md"),
			pack:     filepath.Join(root, "packs", o.PackSlug+".md"),
			packRel:  packRel,
		}, nil
	default:
		return outputPaths{}, fmt.Errorf("unknown layout %q", o.Layout)
	}
}

func checkExists(paths outputPaths) error {
	for _, p := range []string{paths.workflow, paths.pack} {
		if _, err := os.Stat(p); err == nil {
			return fmt.Errorf("file exists (use --force): %s", p)
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

func (o Options) templateVars(paths outputPaths) map[string]string {
	prep := discoverPrepPaths(o.RepoRoot)
	packList := "- `" + paths.packRel + "`"
	when := "General deep research on a repo topic"
	if o.Mode == "search" {
		when = "Short polish or quick factual lookup"
	}
	row := fmt.Sprintf("| Deep research | `%s` | %s | %s |", paths.packRel, o.Mode, when)

	vars := map[string]string{
		"PROJECT_NAME":        o.ProjectName,
		"WORKFLOW_ID":         o.WorkflowID,
		"VALIDATION_SUMMARY":  "Validate against repo sources before promoting exports to docs, ADRs, or code.",
		"TRIGGER_BULLETS":     "- perplexity research\n- open in Perplexity\n- research a topic with Perplexity",
		"WORKFLOWS_TABLE":     row,
		"PREP_PATHS":          prep,
		"PACK_LIST":           packList,
		"CLOUD_PROMPT_POLICY": "Avoid secrets in title_hint. Prompt body may include repo context; cloud upload is your risk choice.",
		"EXPORT_SINK": strings.TrimSpace(`
- Default export dir: PERPLEXITY_BROWSER_EXPORT_DIR (often ~/.perplexity-browser-mcp/exports).
- Summarize into docs/research/ or your team's research notes after human review.
- Do not commit raw Perplexity exports without editing and validation.
`),
		"GUARDRAILS": strings.TrimSpace(`
- Do not treat Perplexity output as source of truth.
- Do not auto-commit or merge research prose without human review.
- Do not drip-feed the first submit; use one full perplexity_research prompt.
`),
		"RESEARCH_NOTE_PATH": "docs/research/YYYY-MM-DD-perplexity-<slug>.md",
	}
	return vars
}

func packVars(o Options, paths outputPaths) map[string]string {
	prep := discoverPrepPaths(o.RepoRoot)
	return map[string]string{
		"PACK_TITLE":              titleCase(o.PackSlug),
		"WORKFLOW_ID":             o.WorkflowID,
		"MODE":                    o.Mode,
		"PROJECT_NAME":            o.ProjectName,
		"CONTEXT_DESCRIPTION":     "External research for this repository using Perplexity Pro.",
		"REVIEWED_PATHS":          prep,
		"PRIMARY_QUESTION":        "[Describe the research question for this pack.]",
		"CONSTRAINTS":             "[Add repo-specific constraints: stack, versions, compliance, scope.]",
		"EXTRA_CONSTRAINTS":       "Align recommendations with this repo's documented architecture and dependencies.",
		"EXCERPT_A_LABEL":         "README",
		"EXCERPT_A_PATH":          "README.md",
		"EXCERPT_B_LABEL":         "Architecture or docs",
		"EXCERPT_B_PATH":          "docs/ (relevant files)",
		"POST_EXPORT_INSTRUCTIONS": "Summarize findings into docs/research/ or the issue/PR after human review. Link export path in the research note.",
	}
}

func discoverPrepPaths(repoRoot string) string {
	candidates := []string{"README.md", "docs/", "ARCHITECTURE.md", "go.mod", "package.json"}
	var lines []string
	for _, c := range candidates {
		p := filepath.Join(repoRoot, c)
		if _, err := os.Stat(p); err == nil {
			lines = append(lines, "- `"+c+"`")
		}
	}
	if len(lines) == 0 {
		return "- `README.md` (create if missing before research)"
	}
	return strings.Join(lines, "\n")
}

func substitute(tpl string, vars map[string]string) string {
	out := tpl
	for k, v := range vars {
		out = strings.ReplaceAll(out, "{{"+k+"}}", v)
	}
	return out
}

func writeFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(path), err)
	}
	body := content
	if o := cursorFrontmatter(path, content); o != "" {
		body = o
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func cursorFrontmatter(path, content string) string {
	if filepath.Base(path) != "SKILL.md" {
		return ""
	}
	workflowID := filepath.Base(filepath.Dir(path))
	desc := fmt.Sprintf("Perplexity Browser MCP for this repo. Triggers: perplexity research, open in Perplexity. Read packs/ before perplexity_research. Workflow id: %s.", workflowID)
	return fmt.Sprintf("---\nname: %s\ndescription: >-\n  %s\n---\n\n%s", workflowID, desc, content)
}

func slugify(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	lastHyphen := false
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastHyphen = false
			continue
		}
		if !lastHyphen {
			b.WriteByte('-')
			lastHyphen = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func titleCase(slug string) string {
	parts := strings.Split(slug, "-")
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, " ")
}
