package template

import (
	"fmt"
	"time"

	cmdcore "github.com/k14s/ytt/pkg/cmd/core"
	"github.com/k14s/ytt/pkg/files"
	"github.com/k14s/ytt/pkg/texttemplate"
	"github.com/k14s/ytt/pkg/workspace"
	"github.com/k14s/ytt/pkg/yamlmeta"
	"github.com/k14s/ytt/pkg/yttlibrary"
	"github.com/spf13/cobra"
)

type TemplateOptions struct {
	Debug                  bool
	BulkFilesSourceOpts    BulkFilesSourceOpts
	RegularFilesSourceOpts RegularFilesSourceOpts
}

type TemplateInput struct {
	Files []*files.File
}

type TemplateOutput struct {
	Files  []files.OutputFile
	DocSet *yamlmeta.DocumentSet
	Err    error
}

type FileSource interface {
	HasInput() bool
	HasOutput() bool
	Input() (TemplateInput, error)
	Output(TemplateOutput) error
}

var _ []FileSource = []FileSource{&BulkFilesSource{}, &RegularFilesSource{}}

func NewOptions() *TemplateOptions {
	return &TemplateOptions{}
}

func NewCmd(o *TemplateOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "template",
		Aliases: []string{"t", "tpl"},
		Short:   "Process YAML templates",
		RunE:    func(_ *cobra.Command, _ []string) error { return o.Run() },
	}
	cmd.Flags().BoolVar(&o.Debug, "debug", false, "Enable debug output")
	o.BulkFilesSourceOpts.Set(cmd)
	o.RegularFilesSourceOpts.Set(cmd)
	return cmd
}

func (o *TemplateOptions) Run() error {
	ui := cmdcore.NewPlainUI(o.Debug)
	t1 := time.Now()

	defer func() {
		ui.Debugf("total: %s\n", time.Now().Sub(t1))
	}()

	srcs := []FileSource{
		NewBulkFilesSource(o.BulkFilesSourceOpts, ui),
		NewRegularFilesSource(o.RegularFilesSourceOpts, ui),
	}

	in, err := o.pickSource(srcs, func(s FileSource) bool { return s.HasInput() }).Input()
	if err != nil {
		return err
	}

	out := o.RunWithFiles(in, ui)

	return o.pickSource(srcs, func(s FileSource) bool { return s.HasOutput() }).Output(out)
}

func (o *TemplateOptions) RunWithFiles(in TemplateInput, ui cmdcore.PlainUI) TemplateOutput {
	outputFiles := []files.OutputFile{}
	outputDocSets := map[string]*yamlmeta.DocumentSet{}

	rootLibrary := workspace.NewRootLibrary(in.Files)
	rootLibrary.Print(ui.DebugWriter())

	forOutputFiles, values, err := o.categorizeFiles(rootLibrary.ListAccessibleFiles())
	if err != nil {
		return TemplateOutput{Err: err}
	}

	loader := workspace.NewTemplateLoader(values, ui)

	for _, file := range forOutputFiles {
		// TODO find more generic way
		switch file.Type() {
		case files.TypeYAML:
			_, resultVal, err := loader.EvalYAML(rootLibrary, file)
			if err != nil {
				return TemplateOutput{Err: err}
			}

			resultDocSet := resultVal.(*yamlmeta.DocumentSet)
			outputDocSets[file.RelativePath()] = resultDocSet

		case files.TypeText:
			_, resultVal, err := loader.EvalText(rootLibrary, file)
			if err != nil {
				return TemplateOutput{Err: err}
			}

			resultStr := resultVal.(*texttemplate.NodeRoot).AsString()

			ui.Debugf("### %s result\n%s", file.RelativePath(), resultStr)
			outputFiles = append(outputFiles, files.NewOutputFile(file.RelativePath(), []byte(resultStr)))

		default:
			return TemplateOutput{Err: fmt.Errorf("Unknown file type")}
		}
	}

	outputDocSets, err = OverlayPostProcessing{outputDocSets}.Apply()
	if err != nil {
		return TemplateOutput{Err: err}
	}

	combinedDocSet := &yamlmeta.DocumentSet{}

	for relPath, docSet := range outputDocSets {
		combinedDocSet.Items = append(combinedDocSet.Items, docSet.Items...)

		resultDocBytes, err := docSet.AsBytes()
		if err != nil {
			return TemplateOutput{Err: fmt.Errorf("Marshaling template result: %s", err)}
		}

		ui.Debugf("### %s result\n%s", relPath, resultDocBytes)
		outputFiles = append(outputFiles, files.NewOutputFile(relPath, resultDocBytes))
	}

	return TemplateOutput{Files: outputFiles, DocSet: combinedDocSet}
}

func (o *TemplateOptions) categorizeFiles(allFiles []*files.File) ([]*files.File, interface{}, error) {
	allFiles, values, err := o.extractValues(allFiles)
	if err != nil {
		return nil, nil, err
	}

	forOutputFiles := []*files.File{}

	for _, file := range allFiles {
		if file.IsForOutput() {
			forOutputFiles = append(forOutputFiles, file)
		}
	}

	return forOutputFiles, values, nil
}

func (o *TemplateOptions) extractValues(fs []*files.File) ([]*files.File, interface{}, error) {
	var foundValues interface{}
	var valuesFile *files.File
	var newFs []*files.File

	for _, file := range fs {
		if file.Type() == files.TypeYAML && file.IsTemplate() {
			fileBs, err := file.Bytes()
			if err != nil {
				return nil, nil, err
			}

			docSet, err := yamlmeta.NewDocumentSetFromBytes(fileBs, file.RelativePath())
			if err != nil {
				return nil, nil, fmt.Errorf("Unmarshaling YAML template: %s", err)
			}

			values, found, err := yttlibrary.DataValues{docSet}.Find()
			if err != nil {
				return nil, nil, err
			}

			if found {
				if valuesFile != nil {
					// TODO until overlays are here
					return nil, nil, fmt.Errorf(
						"Template values could only be specified once, but found multiple (%s, %s)",
						valuesFile.RelativePath(), file.RelativePath())
				}
				valuesFile = file
				foundValues = values
				continue
			}
		}

		newFs = append(newFs, file)
	}

	return newFs, foundValues, nil
}

func (o *TemplateOptions) pickSource(srcs []FileSource, pickFunc func(FileSource) bool) FileSource {
	for _, src := range srcs {
		if pickFunc(src) {
			return src
		}
	}
	return srcs[len(srcs)-1]
}
