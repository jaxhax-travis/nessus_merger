package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"runtime"
)

type NessusReport struct {
	XMLName xml.Name `xml:"NessusClientData_v2"`
	Policy  Policy   `xml:"Policy"`
	Report  Report   `xml:"Report"`
}

// ParseNessusFile() will open the .nessus file specified by the xml_file parameter and
// unmarshal it into the NessusReport struct pointed to with the report parameter.
func ParseNessusFile(xml_file string, report *NessusReport) error {
	// Open our xmlFile
	xmlFile, err := os.Open(xml_file)
	// if we os.Open returns an error then handle it
	if err != nil {
		return err
	}
	defer xmlFile.Close()

	// read our opened xmlFile as a byte array.
	byteValue, _ := io.ReadAll(xmlFile)

	xml.Unmarshal(byteValue, &report)
	return nil
}

// OutputReport() will marshal the NessusReport struct back to XML and write it out
// to the filepath specified in the xml_file parameter.
func (report *NessusReport) OutputReport(xml_file string) error {
	fmt.Printf(" [\033[34;1m*\033[m] Writing %s report to \033[33;1m%s\033[m\n", report.Report.Name, xml_file)

	runtime.GC() // Run garbage before marshal
	out, err := xml.MarshalIndent(&report, "", "")
	if err != nil {
		return err
	}

	out = []byte(xml.Header + string(out)) // Attache the XML header

	runtime.GC() // Run garbage before write

	err = os.WriteFile(xml_file, out, 0644) // Write the contents to the file
	if err != nil {
		return err
	}

	return nil
}

type Policy struct {
	Content []byte `xml:",innerxml"`
}

type Report struct {
	Name       string       `xml:"name,attr"`
	Cm         string       `xml:"cm,attr"`
	ReportHost []ReportHost `xml:"ReportHost"`
}

// HasHostNyName() is a support function used by the worker_process_report() worker to determine
// if the host in the current report already exist in the final merged report it is building.
func (r Report) HasHostNyName(name string) bool {
	for idx := 0; idx < len(r.ReportHost); idx++ {
		if r.ReportHost[idx].Name == name {
			return true
		}
	}
	return false
}

// GetHostNyName() is a function to get a pointer to the ReportHost struct in
// the Report that matches a give hostname.
func (r Report) GetHostNyName(name string) *ReportHost {
	for idx := 0; idx < len(r.ReportHost); idx++ {
		if r.ReportHost[idx].Name == name {
			return &r.ReportHost[idx]
		}
	}
	return nil
}

type ReportHost struct {
	Name           string         `xml:"name,attr"`
	HostProperties HostProperties `xml:"HostProperties"`
	ReportItem     []ReportItem   `xml:"ReportItem"`
}

// AlreadyHasFinding() is a support function used by the worker_process_report() worker to determine
// if the finding already exist in the final merged report for the host being processed.
func (r ReportHost) AlreadyHasFinding(pluginID int, port uint16, protocol string) bool {
	for idx := 0; idx < len(r.ReportItem); idx++ {
		if r.ReportItem[idx].PluginID == pluginID && r.ReportItem[idx].Port == port && r.ReportItem[idx].Protocol == protocol {
			return true
		}
	}
	return false
}

type HostProperties struct {
	Content []byte `xml:",innerxml"`
}

type ReportItem struct {
	Port         uint16 `xml:"port,attr"`
	SvcName      string `xml:"svc_name,attr"`
	Protocol     string `xml:"protocol,attr"`
	Severity     string `xml:"severity,attr"`
	PluginID     int    `xml:"pluginID,attr"`
	PluginName   string `xml:"pluginName,attr"`
	PluginFamily string `xml:"pluginFamily,attr"`
	Content      []byte `xml:",innerxml"`
}
