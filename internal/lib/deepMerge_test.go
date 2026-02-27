package lib_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/internal/lib"
)

func TestDeepMerge(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		src    map[string]any
		dst    map[string]any
		expect map[string]any
	}{
		{
			name:   "NilSrc",
			src:    nil,
			dst:    map[string]any{"a": 1},
			expect: map[string]any{"a": 1},
		},
		{
			name:   "NilDst",
			src:    map[string]any{"a": 1},
			dst:    nil,
			expect: map[string]any{"a": 1},
		},
		{
			name:   "BothNil",
			src:    nil,
			dst:    nil,
			expect: nil,
		},
		{
			name:   "EmptySrc",
			src:    map[string]any{},
			dst:    map[string]any{"a": 1},
			expect: map[string]any{"a": 1},
		},
		{
			name:   "EmptyDst",
			src:    map[string]any{"a": 1},
			dst:    map[string]any{},
			expect: map[string]any{"a": 1},
		},
		{
			name:   "NoOverlap",
			src:    map[string]any{"a": 1},
			dst:    map[string]any{"b": 2},
			expect: map[string]any{"a": 1, "b": 2},
		},
		{
			name:   "FullOverlap",
			src:    map[string]any{"a": 1, "b": 2},
			dst:    map[string]any{"a": 10, "b": 20},
			expect: map[string]any{"a": 10, "b": 20},
		},
		{
			name:   "PartialOverlap",
			src:    map[string]any{"a": 1, "b": 2},
			dst:    map[string]any{"b": 20, "c": 30},
			expect: map[string]any{"a": 1, "b": 20, "c": 30},
		},
		{
			name: "NestedMapsNoOverlap",
			src: map[string]any{
				"nested": map[string]any{"x": 1},
			},
			dst: map[string]any{
				"nested": map[string]any{"y": 2},
			},
			expect: map[string]any{
				"nested": map[string]any{"x": 1, "y": 2},
			},
		},
		{
			name: "NestedMapsWithOverlap",
			src: map[string]any{
				"nested": map[string]any{"x": 1, "y": 2},
			},
			dst: map[string]any{
				"nested": map[string]any{"y": 20, "z": 30},
			},
			expect: map[string]any{
				"nested": map[string]any{"x": 1, "y": 20, "z": 30},
			},
		},
		{
			name: "DeeplyNestedMaps",
			src: map[string]any{
				"level1": map[string]any{
					"level2": map[string]any{"a": 1},
				},
			},
			dst: map[string]any{
				"level1": map[string]any{
					"level2": map[string]any{"b": 2},
				},
			},
			expect: map[string]any{
				"level1": map[string]any{
					"level2": map[string]any{"a": 1, "b": 2},
				},
			},
		},
		{
			name: "SrcMapDstScalarOverwrites",
			src: map[string]any{
				"key": map[string]any{"a": 1},
			},
			dst: map[string]any{
				"key": "scalar",
			},
			expect: map[string]any{
				"key": "scalar",
			},
		},
		{
			name: "SrcScalarDstMapOverwrites",
			src: map[string]any{
				"key": "scalar",
			},
			dst: map[string]any{
				"key": map[string]any{"a": 1},
			},
			expect: map[string]any{
				"key": map[string]any{"a": 1},
			},
		},
		{
			name: "MixedTopLevelAndNested",
			src: map[string]any{
				"scalar":   "src-value",
				"nested":   map[string]any{"x": 1},
				"only-src": true,
			},
			dst: map[string]any{
				"scalar":   "dst-value",
				"nested":   map[string]any{"y": 2},
				"only-dst": true,
			},
			expect: map[string]any{
				"scalar":   "dst-value",
				"nested":   map[string]any{"x": 1, "y": 2},
				"only-src": true,
				"only-dst": true,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := lib.DeepMerge(tc.src, tc.dst)
			require.Equal(t, tc.expect, result)
		})
	}
}
