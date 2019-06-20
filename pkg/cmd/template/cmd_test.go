package template_test

import (
	"strings"
	"testing"

	cmdcore "github.com/k14s/ytt/pkg/cmd/core"
	cmdtpl "github.com/k14s/ytt/pkg/cmd/template"
	"github.com/k14s/ytt/pkg/files"
)

func TestLoad(t *testing.T) {
	yamlTplData := []byte(`
#@ load("@ytt:data", "data")
#@ load("funcs/funcs.lib.yml", "yamlfunc")
#@ load("funcs/funcs.lib.txt", "textfunc")
#@ load("funcs/funcs.star", "starfunc")
yamlfunc: #@ yamlfunc()
textfunc: #@ textfunc()
starfunc: #@ starfunc()
listdata: #@ data.list()
loaddata: #@ data.read("funcs/funcs.star")`)

	expectedYAMLTplData := `yamlfunc:
  yamlfunc: yamlfunc
textfunc: textfunc
starfunc:
- 1
- 2
listdata:
- tpl.yml
- funcs/funcs.lib.yml
- funcs/funcs.lib.txt
- funcs/funcs.star
loaddata: |2-

  def starfunc():
    return [1,2]
  end
`

	yamlFuncsData := []byte(`
#@ def/end yamlfunc():
yamlfunc: yamlfunc`)

	starlarkFuncsData := []byte(`
def starfunc():
  return [1,2]
end`)

	txtFuncsData := []byte(`(@ def textfunc(): @)textfunc(@ end @)`)

	filesToProcess := []*files.File{
		files.MustNewFileFromSource(files.NewBytesSource("tpl.yml", yamlTplData)),
		files.MustNewFileFromSource(files.NewBytesSource("funcs/funcs.lib.yml", yamlFuncsData)),
		files.MustNewFileFromSource(files.NewBytesSource("funcs/funcs.lib.txt", txtFuncsData)),
		files.MustNewFileFromSource(files.NewBytesSource("funcs/funcs.star", starlarkFuncsData)),
	}

	ui := cmdcore.NewPlainUI(false)
	opts := cmdtpl.NewOptions()

	out := opts.RunWithFiles(cmdtpl.TemplateInput{Files: filesToProcess}, ui)
	if out.Err != nil {
		t.Fatalf("Expected RunWithFiles to succeed, but was error: %s", out.Err)
	}

	if len(out.Files) != 1 {
		t.Fatalf("Expected number of output files to be 1, but was %d", len(out.Files))
	}

	file := out.Files[0]

	if file.RelativePath() != "tpl.yml" {
		t.Fatalf("Expected output file to be tpl.yml, but was %#v", file.RelativePath())
	}

	if string(file.Bytes()) != expectedYAMLTplData {
		t.Fatalf("Expected output file to have specific data, but was: >>>%s<<<", file.Bytes())
	}
}

func TestBacktraceAcrossFiles(t *testing.T) {
	yamlTplData := []byte(`
#@ load("funcs/funcs.lib.yml", "some_data")
#! line
#! other line
#! another line
#@ def another_data():
#@   return some_data()
#@ end
simple_key: #@ another_data()
`)

	yamlFuncsData := []byte(`
#@ def some_data():
#@   return 1+"2"
#@ end
`)

	expectedErr := `
- unknown binary op: int + string
    funcs/funcs.lib.yml:3 in some_data
     L #@   return 1+"2"
    tpl.yml:7 in another_data
     L #@   return some_data()
    tpl.yml:9 in <toplevel>
     L simple_key: #@ another_data()`

	filesToProcess := []*files.File{
		files.MustNewFileFromSource(files.NewBytesSource("tpl.yml", yamlTplData)),
		files.MustNewFileFromSource(files.NewBytesSource("funcs/funcs.lib.yml", yamlFuncsData)),
	}

	ui := cmdcore.NewPlainUI(false)
	opts := cmdtpl.NewOptions()

	out := opts.RunWithFiles(cmdtpl.TemplateInput{Files: filesToProcess}, ui)
	if out.Err == nil {
		t.Fatalf("Expected RunWithFiles fail")
	}

	if out.Err.Error() != expectedErr {
		t.Fatalf("Expected err, but was: >>>%s<<<", out.Err.Error())
	}
}

func TestDisallowDirectLibraryLoading(t *testing.T) {
	yamlTplData := []byte(`#@ load("_ytt_lib/data.lib.star", "data")`)

	expectedErr := `
- cannot load _ytt_lib/data.lib.star: Could not load file '_ytt_lib/data.lib.star' because it's contained in private library '' (use load("@lib:file", "symbol") where 'lib' is library name under _ytt_lib, for example, 'github.com/k14s/test')
    tpl.yml:1 in <toplevel>
     L #@ load("_ytt_lib/data.lib.star", "data")`

	filesToProcess := []*files.File{
		files.MustNewFileFromSource(files.NewBytesSource("tpl.yml", yamlTplData)),
		files.MustNewFileFromSource(files.NewBytesSource("_ytt_lib/data.lib.star", []byte("data = 3"))),
	}

	ui := cmdcore.NewPlainUI(false)
	opts := cmdtpl.NewOptions()

	out := opts.RunWithFiles(cmdtpl.TemplateInput{Files: filesToProcess}, ui)
	if out.Err == nil {
		t.Fatalf("Expected RunWithFiles fail")
	}

	if out.Err.Error() != expectedErr {
		t.Fatalf("Expected err, but was: >>>%s<<<", out.Err.Error())
	}
}

func TestRelativeLoadInLibraries(t *testing.T) {
	yamlTplData := []byte(`
#@ load("@library1:funcs.lib.yml", "yamlfunc")
#@ load("@library1:sub-dir/funcs.lib.txt", "textfunc")
#@ load("@library2:funcs.star", "starfunc")
#@ load("funcs.star", "localstarfunc")
yamlfunc: #@ yamlfunc()
textfunc: #@ textfunc()
starfunc: #@ starfunc()
localstarfunc: #@ localstarfunc()`)

	expectedYAMLTplData := `yamlfunc:
  yamlfunc: textfunc
textfunc: textfunc
starfunc:
- 1
- 2
localstarfunc:
- 3
- 4
`

	yamlFuncsData := []byte(`
#@ load("sub-dir/funcs.lib.txt", "textfunc")
#@ def/end yamlfunc():
yamlfunc: #@ textfunc()`)

	starlarkFuncsData := []byte(`
load("@funcs:funcs.star", "libstarfunc")
def starfunc():
  return libstarfunc()
end`)

	starlarkFuncsLibData := []byte(`
def libstarfunc():
  return [1,2]
end`)

	localStarlarkFuncsData := []byte(`
def localstarfunc():
  return [3,4]
end`)

	txtFuncsData := []byte(`(@ def textfunc(): @)textfunc(@ end @)`)

	filesToProcess := []*files.File{
		files.MustNewFileFromSource(files.NewBytesSource("tpl.yml", yamlTplData)),
		files.MustNewFileFromSource(files.NewBytesSource("funcs.star", localStarlarkFuncsData)),
		files.MustNewFileFromSource(files.NewBytesSource("_ytt_lib/library1/funcs.lib.yml", yamlFuncsData)),
		files.MustNewFileFromSource(files.NewBytesSource("_ytt_lib/library1/sub-dir/funcs.lib.txt", txtFuncsData)),
		files.MustNewFileFromSource(files.NewBytesSource("_ytt_lib/library2/funcs.star", starlarkFuncsData)),
		files.MustNewFileFromSource(files.NewBytesSource("_ytt_lib/library2/_ytt_lib/funcs/funcs.star", starlarkFuncsLibData)),
	}

	ui := cmdcore.NewPlainUI(false)
	opts := cmdtpl.NewOptions()

	out := opts.RunWithFiles(cmdtpl.TemplateInput{Files: filesToProcess}, ui)
	if out.Err != nil {
		t.Fatalf("Expected RunWithFiles to succeed, but was error: %s", out.Err)
	}

	if len(out.Files) != 1 {
		t.Fatalf("Expected number of output files to be 1, but was %d", len(out.Files))
	}

	file := out.Files[0]

	if file.RelativePath() != "tpl.yml" {
		t.Fatalf("Expected output file to be tpl.yml, but was %#v", file.RelativePath())
	}

	if string(file.Bytes()) != expectedYAMLTplData {
		t.Fatalf("Expected output file to have specific data, but was: >>>%s<<<", file.Bytes())
	}
}

func TestRelativeLoadInLibrariesForNonRootTemplates(t *testing.T) {
	expectedYAMLTplData := `libstarfunc:
- 1
- 2
`

	nonTopLevelYmlTplData := []byte(`
#@ load("@funcs:funcs.star", "libstarfunc")
libstarfunc: #@ libstarfunc()`)

	nonTopLevelStarlarkFuncsLibData := []byte(`
def libstarfunc():
  return [1,2]
end`)

	filesToProcess := []*files.File{
		files.MustNewFileFromSource(files.NewBytesSource("non-top-level/tpl.yml", nonTopLevelYmlTplData)),
		files.MustNewFileFromSource(files.NewBytesSource("non-top-level/_ytt_lib/funcs/funcs.star", nonTopLevelStarlarkFuncsLibData)),
	}

	ui := cmdcore.NewPlainUI(false)
	opts := cmdtpl.NewOptions()

	out := opts.RunWithFiles(cmdtpl.TemplateInput{Files: filesToProcess}, ui)
	if out.Err != nil {
		t.Fatalf("Expected RunWithFiles to succeed, but was error: %s", out.Err)
	}

	if len(out.Files) != 1 {
		t.Fatalf("Expected number of output files to be 1, but was %d", len(out.Files))
	}

	file := out.Files[0]

	if file.RelativePath() != "non-top-level/tpl.yml" {
		t.Fatalf("Expected output file to be non-top-level/tpl.yml, but was %#v", file.RelativePath())
	}

	if string(file.Bytes()) != expectedYAMLTplData {
		t.Fatalf("Expected output file to have specific data, but was: >>>%s<<<", file.Bytes())
	}
}

func TestIgnoreUnknownCommentsFalse(t *testing.T) {
	yamlTplData := []byte(`
# plain YAML comment
#@ load("funcs/funcs.lib.yml", "yamlfunc")
yamlfunc: #@ yamlfunc()`)

	yamlFuncsData := []byte(`
#@ def/end yamlfunc():
yamlfunc: yamlfunc`)

	filesToProcess := []*files.File{
		files.MustNewFileFromSource(files.NewBytesSource("tpl.yml", yamlTplData)),
		files.MustNewFileFromSource(files.NewBytesSource("funcs/funcs.lib.yml", yamlFuncsData)),
	}

	ui := cmdcore.NewPlainUI(false)
	opts := cmdtpl.NewOptions()

	out := opts.RunWithFiles(cmdtpl.TemplateInput{Files: filesToProcess}, ui)
	if out.Err == nil {
		t.Fatalf("Expected RunWithFiles to fail")
	}

	if out.Err.Error() != "Unknown comment syntax at line tpl.yml:2: ' plain YAML comment': Unknown metadata format (use '#@' or '#!')" {
		t.Fatalf("Expected RunWithFiles to fail with error, but was '%s'", out.Err.Error())
	}
}

func TestIgnoreUnknownCommentsTrue(t *testing.T) {
	yamlTplData := []byte(`
# plain YAML comment
#@ load("funcs/funcs.lib.yml", "yamlfunc")
yamlfunc: #@ yamlfunc()`)

	expectedYAMLTplData := `yamlfunc:
  yamlfunc: yamlfunc
`

	yamlFuncsData := []byte(`
#@ def/end yamlfunc():
yamlfunc: yamlfunc`)

	filesToProcess := []*files.File{
		files.MustNewFileFromSource(files.NewBytesSource("tpl.yml", yamlTplData)),
		files.MustNewFileFromSource(files.NewBytesSource("funcs/funcs.lib.yml", yamlFuncsData)),
	}

	ui := cmdcore.NewPlainUI(false)
	opts := cmdtpl.NewOptions()
	opts.IgnoreUnknownComments = true

	out := opts.RunWithFiles(cmdtpl.TemplateInput{Files: filesToProcess}, ui)
	if out.Err != nil {
		t.Fatalf("Expected RunWithFiles to succeed, but was error: %s", out.Err)
	}

	if len(out.Files) != 1 {
		t.Fatalf("Expected number of output files to be 1, but was %d", len(out.Files))
	}

	file := out.Files[0]

	if file.RelativePath() != "tpl.yml" {
		t.Fatalf("Expected output file to be tpl.yml, but was %#v", file.RelativePath())
	}

	if string(file.Bytes()) != expectedYAMLTplData {
		t.Fatalf("Expected output file to have specific data, but was: >>>%s<<<", file.Bytes())
	}
}

func TestParseErrTemplateFile(t *testing.T) {
	yamlTplData := []byte(`
key: val
yamlfunc yamlfunc`)

	filesToProcess := []*files.File{
		files.MustNewFileFromSource(files.NewBytesSource("tpl.yml", yamlTplData)),
	}

	ui := cmdcore.NewPlainUI(false)
	opts := cmdtpl.NewOptions()

	out := opts.RunWithFiles(cmdtpl.TemplateInput{Files: filesToProcess}, ui)
	if out.Err == nil {
		t.Fatalf("Expected RunWithFiles to fail")
	}

	if out.Err.Error() != "Unmarshaling YAML template 'tpl.yml': yaml: line 4: could not find expected ':'" {
		t.Fatalf("Expected RunWithFiles to fail with error, but was '%s'", out.Err.Error())
	}
}

func TestParseErrLoadFile(t *testing.T) {
	yamlTplData := []byte(`
#@ load("funcs/funcs.lib.yml", "yamlfunc")
yamlfunc: #@ yamlfunc()`)

	yamlFuncsData := []byte(`
#@ def yamlfunc():
key: val
yamlfunc yamlfunc
#@ end`)

	filesToProcess := []*files.File{
		files.MustNewFileFromSource(files.NewBytesSource("tpl.yml", yamlTplData)),
		files.MustNewFileFromSource(files.NewBytesSource("funcs/funcs.lib.yml", yamlFuncsData)),
	}

	ui := cmdcore.NewPlainUI(false)
	opts := cmdtpl.NewOptions()

	out := opts.RunWithFiles(cmdtpl.TemplateInput{Files: filesToProcess}, ui)
	if out.Err == nil {
		t.Fatalf("Expected RunWithFiles to fail")
	}

	if !strings.Contains(out.Err.Error(), "cannot load funcs/funcs.lib.yml: Unmarshaling YAML template 'funcs/funcs.lib.yml': yaml: line 5: could not find expected ':'") {
		t.Fatalf("Expected RunWithFiles to fail with error, but was '%s'", out.Err.Error())
	}
}
