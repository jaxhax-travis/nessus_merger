package main

import (
	"fmt"
	"sync"
	"time"
)

// WorkerReport is a data struct that stores the name of the .nessus file and the
// unmarshal NessusReport data struct.  This data struct is for passing data through
// the chproc channel between the worker_parse_files() workers to the worker_process_report()
// worker.
type WorkerReport struct {
	Filename string
	Report   NessusReport
}

// This function is intended to be used as a goroutine, one per Nessus report. It will open the file
// and unmarshal the XML into a NessusReport data struct. Once complete, it will send it over to the
// report processor goroutine worker_process_report(), via the chproc channel.
func worker_parse_files(filename string, chproc chan<- WorkerReport, chstatus chan<- JobStatus, wg *sync.WaitGroup) {
	defer wg.Done()

	// Create a new WorkerReport struct
	var report WorkerReport
	report.Filename = filename

	// Let the user know we are working on it.
	chstatus <- JobStatus{filename, "\033[35;1mParsing Nessus File\033[m"}

	// Unmarshal of Nessus XML file to NessusReport data struct
	err := ParseNessusFile(filename, &report.Report)
	if err != nil {
		msg := fmt.Sprintf("\033[31;1mERROR:\033[m %s", err)
		chstatus <- JobStatus{filename, msg}
		return
	}

	// Let the user know the report has been unmarshalled and waiting
	// to be processed by worker_process_report().
	chstatus <- JobStatus{filename, "\033[34;1mAwaiting Merge...\033[m"}

	// Send the WorkerReport struct to the job queue for worker_process_report().
	chproc <- report
}

// This function is intended to be used as a goroutine, one total running in the background.  This takes the
// WorkerReports as they trickle in from the worker_parse_files() workers and merges them into a final report.
// Once the final report is done, its sent back to main() via the chfinal channel.
func worker_process_report(chproc <-chan WorkerReport, chstatus chan<- JobStatus, chfinal chan<- NessusReport) {
	var final NessusReport = NessusReport{}
	var first_report bool = true
	for {
		report, more := <-chproc
		if more {
			// If this is the first report, we will just copy it over and use
			// the whole thing to save time.
			if first_report {
				final = report.Report
				first_report = false
				chstatus <- JobStatus{report.Filename, "\033[32;1mComplete\033[m"}
				continue
			}

			// Review each host of the current report to see if it
			// needs to be added to the final report.
			total_hosts := len(report.Report.Report.ReportHost)
			last_print_ts := time.Now().Add(-time.Millisecond * 250)
			for idx, host := range report.Report.Report.ReportHost {
				// Throttle down the messages sent to the status printer
				if time.Now().Sub(last_print_ts) >= time.Millisecond*250 {
					// Send a status message to the printer.
					progress := (float64(idx) / float64(total_hosts)) * 100.00
					status := fmt.Sprintf("\033[33;1mProcessing - %3.2f%% [Hosts: %d/%d]\033[m", progress, idx, total_hosts)
					chstatus <- JobStatus{report.Filename, status}
					last_print_ts = time.Now()
				}

				// if the host isn't in the final report yet, move the full ReportHost over
				if !final.Report.HasHostNyName(host.Name) {
					final.Report.ReportHost = append(final.Report.ReportHost, host)
					continue
				}

				// Otherwise, focus on just appending new report entries to it.
				final_host := final.Report.GetHostNyName(host.Name)
				for _, finding := range host.ReportItem {
					if !final_host.AlreadyHasFinding(finding.PluginID, finding.Port, finding.Protocol) {
						final_host.ReportItem = append(final_host.ReportItem, finding)
					}
				}
			}
			// Notify user that the report has been processed completely.
			chstatus <- JobStatus{report.Filename, "\033[32;1mComplete\033[m"}
		} else {
			// The chproc channel was closed by main() and there are no more jobs
			// coming in.  Send the final report to main().
			chfinal <- final
			return
		}
	}
}
