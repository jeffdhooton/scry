package coverage

import (
	"encoding/xml"
	"os"
	"strconv"
)

// parseClover parses Clover XML coverage (PHPUnit --coverage-clover output).
//
// Structure:
//
//	<coverage>
//	  <project>
//	    <file name="/abs/path/to/file.php">
//	      <line num="10" type="stmt" count="1"/>
//	      <line num="11" type="stmt" count="0"/>
//	    </file>
//	  </project>
//	</coverage>
func parseClover(path, repoPath string) ([]CoveredRange, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var doc cloverCoverage
	if err := xml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}

	var ranges []CoveredRange
	for _, proj := range doc.Projects {
		for _, file := range proj.Files {
			relPath := toRelPath(file.Name, repoPath)
			for _, line := range file.Lines {
				if line.Type != "stmt" && line.Type != "method" {
					continue
				}
				num, err := strconv.Atoi(line.Num)
				if err != nil || num <= 0 {
					continue
				}
				count, _ := strconv.Atoi(line.Count)
				ranges = append(ranges, CoveredRange{
					File:    relPath,
					Line:    num,
					EndLine: num, // Clover is line-level, no range info
					Count:   count,
				})
			}
		}
	}
	return ranges, nil
}

type cloverCoverage struct {
	XMLName  xml.Name       `xml:"coverage"`
	Projects []cloverProject `xml:"project"`
}

type cloverProject struct {
	Files []cloverFile `xml:"file"`
}

type cloverFile struct {
	Name  string       `xml:"name,attr"`
	Lines []cloverLine `xml:"line"`
}

type cloverLine struct {
	Num   string `xml:"num,attr"`
	Type  string `xml:"type,attr"`
	Count string `xml:"count,attr"`
}

func init() {
	registerParser("clover", func(path, repoPath string) ([]CoveredRange, error) {
		return parseClover(path, repoPath)
	})
}
