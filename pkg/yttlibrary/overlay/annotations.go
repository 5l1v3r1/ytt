package overlay

import (
	"fmt"

	"github.com/get-ytt/ytt/pkg/structmeta"
	"github.com/get-ytt/ytt/pkg/template"
	"github.com/get-ytt/ytt/pkg/yamlmeta"
)

const (
	AnnotationNs structmeta.AnnotationNs = "overlay"

	AnnotationMerge   structmeta.AnnotationName = "overlay/merge" // default
	AnnotationRemove  structmeta.AnnotationName = "overlay/remove"
	AnnotationReplace structmeta.AnnotationName = "overlay/replace"
	AnnotationInsert  structmeta.AnnotationName = "overlay/insert" // array only
	AnnotationAppend  structmeta.AnnotationName = "overlay/append" // array only

	AnnotationMatch structmeta.AnnotationName = "overlay/match"
)

var (
	allOps = []structmeta.AnnotationName{
		AnnotationMerge,
		AnnotationRemove,
		AnnotationReplace,
		AnnotationInsert,
		AnnotationAppend,
	}
)

func whichOp(node yamlmeta.Node) (structmeta.AnnotationName, error) {
	var foundOp structmeta.AnnotationName

	for _, op := range allOps {
		if template.NewAnnotations(node).Has(op) {
			if len(foundOp) > 0 {
				return "", fmt.Errorf("Expected to find only one overlay operation")
			}
			foundOp = op
		}
	}

	if len(foundOp) == 0 {
		foundOp = AnnotationMerge
	}

	return foundOp, nil
}
