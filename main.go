package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

func main() {
	fmt.Printf("\n\t\033[33;1m---===[ Merge Nessus Reports v1.0 ]===---\033[m\n\n")

	// Fetch Arguments
	dirPtr := flag.String("dir", "", "directory that contains existing .nessus files")
	outFilePtr := flag.String("out", "", "Filepath you want to export the merged report to")
	reportNamePtr := flag.String("title", "Merged Report", "The display name of merged report for the Nessus Web UI")
	flag.Parse()

	// Ensure that the user provided a directory argument.
	if *dirPtr == "" {
		fmt.Println(" \033[31;1m[!] ERROR:\033[m you must specify an input reports directory with --dir")
		flag.Usage()
		os.Exit(1)
	}

	// Ensure that the user provided a output argument.
	if *outFilePtr == "" {
		fmt.Println(" \033[31;1m[!] ERROR:\033[m you must specify an merged report output filename with --out")
		flag.Usage()
		os.Exit(1)
	}

	// Display the arguments in the output.
	fmt.Println(" [\033[34;1m*\033[m] Input Dir:", *dirPtr)
	fmt.Println(" [\033[34;1m*\033[m] Output Filename:", *outFilePtr)

	// Get a slice of the nessus files in the directory.
	nessus_files, err := filepath.Glob(*dirPtr + "/*.nessus")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Abort if no files were found.
	if len(nessus_files) == 0 {
		fmt.Println(" \033[31;1m[!] ERROR:\033[m", *dirPtr, "contains no Nessus files.")
		os.Exit(1)
	}

	// Declare vars for worker sync.
	var wg sync.WaitGroup
	var chproc = make(chan WorkerReport, len(nessus_files))
	var chstatus = make(chan JobStatus, len(nessus_files))
	var chfinal = make(chan NessusReport)

	// Start report parser and status printer workers.
	go worker_process_report(chproc, chstatus, chfinal)
	go worker_status_printer(nessus_files, chstatus, &wg)

	// Launch Report Parser Workers
	for _, file := range nessus_files {
		wg.Add(1)
		go worker_parse_files(file, chproc, chstatus, &wg)
	}

	// Wait for the reports to finish parsing and get to processing.
	// Once all of them are in queue for processing, we can close the
	// processing queue channel, which will signal to it we are ready
	// for the final report.
	wg.Wait()
	close(chproc)

	// Wait for the final report to arrive from the report processor worker.
	final_report := <-chfinal

	// Close the final channel since we got the report.
	close(chfinal)

	// Add one to the wait group, send a finalization message to the
	// status printer, and close the chstatus. Once it is done and the worker_status_printer()
	// shuts down, we can continue.
	wg.Add(1)
	chstatus <- JobStatus{"Finalize", ""}
	close(chstatus)
	wg.Wait()

	// Finally, write out the final report to a file.
	final_report.Report.Name = *reportNamePtr
	err = final_report.OutputReport(*outFilePtr)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println(" \033[32;1m[+] Done Son!\033[m")
}
