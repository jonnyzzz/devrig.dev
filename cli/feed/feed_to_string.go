package feed

import "fmt"

func logFeedItem(item feedEntry) {
	fmt.Printf("Product: %s\n", item.Name)
	fmt.Printf("  Version: %s (Build: %s)\n", item.Version, item.Build)
	fmt.Printf("  Released: %s\n", item.Released)

	if item.Package != nil {
		pkg := item.Package
		fmt.Printf("  feedItemPackage:\n")
		fmt.Printf("    OS: %s\n", pkg.OS)
		fmt.Printf("    Type: %s\n", pkg.Type)
		fmt.Printf("    Size: %d mb\n", pkg.Size/1024/1024)

		if len(pkg.Checksums) > 0 {
			fmt.Printf("    Checksums:\n")
			for _, checksum := range pkg.Checksums {
				fmt.Printf("      %s: %s\n", checksum.Algorithm, checksum.Value)
			}
		}

		if pkg.Requirements.CPUArch.Equals != "" {
			fmt.Printf("    CPU Architecture: %s\n", pkg.Requirements.CPUArch.Equals)
		}

		fmt.Printf("    URL: %s\n", pkg.URL)
	}
	fmt.Println()
}
