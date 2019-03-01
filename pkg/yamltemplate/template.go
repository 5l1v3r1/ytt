package yamltemplate

import (
	"bytes"
	"fmt"

	"github.com/get-ytt/ytt/pkg/filepos"
	"github.com/get-ytt/ytt/pkg/template"
	"github.com/get-ytt/ytt/pkg/texttemplate"
	"github.com/get-ytt/ytt/pkg/yamlmeta"
)

type Template struct {
	name         string
	docSet       *yamlmeta.DocumentSet
	nodes        *template.Nodes
	instructions *template.InstructionSet

	// memoized source lines
	srcLinesByLine map[int]*template.SourceLine
}

func NewTemplate(name string) *Template {
	return &Template{name: name, instructions: template.NewInstructionSet()}
}

func (e *Template) Compile(docSet *yamlmeta.DocumentSet) (*template.CompiledTemplate, error) {
	e.docSet = docSet
	e.nodes = template.NewNodes()

	code, err := e.build(docSet, nil, template.NodeTagRoot)
	if err != nil {
		return nil, err
	}

	code = append([]template.TemplateLine{
		e.resetCtxType(),
		{Instruction: e.instructions.NewStartCtx(EvaluationCtxDialectName)},
	}, code...)

	code = append(code, template.TemplateLine{
		Instruction: e.instructions.NewEndCtxNone(), // TODO ideally we would return array of docset
	})

	return template.NewCompiledTemplate(e.name, code, e.instructions, e.nodes, template.EvaluationCtxDialects{
		EvaluationCtxDialectName:              EvaluationCtx{},
		texttemplate.EvaluationCtxDialectName: texttemplate.EvaluationCtx{},
	}), nil
}

func (e *Template) build(val interface{}, parentNode yamlmeta.Node, parentTag template.NodeTag) ([]template.TemplateLine, error) {
	node, ok := val.(yamlmeta.Node)
	if !ok {
		if valStr, ok := val.(string); ok {
			return e.buildString(valStr, parentNode, parentTag)
		}

		return []template.TemplateLine{{
			Instruction: e.instructions.NewSetNode(parentTag).WithDebug(e.debugComment(parentNode)),
			SourceLine:  e.newSourceLine(parentNode.GetPosition()),
		}}, nil
	}

	code := []template.TemplateLine{}
	nodeTag := e.nodes.AddNode(node, parentTag)

	metas, err := NewMetas(node)
	if err != nil {
		return nil, err
	}

	for _, blk := range metas.Block {
		code = append(code, template.TemplateLine{
			Instruction: e.instructions.NewCode(blk.Data),
			SourceLine:  e.newSourceLine(blk.Position),
		})
	}

	for _, metaAndAnn := range metas.Annotations {
		code = append(code, template.TemplateLine{
			Instruction: e.instructions.NewStartNodeAnnotation(nodeTag, *metaAndAnn.Annotation).WithDebug(e.debugComment(node)),
			SourceLine:  e.newSourceLine(metaAndAnn.Meta.Position),
		})
	}

	code = append(code, template.TemplateLine{
		Instruction: e.instructions.NewStartNode(nodeTag).WithDebug(e.debugComment(node)),
		SourceLine:  e.newSourceLine(node.GetPosition()),
	})

	if len(metas.Values) > 0 {
		for _, val := range metas.Values {
			code = append(code, template.TemplateLine{
				Instruction: e.instructions.NewSetNodeValue(nodeTag, val.Data).WithDebug(e.debugComment(node)),
				SourceLine:  e.newSourceLine(val.Position),
			})
		}
	} else {
		for _, childVal := range node.GetValues() {
			childCode, err := e.build(childVal, node, nodeTag)
			if err != nil {
				return nil, err
			}
			code = append(code, childCode...)
		}
	}

	if metas.NeedsEnd() {
		code = append(code, template.TemplateLine{
			// TODO should we set position to start node?
			Instruction: e.instructions.NewCode("end"),
		})
	}

	return code, nil
}

func (e *Template) buildString(val string, parentNode yamlmeta.Node,
	parentTag template.NodeTag) ([]template.TemplateLine, error) {

	// TODO line numbers for inlined template
	name := fmt.Sprintf("%s (%s)", e.name, parentNode.GetPosition().AsString())

	textRoot, err := texttemplate.NewParser().Parse([]byte(val), name)
	if err != nil {
		return nil, err
	}

	code, err := texttemplate.NewTemplate(name).CompileInline(textRoot, e.instructions, e.nodes)
	if err != nil {
		return nil, err
	}

	lastInstruction := code[len(code)-1].Instruction
	if lastInstruction.Op() != e.instructions.EndCtx {
		return nil, fmt.Errorf("Expected last instruction to be endctx, but was %#v", lastInstruction.Op())
	}

	code[len(code)-1] = template.TemplateLine{
		Instruction: e.instructions.NewSetNodeValue(
			parentTag, lastInstruction.AsString()).WithDebug(e.debugComment(parentNode)),
		SourceLine: e.newSourceLine(parentNode.GetPosition()),
	}

	code = append(code, e.resetCtxType())

	return code, nil
}

func (e *Template) resetCtxType() template.TemplateLine {
	return template.TemplateLine{
		Instruction: e.instructions.NewSetCtxType(EvaluationCtxDialectName),
	}
}

func (e *Template) debugComment(node yamlmeta.Node) string {
	var details string

	switch typedNode := node.(type) {
	case *yamlmeta.MapItem:
		details = fmt.Sprintf(" key=%s", typedNode.Key)
	case *yamlmeta.ArrayItem:
		details = " idx=?"
	}

	return fmt.Sprintf("%T%s", node, details) // TODO, node.GetRef())
}

func (e *Template) newSourceLine(pos *filepos.Position) *template.SourceLine {
	if pos.IsKnown() {
		if sourceLine, ok := e.sourceCodeLines()[pos.Line()]; ok {
			return &template.SourceLine{
				Position: pos,
				Content:  sourceLine.Content,
			}
		}
	}
	return &template.SourceLine{Position: pos}
}

func (e *Template) sourceCodeLines() map[int]*template.SourceLine {
	if e.srcLinesByLine != nil {
		return e.srcLinesByLine
	}

	e.srcLinesByLine = map[int]*template.SourceLine{}

	sourceCode, present := e.docSet.AsSourceBytes()
	if present {
		for i, line := range bytes.Split(sourceCode, []byte("\n")) {
			srcLine := &template.SourceLine{Content: string(line)}
			e.srcLinesByLine[filepos.NewPosition(i+1).Line()] = srcLine
		}
	}

	return e.srcLinesByLine
}
