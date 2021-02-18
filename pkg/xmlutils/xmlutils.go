package xmlutils

import (
	"fmt"

	"github.com/beevik/etree"
)

// ValueOfAttribute returns an attribute value if it's found in given element
func ValueOfAttribute(elemName string, attrName string, doc *etree.Document) (string, error) {
	return Element(elemName, doc, func(elem *etree.Element) (string, error) {
		return Attrib(attrName, elem, func(attr *etree.Attr) (string, error) {
			return attr.Value, nil
		})
	})

}

// Element returns element if it's found on given document
func Element(elemName string, doc *etree.Document, closure func(*etree.Element) (string, error)) (string, error) {
	elem := doc.FindElement(elemName)
	if elem == nil {
		return "", fmt.Errorf("can't find element %s", elemName)
	}
	return closure(elem)
}

// Attrib returns attribute value if it's found on given element
func Attrib(attrName string, elem *etree.Element, closure func(*etree.Attr) (string, error)) (string, error) {
	attr := elem.SelectAttr(attrName)
	if attr == nil {
		return "", fmt.Errorf("can't find attribute %s", attrName)
	}
	return closure(attr)
}
