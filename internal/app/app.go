package app

import (
	"bufio"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/bjesus/pipet/common"
	"github.com/bjesus/pipet/parsers"
	"github.com/google/shlex"
)

func ParseSpecFile(e *common.PipetApp, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var currentBlock *common.Block

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" {
			if currentBlock != nil {
				e.Blocks = append(e.Blocks, *currentBlock)
				currentBlock = nil
			}
			continue
		}

		if strings.HasPrefix(line, "//") {
			continue
		}
		if currentBlock == nil {
			if strings.HasPrefix(line, "curl ") {
				currentBlock = &common.Block{Type: "curl", Command: line}
			} else if strings.HasPrefix(line, "playwright ") {
				currentBlock = &common.Block{Type: "playwright", Command: line}
			} else {
				return fmt.Errorf("invalid block start: %s", line)
			}
		} else {
			if strings.HasPrefix(line, "> ") {

				currentBlock.NextPage = strings.TrimPrefix(line, ">")
			} else {
				currentBlock.Queries = append(currentBlock.Queries, line)
			}
		}
	}

	log.Println("Found block", currentBlock)
	if currentBlock != nil {
		e.Blocks = append(e.Blocks, *currentBlock)
	}

	return scanner.Err()
}

func ExecuteBlocks(e *common.PipetApp) error {
	for _, block := range e.Blocks {
		var data interface{}
		var err error
		var nextPageURL string

		for page := 0; page < e.MaxPages; page++ {
			if block.Type == "curl" {
				data, nextPageURL, err = parsers.ExecuteCurlBlock(block)
			} else if block.Type == "playwright" {
				data, err = parsers.ExecutePlaywrightBlock(block)
			}

			if err != nil {
				return err
			}

			e.Data = append(e.Data, data)

			if nextPageURL == "" {
				break
			}
			var parts []string
			switch cmd := block.Command.(type) {
			case string:
				parts, _ = shlex.Split(cmd)
			case []string:
				parts = cmd
			default:
			}

			for i, u := range parts {
				if len(u) >= 4 && u[:4] == "http" {
					parts[i] = concatenateURLs(parts[i], nextPageURL)
					break
				}
			}

			block.Command = parts
		}
	}

	return nil
}

func concatenateURLs(base, ref string) string {
	baseURL, err := url.Parse(base)
	if err != nil {
		panic(err)
	}
	refURL, err := url.Parse(ref)
	if err != nil {
		panic(err)
	}

	// Resolve reference URL relative to the base URL
	fullURL := baseURL.ResolveReference(refURL)

	return fullURL.String()
}
