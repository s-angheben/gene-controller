package main

import (
	"bufio"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

        "github.com/antchfx/xmlquery"
	_ "github.com/go-sql-driver/mysql"
)

// DB STRUCTS

type Wg_params struct {
	cushion            uint
	replication_factor uint
	max_time_sleep     uint
	num_pc_wu          uint
	deadline           uint
	out_template       string
	executions_path    string
	results_path       string
}

type Pcim_to_execute struct {
	pcim_id    uint64
	organism   string
	pcim_name  string
	lgn_path   string
	alpha      float64
	iterations uint
	tile_size  uint
	npc        uint
	cutoff     uint
	priority   uint
}

type Benchmark struct {
	id          int
	exp_id      int
	pc_tsize    int
	pc_alpha    float32
	app_name    string
	app_version string
	pc_time     float32
	host_name   string
	host_flops  float64
	host_iops   float64
}

//

type Exp struct {
	path string
	name string
}

type Pcim_estimate struct {
	exp_id     uint64
	pc_time    float32
	host_flops float32
}

// USEFULL FUNCTIONS
func file_exists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// DB FUNCTIONS

func init_db() *sql.DB {
	db, err := sql.Open("mysql", "root:example@tcp(127.0.0.1:3306)/gene")
	if err != nil {
		panic(err.Error())
	}
	return db
}

func load_pcim_to_execute_number(db *sql.DB) uint {
	var pcim_number uint
	err := db.QueryRow(`SELECT COUNT(pcim_id)
                        FROM pcim_to_execute`).Scan(
		&pcim_number)
	if err != nil {
		panic(err.Error())
	}
	return pcim_number
}

func load_gen_param(db *sql.DB) *Wg_params {
	var gen_param Wg_params
	err := db.QueryRow(`SELECT cushion,replication_factor,
                               max_time_sleep,num_pc_wu,
                               deadline,out_template,
			       executions_path,results_path
		        FROM wg_params`).Scan(
		&gen_param.cushion, &gen_param.replication_factor,
		&gen_param.max_time_sleep, &gen_param.num_pc_wu,
		&gen_param.deadline, &gen_param.out_template,
		&gen_param.executions_path, &gen_param.results_path)
	if err != nil {
		panic(err.Error())
	}
	return &gen_param
}

func load_pcim_param(db *sql.DB) *Pcim_to_execute {
	var pcim_param Pcim_to_execute
	// err := db.QueryRow(`SELECT pcim_id,organism,pcim_name,
	//                                  lgn_path,alpha,iterations,
	//    			      tile_size,npc,
	//    			      cutoff,priority
	//    		        FROM pcim_to_execute
	//    		        ORDER BY priority DESC
	//    		        LIMIT 1`).Scan(
				err := db.QueryRow(`SELECT pcim_id,organism,pcim_name,
		                      lgn_path,alpha,iterations,
				      tile_size,npc,
				      cutoff,priority
			        FROM pcim
			        WHERE pcim_id = ? `, 210047).Scan(
		&pcim_param.pcim_id, &pcim_param.organism, &pcim_param.pcim_name,
		&pcim_param.lgn_path, &pcim_param.alpha, &pcim_param.iterations,
		&pcim_param.tile_size, &pcim_param.npc,
		&pcim_param.cutoff, &pcim_param.priority)
	if err != nil {
		panic(err.Error())
	}
	if !file_exists(pcim_param.lgn_path) {
		panic("file " + pcim_param.lgn_path + " doesn't exists")
	}
	return &pcim_param
}

func load_exp_id(db *sql.DB, pcim_id uint64) uint64 {
	var exp_id uint64
	err := db.QueryRow(`SELECT exp_id
                        FROM pcim_experiments
     		        WHERE pcim_id = ?`,
		pcim_id).Scan(
		&exp_id)
	if err != nil {
		panic(err.Error())
	}
	return exp_id
}

func load_pcim_estimate(db *sql.DB, exp_id uint64, tile_size uint, pc_alpha float64, app_name string, version string) *Pcim_estimate {
	var pcim_est Pcim_estimate
	pcim_est.exp_id = exp_id
	err := db.QueryRow(`SELECT pc_time, host_flops
                       FROM benchmark
             	       WHERE exp_id = ? AND pc_tsize = ? AND
             	             pc_alpha = ? AND app_name = ? AND
             	             app_version LIKE ?`,
		exp_id, tile_size,
		pc_alpha, app_name,
		version).Scan(
		&pcim_est.pc_time, &pcim_est.host_flops)
	if err != nil {
		panic(err.Error())
	}
	return &pcim_est
}

// the error field is used as error in the update query
func load_exp_file_path(db *sql.DB, exp_id uint64) ([]Exp, error) { //Exp is already a reference since it's a slice
	var exps []Exp
	var rows_number uint = 0
	rows, err := db.Query(`SELECT exp_path, exp_name
                          FROM experiments
			  WHERE exp_id = ?`,
		exp_id)
	if err != nil {
		panic(err.Error())
	}
	defer rows.Close()
	for rows.Next() {
		var exp Exp
		err := rows.Scan(&exp.path, &exp.name)
		if err != nil {
			panic(err.Error())
		}
		if !file_exists(exp.path) {
			return nil, errors.New("file " + exp.path + " doesn't exists")
		}
		exps = append(exps, exp)
		rows_number += 1
	}
	err = rows.Err()
	if err != nil {
		panic(err.Error())
	}
	if rows_number == 0 {
		return nil, errors.New("No experiments found in database")
	}
	return exps, nil
}

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

func get_lgn_probes(lgn_path string, obs_path string) ([]uint, uint) {
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

	obs_file, err := os.Open(obs_path)
	if err != nil {
		panic(err.Error())
	}
	obs_scanner := bufio.NewScanner(obs_file)
	obs_scanner.Split(bufio.ScanLines)

	var lgn_indexes []uint
	var obs_file_row uint = 0
	obs_scanner.Scan()
	for obs_scanner.Scan() {
		obs_file_row += 1
		col_scanner := bufio.NewScanner(strings.NewReader(obs_scanner.Text()))
		col_scanner.Split(Scan_until_comma)
		col_scanner.Scan() // take the first element
		if _, ok := lgn_map[col_scanner.Text()]; ok {
			lgn_indexes = append(lgn_indexes, obs_file_row)
		}
	}

	lgn_file.Close()
	obs_file.Close()

	return lgn_indexes, obs_file_row
}

func count_colls(obs_path string) (int) {
	obs_file, err := os.Open(obs_path)
	if err != nil {
		panic(err.Error())
	}
	obs_scanner := bufio.NewScanner(obs_file)
	obs_scanner.Split(bufio.ScanLines)

	obs_scanner.Scan()
	s := strings.Split(obs_scanner.Text(), ",")
	obs_file.Close()
	return len(s)
}

func save_parameters(path string, cols int, wg_param *Wg_params, pcim_param *Pcim_to_execute, pcim_estimate *Pcim_estimate) {
	data_str := fmt.Sprint("PC-IM: ", pcim_param.pcim_name, "\n",
		"LGN: ", pcim_param.lgn_path, "\n",
		"alpha: ",  pcim_param.alpha, "\n",
		"iterations: ", pcim_param.iterations, "\n",
		"tile size: ", pcim_param.tile_size, "\n",
		"PC x WU: ", pcim_param.npc, "\n",
		"deadline: ", wg_param.deadline, "\n",
		"num cols: ", cols, "\n",
		"WU FLOP: ", pcim_estimate.pc_time*pcim_estimate.host_flops*1.6, "\n")
	data := []byte(data_str)
	os.WriteFile(path + "parameters.txt", data, 0644)
}

// CHECK CONFIG FILE
func check_disabled (config_file string) bool {
        disabled := true
        file, err := os.Open("config.xml")
        if err != nil {
                return true
        }
        bufio.NewReader(file)
        doc, err := xmlquery.Parse(file)
        if err != nil {
                return true
        }

        root := xmlquery.FindOne(doc, "//boinc")
        for i, n := range xmlquery.Find(doc, "//daemons/daemon/cmd") {
                if strings.Split(n.InnerText(), " ")[0] == "gene_work_generator" {
                        if root.SelectElement("//daemons/daemon[" + strconv.Itoa(i) + "]/disabled").InnerText() == "0" {
                                disabled = false
                        }
                }
        }
        return disabled
}


func main() {
	db := init_db()
	defer db.Close()

	disabled := check_disabled("config.xml")
	if disabled {
		os.Exit(1)
	}

	// get_queue_com := exec.Command(../bin/gene_get_queue)
	// val, err := get_queue_com.Output()


	pcim_to_execute_number := load_pcim_to_execute_number(db)
	fmt.Println("pcim to execute: ", pcim_to_execute_number)

	gen_param := load_gen_param(db)
	fmt.Println("working generator parameters: ", gen_param)

	pcim_param := load_pcim_param(db)
	fmt.Println("pcim parameters: ", pcim_param)

	exp_id := load_exp_id(db, pcim_param.pcim_id)
	fmt.Println("exp_id: ", exp_id)

	pcim_estimate := load_pcim_estimate(db, exp_id, pcim_param.tile_size, pcim_param.alpha, "gene_pcim", "1.00")
	fmt.Println("pc_time: ", pcim_estimate.pc_time, ", host_flops: ", pcim_estimate.host_flops)
	fmt.Println("time: ", pcim_estimate.pc_time*pcim_estimate.host_flops*1.6)

	//TODO
	exps, err := load_exp_file_path(db, exp_id)
	if err != nil {
		// save the error in the database
		db.Exec(`UPDATE pcim
	         SET error = 1, email_sent = 0
     		 WHERE pcim_id = ?`, pcim_param.pcim_id)
		panic(err.Error())
	}
	fmt.Printf("%v\n", exps)

	// RUN THE ALGORITHM
	lgn, size := get_lgn_probes(pcim_param.lgn_path, exps[0].path)
	fmt.Println(lgn, size)

	fmt.Println("colls number:", count_colls(exps[0].path))
	tmp := math.Ceil((float64(size-uint(len(lgn))) / float64(pcim_param.tile_size-uint(len(lgn)))))
	number_wu := math.Ceil(tmp * float64(pcim_param.iterations) / float64(pcim_param.npc))
	fmt.Println("wu:", math.Ceil(tmp * float64(pcim_param.iterations) / float64(pcim_param.npc)))
	fmt.Println(size, len(lgn), pcim_param.tile_size)

	path := "/mnt/ramdisk/" + strconv.FormatUint(pcim_param.pcim_id, 10) + "_" + pcim_param.organism + "/"
	os.Mkdir(path, os.ModePerm)
	save_parameters(path, count_colls(exps[0].path), gen_param, pcim_param, pcim_estimate)

	/////////////
	var lgn_string, size_string, tile_size_string, iterations_string string


	iterations_string = strconv.FormatUint(uint64(pcim_param.iterations), 10)
	size_string = strconv.FormatUint(uint64(size), 10)
	tile_size_string = strconv.FormatUint(uint64(pcim_param.tile_size), 10)
	npc_string := strconv.FormatUint(uint64(pcim_param.npc), 10)
	lgn_string = ""
	for _, elem := range lgn {
		lgn_string += strconv.FormatUint(uint64(elem), 10) + " "
	}
	fmt.Println(size_string, lgn_string)

	fmt.Println("command: ", "gene", "--lgn", lgn_string, "-s", size_string, "-t", tile_size_string, "-i", iterations_string,
		"--path", path,  "-n", npc_string,
	)

	cmd := exec.Command("gene", "--lgn", lgn_string, "-s", size_string, "-t", tile_size_string, "-i", iterations_string,
		"--path", path,  "-n", npc_string,
	)
	// cmd := exec.Command("ls")
	err = cmd.Start()
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	db.Exec(`UPDATE pcim
	         SET in_execution = 1,
                 number_wus = ?, time_creation = NOW()
     		 WHERE pcim_id = ?`, number_wu, pcim_param.pcim_id)
	fmt.Printf("%v tile_creation started with the command gene\n", time.Now().Unix())
	go func() {
		err = cmd.Wait()
		fmt.Printf("Command finished with error: %v\n", err)
		fmt.Printf("%v tile_creation finished\n", time.Now().Unix())

		// sticky_f, _ := os.Create(path + "/in_sticky.txt")
		// sticky_f.WriteString(exps[0].path)
		// sticky_f.Close()

		// CALL BOINC
		// exec.Command('../bin/gene_make_one_wu gene_pcim 4', pcim_name, app_name, out_template, repr(time_estimation), in_sticky_path, repr(len(exps_list)), in_nosticky_path, repr(1), repr(replication_factor), repr(deadline), repr(alpha), OUTPUT, repr(col_number), repr(cut_results))

	}()


	for {
	}
}
