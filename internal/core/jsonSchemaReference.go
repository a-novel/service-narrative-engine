package core

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

const (
	jsonSchemaDraft7    = "http://json-schema.org/draft-07/schema#"
	jsonSchemaDraft7TLS = "https://json-schema.org/draft-07/schema#"
)

var (
	errJSONSchemaReferenceInvalid = errors.New("invalid JSON Schema reference")
	jsonSchemaReferenceKeywords   = [...]string{"$ref", "$dynamicRef"}
)

type partialSchemaResource struct {
	uri     *url.URL
	pointer string
	anchors map[string]string
}

type partialSchemaLocation struct {
	value    any
	pointer  string
	resource *partialSchemaResource
}

// partialSchemaReferenceResolver mirrors jsonschema-go's in-document resource
// and anchor scoping. Resolved reference targets are not exposed by that
// package, but partial oneOf classification must inspect them without loading
// remote schemas.
type partialSchemaReferenceResolver struct {
	draft7               bool
	resources            map[string]*partialSchemaResource
	locations            map[string]partialSchemaLocation
	partializableSchemas []partialSchemaLocation
}

func newPartialSchemaReferenceResolver(root any) (partialSchemaReferenceResolver, error) {
	rootResource := &partialSchemaResource{
		uri:     &url.URL{},
		pointer: "",
		anchors: make(map[string]string),
	}

	resolver := partialSchemaReferenceResolver{
		draft7: jsonSchemaUsesDraft7(root),
		resources: map[string]*partialSchemaResource{
			"": rootResource,
		},
		locations: make(map[string]partialSchemaLocation),
	}

	partializableSchemas := make([]partialSchemaLocation, 0)

	err := resolver.indexSchema(root, "", rootResource, true, true, &partializableSchemas)
	if err != nil {
		return partialSchemaReferenceResolver{}, err
	}

	resolver.partializableSchemas = partializableSchemas

	return resolver, nil
}

func jsonSchemaUsesDraft7(root any) bool {
	schema, ok := root.(map[string]any)
	if !ok {
		return false
	}

	version, _ := schema["$schema"].(string)

	return version == jsonSchemaDraft7 || version == jsonSchemaDraft7TLS
}

func (resolver *partialSchemaReferenceResolver) indexSchema(
	value any,
	pointer string,
	parentResource *partialSchemaResource,
	root bool,
	partializable bool,
	partializableSchemas *[]partialSchemaLocation,
) error {
	schema, objectSchema := value.(map[string]any)
	if !objectSchema {
		if _, booleanSchema := value.(bool); booleanSchema {
			resolver.locations[pointer] = partialSchemaLocation{
				value:    value,
				pointer:  pointer,
				resource: parentResource,
			}
		}

		return nil
	}

	resource, err := resolver.indexSchemaResource(schema, pointer, parentResource, root)
	if err != nil {
		return err
	}

	location := partialSchemaLocation{
		value:    schema,
		pointer:  pointer,
		resource: resource,
	}
	resolver.locations[pointer] = location

	if partializable {
		*partializableSchemas = append(*partializableSchemas, location)
	}

	if !resolver.draft7 {
		if anchor, hasAnchor := schema["$anchor"].(string); hasAnchor {
			err = resolver.addAnchor(resource, pointer, anchor)
			if err != nil {
				return err
			}
		}

		if anchor, hasAnchor := schema["$dynamicAnchor"].(string); hasAnchor {
			err = resolver.addAnchor(resource, pointer, anchor)
			if err != nil {
				return err
			}
		}
	}

	return forEachPartialSchemaChildAt(
		schema,
		pointer,
		true,
		true,
		func(child any, childPointer string, predicate bool) error {
			return resolver.indexSchema(
				child,
				childPointer,
				resource,
				false,
				partializable && !predicate,
				partializableSchemas,
			)
		},
	)
}

func (resolver *partialSchemaReferenceResolver) indexSchemaResource(
	schema map[string]any,
	pointer string,
	parentResource *partialSchemaResource,
	root bool,
) (*partialSchemaResource, error) {
	id, hasID := schema["$id"].(string)
	if !hasID || id == "" {
		return parentResource, nil
	}

	parsedID, err := url.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("parse $id %q: %w", id, err)
	}

	if resolver.draft7 {
		if reference, hasReference := schema["$ref"].(string); hasReference && reference != "" {
			return parentResource, nil
		}

		if parsedID.Fragment != "" {
			anchor := strings.TrimPrefix(id, "#")

			err = resolver.addAnchor(parentResource, pointer, anchor)
			if err != nil {
				return nil, err
			}

			return parentResource, nil
		}
	} else if parsedID.Fragment != "" {
		return nil, fmt.Errorf("%w: $id %q must not have a fragment", errJSONSchemaReferenceInvalid, id)
	}

	resolvedID := parentResource.uri.ResolveReference(parsedID)
	if !resolvedID.IsAbs() {
		return nil, fmt.Errorf(
			"%w: $id %q does not resolve to an absolute URI",
			errJSONSchemaReferenceInvalid,
			id,
		)
	}

	if root {
		parentResource.uri = resolvedID
		resolver.resources[resolvedID.String()] = parentResource

		return parentResource, nil
	}

	resource := &partialSchemaResource{
		uri:     resolvedID,
		pointer: pointer,
		anchors: make(map[string]string),
	}
	resolver.resources[resolvedID.String()] = resource

	return resource, nil
}

func (resolver *partialSchemaReferenceResolver) addAnchor(
	resource *partialSchemaResource,
	pointer string,
	anchor string,
) error {
	if anchor == "" {
		return nil
	}

	if _, exists := resource.anchors[anchor]; exists {
		return fmt.Errorf("%w: duplicate anchor %q", errJSONSchemaReferenceInvalid, anchor)
	}

	resource.anchors[anchor] = pointer

	return nil
}

func (resolver *partialSchemaReferenceResolver) resolve(
	source partialSchemaLocation,
	reference string,
) (partialSchemaLocation, error) {
	parsedReference, err := url.Parse(reference)
	if err != nil {
		return partialSchemaLocation{}, fmt.Errorf("parse reference %q: %w", reference, err)
	}

	resolvedReference := source.resource.uri.ResolveReference(parsedReference)
	fragment := resolvedReference.Fragment

	resourceURI := *resolvedReference
	resourceURI.Fragment = ""
	resourceURI.RawFragment = ""

	resource := resolver.resources[resourceURI.String()]
	if resource == nil {
		return partialSchemaLocation{}, fmt.Errorf(
			"%w: %q does not resolve within the root document",
			errJSONSchemaReferenceInvalid,
			reference,
		)
	}

	var targetPointer string

	switch {
	case fragment == "":
		targetPointer = resource.pointer
	case strings.HasPrefix(fragment, "/"):
		targetPointer = resource.pointer + fragment
	default:
		var hasAnchor bool

		targetPointer, hasAnchor = resource.anchors[fragment]
		if !hasAnchor {
			return partialSchemaLocation{}, fmt.Errorf(
				"%w: %q has no anchor %q",
				errJSONSchemaReferenceInvalid,
				reference,
				fragment,
			)
		}
	}

	target, exists := resolver.locations[targetPointer]
	if !exists {
		return partialSchemaLocation{}, fmt.Errorf(
			"%w: %q does not point to a schema",
			errJSONSchemaReferenceInvalid,
			reference,
		)
	}

	return target, nil
}

// forEachPartialSchemaChildAt can omit declaration containers so an unused
// $defs entry cannot make an otherwise scalar oneOf look presence-dependent.
// Referenced declarations are followed explicitly by the resolver.
func forEachPartialSchemaChildAt(
	schema map[string]any,
	pointer string,
	includePredicates bool,
	includeDeclarations bool,
	visit func(any, string, bool) error,
) error {
	for _, keyword := range jsonSchemaSingleSubschemaKeywords {
		err := visitPartialSchemaValueAt(
			schema[keyword],
			appendJSONSchemaPointer(pointer, keyword),
			false,
			visit,
		)
		if err != nil {
			return err
		}
	}

	if includePredicates {
		for _, keyword := range jsonSchemaPredicateSubschemaKeywords {
			err := visitPartialSchemaValueAt(
				schema[keyword],
				appendJSONSchemaPointer(pointer, keyword),
				true,
				visit,
			)
			if err != nil {
				return err
			}
		}
	}

	for _, keyword := range jsonSchemaSubschemaArrayKeywords {
		err := visitPartialSchemaValueAt(
			schema[keyword],
			appendJSONSchemaPointer(pointer, keyword),
			false,
			visit,
		)
		if err != nil {
			return err
		}
	}

	for _, keyword := range jsonSchemaSubschemaMapKeywords {
		if !includeDeclarations && (keyword == "$defs" || keyword == "definitions") {
			continue
		}

		children, childrenOK := schema[keyword].(map[string]any)
		if !childrenOK {
			continue
		}

		for key, child := range children {
			err := visitPartialSchemaValueAt(
				child,
				appendJSONSchemaPointer(pointer, keyword, key),
				false,
				visit,
			)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func visitPartialSchemaValueAt(
	value any,
	pointer string,
	predicate bool,
	visit func(any, string, bool) error,
) error {
	switch value := value.(type) {
	case map[string]any, bool:
		return visit(value, pointer, predicate)
	case []any:
		for index, child := range value {
			err := visitPartialSchemaValueAt(
				child,
				appendJSONSchemaPointer(pointer, strconv.Itoa(index)),
				predicate,
				visit,
			)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func appendJSONSchemaPointer(pointer string, segments ...string) string {
	var builder strings.Builder

	builder.WriteString(pointer)

	for _, segment := range segments {
		builder.WriteByte('/')
		builder.WriteString(escapeJSONSchemaPointerSegment(segment))
	}

	return builder.String()
}

func escapeJSONSchemaPointerSegment(segment string) string {
	segment = strings.ReplaceAll(segment, "~", "~0")

	return strings.ReplaceAll(segment, "/", "~1")
}
