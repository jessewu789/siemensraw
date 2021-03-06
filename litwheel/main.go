package main

// not highly efficient method to parse raidtool information
// creates raidtooltmp.txt, raidtool.txt files in the local directory
import (
	"encoding/csv"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"regexp"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

func main() {
	// command line check >>
	if len(os.Args) < 2 {
		fmt.Println("Example usage: \n \n raidtoolsignatures HASHFILE.txt \n" +
			"\n HASHFILE.txt: REQUIRED - can be an existing list of hashes, or this tool will create a new file with the given name. ")
		//"\n all : OPTIONAL- will force transfer of all data on the RAID, otherwise will check Performing Physician field for the following format:" +
		//" \"PERF PHYS NAME, [A-Z]{4,5}[0-9]{4,6}-[A-Z0-9]{4,10} \" (i.e. NHLBI1234-A0001)- where the comma is the separator key")
		os.Exit(0)
	}
	// command line check <<

	// check if hash record exists >>
	_, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Printf("This file does not exist, creating %v\n", os.Args[1])
		_, err = os.Create(os.Args[1])
		if err != nil {
			panic(err)
		}
	}
	// check if hash record exists <<

	// raidtool dump >>
	// debug //	fmt.Println("Raidtool dump") // debug //
	cmd := exec.Command("cmd.exe", "/C", "raidtool -d -a mars -p 8010 > raidtool.txt")
	// offline debug // cmd := exec.Command("cmd.exe", "/C", "RR_rt_print.exe > rt_temp.txt") // offline debug //
	//stdout, err := cmd.Output()
	_, err = cmd.Output()
	if err != nil {
		log.Fatal(err)
	}
	// raidtool dump <<

	// load raidtool dump >>
	// debug //	fmt.Println("Raidtool read") // debug //
	rtFile, err2 := ioutil.ReadFile("raidtool.txt")
	if err2 != nil {
		log.Fatal(err2)
	}
	// load raidtool dump <<

	// raidtool header print >>
	rt_string := string(rtFile[:])
	idx := strings.Index(rt_string, "FileID")
	rt_head := rt_string[:idx]
	headSlice := strings.Split(rt_head, " ")
	numFiles, _ := strconv.Atoi(headSlice[35]) // empirically consistent
	fileIDs := make([]string, numFiles+20)     // padding to avoid 'panic'
	fmt.Println("fileID size", len(fileIDs), "rt_head: \n", rt_head)
	// raidtool header print <<

	// Attempt to find measurement IDs using csv (tab delimiting doesn't quite work)
	idx = strings.Index(rt_string, "(fileID)")
	rt_body := rt_string[idx+len("(fileID)"):]
	r := csv.NewReader(strings.NewReader(rt_body))
	r.Comma = '\t' // ? is this reduntant?

	// loop through raidtool dump >>
	 //debug_tick := 0
	for {
		// debug //		fmt.Println("Reading CSV") // debug //
/*		debug_tick += 1
		if debug_tick > 10 {
			break
		} // limit how much of the RAID is processed for testing
*/
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		reg, err2 := regexp.Compile("[^0-9]+")
		if err2 != nil {
			log.Fatal(err2)
		}

		a := record[0]
		// debug // fmt.Println(a) // debug //
		if len(record[0]) < 100 { // end of file catch
			break
		}

		// determine filename & whether file is a retrorecon >>
		b := strings.SplitAfterN(a, " ", 500) // empirical

		c := "a"
		RRflag := 0
		e := 0
		i := 0
		xf := 0
		fileID := "und" // undefined
		MeasID := "und"
		fileNameStr := "und"
		dateStr := "und"
		timeStr := "und"

		for e < 9 { // empirical

			c = b[i]

			c = strings.Replace(c, " ", "", -1)

			if len(c) > 0 { // find text

				// debug // fmt.Println(c)
				e += 1

				if e == 1 {
					fileID = c
				} else if e == 2 {
					if len(c) > 5 { // retrorecon jobs have 7-digit FID's, no need to download these duplicates.
						fmt.Println("retrorecon")
						RRflag = 1
					} else {
						MeasID = strings.Repeat("0", 5-len(c)) + c
					}
				} else if e == 3 {
					fileNameStr = c
					if len(c) > 2 {
						if c[0:3] == "Adj" {
							RRflag = 1 // borrowing retrorecon flag to not copy adjustment scans
							fmt.Println("adj")
						}
					}
				} else if e > 3 && e < 7 {
					// sift through possible spaces in the filename. For simplicity, spaces are replaced with underscores.
					if c != "xxxxxx" && xf == 0 {
						e -= 1
						fileNameStr = fileNameStr + "_" + c
					} else if c == "xxxxxx" && xf == 0 {
						xf = 1
					}
				} else if e == 8 {
					// extract date in to YYYYMMDD format
					date1 := c
					dateStr = date1[6:10] + date1[3:5] + date1[0:2]

				} else if e == 9 {
					// extract creation time-stamp and remove colons for saving
					time1 := c
					timeStr = reg.ReplaceAllString(time1, "")
				}
			} else {
			}
			i += 1
		}

		// target format: meas_MID00000_FID00000_NAME.dat
		fileNameStr = dateStr + "_" + timeStr + "_" + "meas_" + "MID" + MeasID + "_FID" + strings.Repeat("0", 5-len(fileID)) + fileID + "_" + fileNameStr + ".dat"
	// determine filename <<

		if RRflag == 0 {
			// Suitable for transfer - now check if hash exists locally >>

			// debug //fmt.Println("raidtool -h "+fileID+" -o raidtooltmp.txt -a mars -p 8010") // offline debug //
			cmd := exec.Command("cmd.exe", "/C", "raidtool -h "+fileID+" -o raidtooltmp.txt -a mars -p 8010")
			_, err = cmd.Output()
			if err != nil {
				log.Fatal(err)
			}

			cmd = exec.Command("cmd.exe", "/C", "hdrsignature raidtooltmp.txt")
			stdout, err := cmd.Output()
			if err != nil {
				panic(err)
			}
			// offline debug // fmt.Println("cmd.exe", "/C", "hdrsignature raidtooltmp.txt") // offline debug //

			hdrHash := string(stdout[:])

			// check if hash exists >>

			read, err := ioutil.ReadFile(os.Args[1])
			if strings.Contains(string(read), hdrHash) {
				// debug //
				fmt.Println("file ID " + fileID + " : Hash exists, no need to transfer") // debug //

			} else {

				fmt.Println("file ID " + fileID + " : No hash, transferring and appending to log. ***")
				// @ LIT wheel - Data transfer >>

				// download data : -D > dependent measurements
				// debug // fmt.Printf("raidtool -m " + fileID + " -o " + fileNameStr + " -a mars -p 8010 -D \n")

				cmd = exec.Command("cmd.exe", "/C", "raidtool -m "+fileID+" -o "+fileNameStr+" -a mars -p 8010 -D")

				_, err = cmd.Output()
				if err != nil {
					log.Fatal(err)
				}

				// transfer data & remove host copy
				// debug // fmt.Printf("scp " + fileNameStr + " user@host:/folder/ \n")
				cmd = exec.Command("cmd.exe", "/C", "scp "+fileNameStr+" user@host:/folder/")

				_, err = cmd.Output()
				if err != nil {
					log.Fatal(err)
				}

				// debug //
				//		fmt.Printf("rm " + fileNameStr + " \n")
				cmd = exec.Command("cmd.exe", "/C", "rm "+fileNameStr)

				_, err = cmd.Output()
				if err != nil {
					log.Fatal(err)
				}
				// @ LIT wheel - Data transfer <<

				// append hash >>

				f, err := os.OpenFile(os.Args[1], os.O_APPEND, 0660)
				if err != nil {
					panic(err)
				}

				// debug //				n3, err := f.WriteString(hdrHash) // debug //
				_, err = f.WriteString(hdrHash)
				if err != nil {
					panic(err)
				}
				// debug //		fmt.Printf("wrote %d bytes\n", n3) // debug //
				f.Sync()

				// append hash <<

			} // check if hash exists <<
		}
	} // loop through raidtool dump <<

}
