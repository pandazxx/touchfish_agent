package ai

import (
	"embed"
	"fmt"
	"strings"
)

//go:embed templates/*.md
var templateFS embed.FS

func loadTemplate(name string) (string, error) {
	data, err := templateFS.ReadFile("templates/" + name)
	if err != nil {
		return "", fmt.Errorf("loading template %q: %w", name, err)
	}
	return string(data), nil
}

func render(tmpl string, vars map[string]string) string {
	result := tmpl
	for k, v := range vars {
		result = strings.ReplaceAll(result, "{"+k+"}", v)
	}
	return result
}

// SEIssueFixParams holds the variables for the SE issue fix template.
type SEIssueFixParams struct {
	ProjectSetup string
	IssueNumber  int
	IssueTitle   string
	IssueBody    string
	Comments     []CommentParam
}

type CommentParam struct {
	Author string
	Date   string
	Body   string
}

// RenderSEIssueFix renders the se_issue_fix.md template.
func RenderSEIssueFix(p SEIssueFixParams) (string, error) {
	tmpl, err := loadTemplate("se_issue_fix.md")
	if err != nil {
		return "", err
	}

	var commentBlock string
	for _, c := range p.Comments {
		commentBlock += fmt.Sprintf("**%s** (%s):\n%s\n\n", c.Author, c.Date, c.Body)
	}

	// The template has {for each comment:}...{end for} pseudo-syntax.
	// Replace the loop markers and content with rendered comments.
	tmpl = replaceLoop(tmpl, commentBlock)

	return render(tmpl, map[string]string{
		"PROJECT_SETUP.md content": p.ProjectSetup,
		"issue.number":             fmt.Sprintf("%d", p.IssueNumber),
		"issue.title":              p.IssueTitle,
		"issue.body":               p.IssueBody,
	}), nil
}

// SEImplementationParams holds the variables for the SE implementation template.
type SEImplementationParams struct {
	ProjectSetup     string
	Requirement      string
	TestRequirements string
	ReqDiff          string
	TestReqDiff      string
	ReqChanged       bool
	TestReqChanged   bool
}

// RenderSEImplementation renders the se_implementation.md template.
func RenderSEImplementation(p SEImplementationParams) (string, error) {
	tmpl, err := loadTemplate("se_implementation.md")
	if err != nil {
		return "", err
	}

	testReq := p.TestRequirements
	if testReq == "" {
		testReq = "No test requirements yet."
	}

	// Handle conditional sections.
	if p.ReqChanged {
		tmpl = strings.ReplaceAll(tmpl, "{if REQUIREMENT.md changed:}", "")
		tmpl = strings.ReplaceAll(tmpl, "{end if}", "")
	} else {
		tmpl = removeConditional(tmpl, "{if REQUIREMENT.md changed:}")
	}
	if p.TestReqChanged {
		tmpl = strings.ReplaceAll(tmpl, "{if TEST_REQUIREMENTS.md changed:}", "")
	} else {
		tmpl = removeConditional(tmpl, "{if TEST_REQUIREMENTS.md changed:}")
	}

	return render(tmpl, map[string]string{
		"PROJECT_SETUP.md content":                                      p.ProjectSetup,
		"REQUIREMENT.md content":                                        p.Requirement,
		"TEST_REQUIREMENTS.md content, or \"No test requirements yet.\"": testReq,
		"diff of REQUIREMENT.md from last processed version to current": p.ReqDiff,
		"diff of TEST_REQUIREMENTS.md from last processed version to current": p.TestReqDiff,
	}), nil
}

// SETestFixParams holds the variables for the SE test fix template.
type SETestFixParams struct {
	TestStdout string
	TestStderr string
}

// RenderSETestFix renders the se_test_fix.md template.
func RenderSETestFix(p SETestFixParams) (string, error) {
	tmpl, err := loadTemplate("se_test_fix.md")
	if err != nil {
		return "", err
	}
	return render(tmpl, map[string]string{
		"test stdout": p.TestStdout,
		"test stderr": p.TestStderr,
	}), nil
}

// QATestRequirementsParams holds the variables for the QA test requirements template.
type QATestRequirementsParams struct {
	Requirement     string
	ProjectSetup    string
	ReqDiff         string
	ExistingContent string
	HasExisting     bool
}

// RenderQATestRequirements renders the qa_test_requirements.md template.
func RenderQATestRequirements(p QATestRequirementsParams) (string, error) {
	tmpl, err := loadTemplate("qa_test_requirements.md")
	if err != nil {
		return "", err
	}

	tmpl = resolveIfElse(tmpl, "{if TEST_REQUIREMENTS.md exists:}", "{else:}", "{end if}", p.HasExisting)

	return render(tmpl, map[string]string{
		"REQUIREMENT.md content":                                        p.Requirement,
		"PROJECT_SETUP.md content":                                      p.ProjectSetup,
		"diff of REQUIREMENT.md from last processed version to current": p.ReqDiff,
		"existing content": p.ExistingContent,
	}), nil
}

// QAReviewParams holds the variables for the QA review template.
type QAReviewParams struct {
	TestRequirements string
	Requirement      string
	ProjectSetup     string
	FromHash         string
	ToHash           string
	ChangedFiles     []ChangedFileParam
}

type ChangedFileParam struct {
	Path   string
	Change string // "Added", "Modified", "Deleted"
}

// RenderQAReview renders the qa_review.md template.
func RenderQAReview(p QAReviewParams) (string, error) {
	tmpl, err := loadTemplate("qa_review.md")
	if err != nil {
		return "", err
	}

	var fileBlock string
	for _, f := range p.ChangedFiles {
		fileBlock += fmt.Sprintf("- %s (%s)\n", f.Path, f.Change)
	}

	tmpl = replaceLoop(tmpl, fileBlock)

	return render(tmpl, map[string]string{
		"TEST_REQUIREMENTS.md content": p.TestRequirements,
		"REQUIREMENT.md content":       p.Requirement,
		"PROJECT_SETUP.md content":     p.ProjectSetup,
		"fromHash":                     p.FromHash,
		"toHash":                       p.ToHash,
	}), nil
}

// replaceLoop removes {for each ...} and {end for} markers and inserts rendered content.
func replaceLoop(tmpl, rendered string) string {
	// Find and remove the loop markers.
	forStart := strings.Index(tmpl, "{for each")
	if forStart == -1 {
		return tmpl
	}
	forEnd := strings.Index(tmpl[forStart:], "}")
	if forEnd == -1 {
		return tmpl
	}
	endFor := strings.Index(tmpl, "{end for}")
	if endFor == -1 {
		return tmpl
	}

	return tmpl[:forStart] + rendered + tmpl[endFor+len("{end for}"):]
}

// removeConditional removes content between a conditional marker and the next {end if}.
func removeConditional(tmpl, marker string) string {
	start := strings.Index(tmpl, marker)
	if start == -1 {
		return tmpl
	}
	end := strings.Index(tmpl[start:], "{end if}")
	if end == -1 {
		return tmpl
	}
	return tmpl[:start] + tmpl[start+end+len("{end if}"):]
}

// removeSection removes content between start and end markers (inclusive).
func removeSection(tmpl, startMarker, endMarker string) string {
	start := strings.Index(tmpl, startMarker)
	if start == -1 {
		return tmpl
	}
	end := strings.Index(tmpl[start:], endMarker)
	if end == -1 {
		return tmpl
	}
	return tmpl[:start] + tmpl[start+end+len(endMarker):]
}

// resolveIfElse handles {if ...}{else:}{end if} blocks.
// If condition is true, keeps the if-branch; otherwise keeps the else-branch.
func resolveIfElse(tmpl, ifMarker, elseMarker, endMarker string, condition bool) string {
	ifStart := strings.Index(tmpl, ifMarker)
	if ifStart == -1 {
		return tmpl
	}
	elseStart := strings.Index(tmpl[ifStart:], elseMarker)
	endStart := strings.Index(tmpl[ifStart:], endMarker)
	if endStart == -1 {
		return tmpl
	}

	before := tmpl[:ifStart]
	after := tmpl[ifStart+endStart+len(endMarker):]

	if condition {
		ifContent := ""
		if elseStart != -1 {
			ifContent = tmpl[ifStart+len(ifMarker) : ifStart+elseStart]
		} else {
			ifContent = tmpl[ifStart+len(ifMarker) : ifStart+endStart]
		}
		return before + ifContent + after
	}
	// else branch
	if elseStart != -1 {
		elseContent := tmpl[ifStart+elseStart+len(elseMarker) : ifStart+endStart]
		return before + elseContent + after
	}
	return before + after
}
