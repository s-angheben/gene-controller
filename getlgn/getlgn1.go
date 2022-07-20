package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

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
		s := strings.Split(obs_scanner.Text(), ",")
		if _, ok := lgn_map[s[0]]; ok {
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
