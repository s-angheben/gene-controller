package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"
)

type Lgn_data struct {
	to   string
	from string
}

func Scan_until_comma(data []byte, atEOF bool) (advance int, token []byte, err error) {
	// Skip leading commas.
	start := 0
	for width := 0; start < len(data); start += width {
		var r rune
		r, width = utf8.DecodeRune(data[start:])
		if r != ',' {
			break
		}
	}
	// Scan until comma, marking end of word.
	for width, i := 0, start; i < len(data); i += width {
		var r rune
		r, width = utf8.DecodeRune(data[i:])
		if r == ',' {
			return i + width, data[start:i], nil
		}
	}
	// If we're at EOF, we have a final, non-empty, non-terminated word. Return it.
	if atEOF && len(data) > start {
		return len(data), data[start:], nil
	}
	// Request more data.
	return start, nil, nil
}

func get_lgn_probes(lgn_path string, obs_path string) ([]uint64, uint64) {
	lgn_file, err := os.Open(lgn_path)
	if err != nil {
		panic(err.Error())
	}
	lgn_scanner := bufio.NewScanner(lgn_file)
	lgn_scanner.Split(bufio.ScanLines)

	lgn_map := make(map[string]string)

	lgn_scanner.Scan() // skip the first line
	for lgn_scanner.Scan() {
		s := strings.Split(lgn_scanner.Text(), ",")
		if len(s) >= 2 {
			lgn_map[s[0]] = s[1]
		}
	}
	fmt.Println(lgn_map)

	obs_file, err := os.Open(obs_path)
	if err != nil {
		panic(err.Error())
	}
	obs_scanner := bufio.NewScanner(obs_file)
	obs_scanner.Split(bufio.ScanLines)

	var lgn_indexes []uint64
	var obs_file_row uint64 = 0
	obs_scanner.Scan()
	for obs_scanner.Scan() {
		obs_file_row += 1
		row_scanner := bufio.NewScanner(strings.NewReader(obs_scanner.Text()))
		row_scanner.Split(Scan_until_comma)
		row_scanner.Scan() // take the first element
		if _, ok := lgn_map[row_scanner.Text()]; ok {
			lgn_indexes = append(lgn_indexes, obs_file_row)
		}
	}
	fmt.Println(lgn_indexes, obs_file_row)

	lgn_file.Close()
	obs_file.Close()

	return lgn_indexes, obs_file_row
}

func main() {
	get_lgn_probes("/home/boincadm/projects/test/gene_input_chaos/hs/T096662-CRYZ.lgn", "/home/boincadm/projects/test/gene_input_chaos/hs/hgnc_data_mat.csv")
}
