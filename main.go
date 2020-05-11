package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/duladissa/go-subword-nmt/bpe"
)

const (
	//CodeFilePath ... Codes file path for BPE
	CodeFilePath = ""
	//TextWithoutBPEFilePath ... Sample text file path without BPE
	TextWithoutBPEFilePath = ""
	//TextWithBPEFilePath ... Sample output text file path with BPE
	TextWithBPEFilePath = ""
	//Query ... Query
	Query = "hello how are you?"
)

func main() {
	bpe, err := bpe.NewBPE(CodeFilePath, "")
	if err == nil {
		//Read from a file and writing to a file
		//readSampleTestAndProcessLine(bpe)

		response := bpe.ProcessLine(Query, 0)
		fmt.Println(response)
	} else {
		fmt.Println("Error ", err)
	}
}

func readSampleTestAndProcessLine(bpe *bpe.BPE) {
	bpeTest, _ := readAndProcessLines(bpe, TextWithoutBPEFilePath)
	writeLines(bpeTest, TextWithBPEFilePath)
}

func readAndProcessLines(bpe *bpe.BPE, path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, bpe.ProcessLine(scanner.Text(), 0))
	}
	return lines, scanner.Err()
}

func writeLines(lines []string, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	w := bufio.NewWriter(file)
	for _, line := range lines {
		fmt.Fprintln(w, line)
	}
	return w.Flush()
}
