package overlay

import (
	"github.com/get-ytt/ytt/pkg/yamlmeta"
)

func (o OverlayOp) mergeArrayItem(
	leftArray *yamlmeta.Array, newItem *yamlmeta.ArrayItem) error {

	ann, err := NewArrayItemMatchAnnotation(newItem, o.Thread)
	if err != nil {
		return err
	}

	leftIdxs, err := ann.Indexes(leftArray)
	if err != nil {
		return err
	}

	if len(leftIdxs) == 0 {
		return o.appendArrayItem(leftArray, newItem)
	}

	for _, leftIdx := range leftIdxs {
		replace, err := o.apply(leftArray.Items[leftIdx].Value, newItem.Value)
		if err != nil {
			return err
		}
		if replace {
			leftArray.Items[leftIdx].Value = newItem.Value
		}
	}

	return nil
}

func (o OverlayOp) removeArrayItem(
	leftArray *yamlmeta.Array, newItem *yamlmeta.ArrayItem) error {

	ann, err := NewArrayItemMatchAnnotation(newItem, o.Thread)
	if err != nil {
		return err
	}

	leftIdxs, err := ann.Indexes(leftArray)
	if err != nil {
		return err
	}

	for _, leftIdx := range leftIdxs {
		leftArray.Items[leftIdx] = nil
	}

	return nil
}

func (o OverlayOp) replaceArrayItem(
	leftArray *yamlmeta.Array, newItem *yamlmeta.ArrayItem) error {

	ann, err := NewArrayItemMatchAnnotation(newItem, o.Thread)
	if err != nil {
		return err
	}

	leftIdxs, err := ann.Indexes(leftArray)
	if err != nil {
		return err
	}

	for _, leftIdx := range leftIdxs {
		leftArray.Items[leftIdx] = newItem.DeepCopy()
	}

	return nil
}

func (o OverlayOp) insertArrayItem(
	leftArray *yamlmeta.Array, newItem *yamlmeta.ArrayItem) error {

	ann, err := NewArrayItemMatchAnnotation(newItem, o.Thread)
	if err != nil {
		return err
	}

	leftIdxs, err := ann.Indexes(leftArray)
	if err != nil {
		return err
	}

	insertAnn, err := NewInsertAnnotation(newItem)
	if err != nil {
		return err
	}

	updatedItems := []*yamlmeta.ArrayItem{}

	for i, leftItem := range leftArray.Items {
		matched := false
		for _, leftIdx := range leftIdxs {
			if i == leftIdx {
				matched = true
				if insertAnn.IsBefore() {
					updatedItems = append(updatedItems, newItem.DeepCopy())
				}
				updatedItems = append(updatedItems, leftItem)
				if insertAnn.IsAfter() {
					updatedItems = append(updatedItems, newItem.DeepCopy())
				}
				break
			}
		}
		if !matched {
			updatedItems = append(updatedItems, leftItem)
		}
	}

	leftArray.Items = updatedItems

	return nil
}

func (o OverlayOp) appendArrayItem(
	leftArray *yamlmeta.Array, newItem *yamlmeta.ArrayItem) error {

	// No need to traverse further
	leftArray.Items = append(leftArray.Items, newItem.DeepCopy())
	return nil
}
