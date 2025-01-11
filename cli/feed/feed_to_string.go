package feed

import "fmt"

func (item *feedEntry) ToString() string {
	var result string

	result += fmt.Sprintf("Product: %s\n", item.Name)
	result += fmt.Sprintf("  Version: %s (Build: %s)\n", item.Version, item.Build)
	result += fmt.Sprintf("  Released: %s\n", item.Released)

	if item.Package != nil {
		pkg := item.Package
		result += "  feedItemPackage:\n"
		result += fmt.Sprintf("	OS: %s\n", pkg.OS)
		result += fmt.Sprintf("	Type: %s\n", pkg.Type)
		result += fmt.Sprintf("	Size: %d mb\n", pkg.Size/1024/1024)

		if len(pkg.Checksums) > 0 {
			result += "	Checksums:\n"
			for _, checksum := range pkg.Checksums {
				result += fmt.Sprintf("	  %s: %s\n", checksum.Algorithm, checksum.Value)
			}
		}

		if pkg.Requirements.CPUArch.Equals != "" {
			result += fmt.Sprintf("	CPU Architecture: %s\n", pkg.Requirements.CPUArch.Equals)
		}

		result += fmt.Sprintf("	URL: %s\n", pkg.URL)
	}

	return result
}

func logFeedItem(item feedEntry) {
	text := item.ToString()
	fmt.Println(text + "\n")
}
