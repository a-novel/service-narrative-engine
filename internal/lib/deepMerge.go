package lib

func DeepMerge(src, dst map[string]any) map[string]any {
	if src == nil {
		return dst
	}

	out := src

	for k, v := range dst {
		srcV := out[k]

		// Key not in src/output, take value from dst.
		if srcV == nil {
			out[k] = v

			continue
		}

		srcVMap, srcVMapOk := srcV.(map[string]any)
		vMap, vMapOk := v.(map[string]any)

		// Only maps can be deep-merged. Otherwise, replace the value entirely.
		if !srcVMapOk || !vMapOk {
			out[k] = v

			continue
		}

		out[k] = DeepMerge(srcVMap, vMap)
	}

	return out
}
