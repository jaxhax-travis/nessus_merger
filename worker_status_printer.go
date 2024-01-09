package main

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// JobStatus is a data struct for channel messages intended for the worker_status_printer() worker.
type JobStatus struct {
	JobName string
	Status  string
}

// JobStatuses is a data struct for tracking information needed by worker_status_printer() to track job status
// cleanly format the output data.
type JobStatuses struct {
	Job        []JobStatus
	Len        int
	MaxNameLen int
	Overwrite  bool
	Finalize   bool
}

// UpdateStatus() will take a JobStatus in and replace the existing status
// for the job in the JobStatuses.Job slice with the new one.
func (jobs *JobStatuses) UpdateStatus(job_status JobStatus) {
	for idx, job := range jobs.Job {
		if job.JobName == job_status.JobName {
			jobs.Job[idx].Status = job_status.Status
		}
	}
}


func (jobs JobStatuses) PrintStatus() {
	// If Overwrite is true, we will overwrite the original status messages in console
	// via an ansi escape sequence that moves the cursor up [jobs.Len] lines.
	if jobs.Overwrite {
		fmt.Printf("\033[%dA", jobs.Len)
	}

	// Next, print the status of each job in the JobStatuses.Job slice.
	for _, job := range jobs.Job {
		job_name := tidyName(job.JobName)
		fmt.Printf(" \033[33m%*s\033[m : %s\033[K\n", jobs.MaxNameLen, job_name, job.Status)
	}

	// Lastly, if JobStatuses.Finalize is set to true, let the user know the report is being
	// written out.
	if jobs.Finalize {
		fmt.Println(" [\033[34;1m*\033[m] Writing Merged Nessus Report...")
	}
}

// tidyName() takes the report filename and extracts just the filename without the extension.
// This function is here so the report name will be displayed with less clutter.
func tidyName(jobname string) string {
	return strings.TrimSuffix(filepath.Base(jobname), filepath.Ext(jobname))
}

// This function is intended to be used as a goroutine, one total running in the background.  This takes the
// JobStatus as they trickle in from various sources and will update the JobStatuses stuct and refresh the
// status output.
func worker_status_printer(report_files []string, chstatus <-chan JobStatus, wg *sync.WaitGroup) {
	var jobs JobStatuses
	// Initalize the JobStatuses data.
	jobs.MaxNameLen = 0
	jobs.Overwrite = false

	// Loop through the report files slice and gather information to track status.
	for _, filename := range report_files {
		// Add the report file as a JobStatus entry to the Job slice.
		jobs.Job = append(jobs.Job, JobStatus{filename, ""})

		// For display purposes, remove the extension and directory path information
		// from the report name.
		base_name := tidyName(filename)

		// Check if this is the longest name.  If it is, update the MaxNameLen value.
		// This value is used for padding to ensure the statuses are displayed at the
		// same offset.
		if len(base_name) >= jobs.MaxNameLen {
			jobs.MaxNameLen = len(base_name)
		}
	}

	// Sort the slice of jobs so they are displayed in order.
	sort.Slice(jobs.Job, func(i, j int) bool {
		return tidyName(jobs.Job[i].JobName) < tidyName(jobs.Job[j].JobName)
	})

	// update the Job slice length value so we can lookup rather than getting the
	// length repeatedly.
	jobs.Len = len(jobs.Job)

	// Print the initial status message and set the Overwrite value to true.
	// This value set to true will cause PrintStatus() to overwrite the old status lines
	// in the console output.
	fmt.Println(" [\033[34;1m*\033[m] Merging\033[32m", jobs.Len, "\033[mFiles:")
	jobs.PrintStatus()
	jobs.Overwrite = true

	// Status listener loop
	for {
		status, more := <-chstatus
		if more {
			if status.JobName == "Finalize" {
				jobs.Finalize = true
			}
			jobs.UpdateStatus(status)
			jobs.PrintStatus()
		} else {
			wg.Done()
			return
		}
	}
}
