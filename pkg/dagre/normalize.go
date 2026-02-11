package dagre

import "fmt"

func makeEdgeID(e EdgeObj) string {
	return fmt.Sprintf("%s\x00%s\x00%s", e.V, e.W, e.Name)
}
