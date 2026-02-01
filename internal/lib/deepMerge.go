package lib

func DeepMerge(src, dst map[string]any) map[string]any {
	if src == nil {
		src = make(map[string]any)
	}

	out := src

	for k, v := range dst {
		srcV, ok := out[k]

		// Key not in dest, add it.
		if !ok {
			out[k] = v

			continue
		}

		srcVMap, srcVMapOk := srcV.(map[string]any)
		vMap, vMapOk := v.(map[string]any)

		// Only maps can be deep merged. Otherwise, replace the value entirely.
		if !srcVMapOk || !vMapOk {
			out[k] = v

			continue
		}

		out[k] = DeepMerge(srcVMap, vMap)
	}

	return out
}
