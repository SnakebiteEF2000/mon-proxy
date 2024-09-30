package normalize

import (
	"github.com/SnakebiteEF2000/mon-proxy/internal/logger"
)

var log = logger.Log

// !! Normalize is irrelevant !!
// Will be removed later
func NormalizePath(path string) string {
	log.Debugf("normalize path: %s", path)
	return path
}

/*func NormalizePath(path string) string {
	log.Debugf("raw request path: %s", path)

	// Remove the scheme and host if present
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		parts := strings.SplitN(path, "/", 4)
		if len(parts) >= 4 {
			path = "/" + parts[3]
		}
	}

	// Split the path into parts
	parts := strings.Split(path, "/")

	// Remove empty parts and handle version prefix
	var cleanParts []string
	for i, part := range parts {
		if part == "" {
			continue
		}
		if i == 0 && strings.HasPrefix(part, "v") && strings.Contains(part, ".") {
			continue // Skip version prefix
		}
		cleanParts = append(cleanParts, part)
	}

	// Reconstruct the path
	normalizedPath := "/" + strings.Join(cleanParts, "/")

	log.Debugf("normalized path: %s", normalizedPath)
	return normalizedPath
}*/

/*func NormalizePath(path string) string {
	log.Debugf("raw path: %s", path)
	parts := strings.Split(path, "/")

	var prefixIndex int
	for i, part := range parts {
		if strings.HasPrefix(part, "v") {
			prefixIndex = i
			break
		}
	}

	if prefixIndex != -1 && len(parts) > prefixIndex+1 {
		return strings.Join(parts[:prefixIndex], "/") + "/" + strings.Join(parts[:prefixIndex+1], "/")
	} else {
		return path
	}
}*/
