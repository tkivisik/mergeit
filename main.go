package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"os"
	"time"
)

const (
	NEWINSTALL int = iota
	NEWUNINSTALL
	NEWBOTH
	instFilename   string = "Instances.csv"
	uninstFilename string = "UninstallEvents.csv"
)

type Element struct {
	Instance string
	DeviceID string
	Datetime time.Time
}

func main() {
	var logIt bool
	flag.BoolVar(&logIt, "verbose", false, "print more verbose output")
	flag.Parse()

	var oldIns, newIns, unIns = Element{}, Element{}, Element{}

	bestMatch := []Element{}
	instruction := NEWBOTH

	fout, err := os.Create("tf.csv")
	if err != nil {
		panic(err)
	}
	defer fout.Close()

	finst, rinst := openCSVReader(instFilename)
	defer finst.Close()
	noMoreInstalls := false
	funinst, runinst := openCSVReader(uninstFilename)
	defer funinst.Close()

	_, _ = rinst.Read()
	_, _ = runinst.Read()
	for uninstallLine, err := runinst.Read(); ; uninstallLine, err = runinst.Read() {
		// Handle eof and line reading errors
		if err != nil {
			if err == io.EOF {
				break
			} else {
				fmt.Println(err)
			}
		}
		parsed := parseTimestamp(uninstallLine[2])
		unIns = Element{DeviceID: uninstallLine[0], Datetime: parsed}

		for noMoreInstalls != true {
			if instruction == NEWINSTALL || instruction == NEWBOTH {
				installLine, err := rinst.Read()
				if err != nil {
					if err == io.EOF {
						noMoreInstalls = true
						break
					} else {
						fmt.Println(err)
					}
				}

				parsed := parseTimestamp(installLine[2])
				newIns = Element{Instance: installLine[0], DeviceID: installLine[1], Datetime: parsed}
			}
			if logIt {
				fmt.Printf("%d (0Install 1Uninstall 2Both)\told:%s:%s | new:%s:%s | un:%s:%s\n",
					instruction,
					oldIns.DeviceID, oldIns.Datetime,
					newIns.DeviceID, newIns.Datetime,
					unIns.DeviceID, unIns.Datetime)
			}

			oldIsMatch, _ := isMatch(oldIns, unIns)
			newIsMatch, suggestionNew := isMatch(newIns, unIns)
			if oldIsMatch && newIsMatch {
				bestMatch = []Element{newIns, unIns}
				instruction = NEWINSTALL
			} else if oldIsMatch && newIsMatch == false {
				bestMatch = []Element{oldIns, unIns}
				writeBestMatch(fout, bestMatch)
				bestMatch = []Element{}
				instruction = NEWBOTH
			} else if oldIsMatch == false && newIsMatch {
				bestMatch = []Element{newIns, unIns}
				instruction = NEWINSTALL
			} else if oldIsMatch == false && newIsMatch == false {
				instruction = suggestionNew
			}

			if instruction == NEWINSTALL {
				oldIns = newIns
				continue
			} else if instruction == NEWUNINSTALL {
				break
			} else if instruction == NEWBOTH {
				oldIns = newIns
				break
			}
		}

	}

	if logIt {
		fmt.Printf("%d (0Install 1Uninstall 2Both)\told:%s:%s | new:%s:%s | un:%s:%s\n",
			instruction,
			oldIns.DeviceID, oldIns.Datetime,
			newIns.DeviceID, newIns.Datetime,
			unIns.DeviceID, unIns.Datetime)
	}

	if newIsMatch, _ := isMatch(newIns, unIns); newIsMatch {
		bestMatch := []Element{newIns, unIns}
		writeBestMatch(fout, bestMatch)
	} else if oldIsMatch, _ := isMatch(oldIns, unIns); oldIsMatch {
		bestMatch := []Element{oldIns, unIns}
		writeBestMatch(fout, bestMatch)
	}
}

func writeBestMatch(w io.Writer, bm []Element) {
	s := fmt.Sprintf("%s,%s,uninstall,%s,%s", bm[0].Instance, bm[0].DeviceID, bm[0].Datetime.Format("2006-01-02 15:04:05"), bm[1].Datetime.Format("2006-01-02 15:04:05"))
	fmt.Fprintln(w, s)
}

func isMatch(install Element, uninstall Element) (ismatch bool, fix int) {
	if install.DeviceID != uninstall.DeviceID {
		if install.DeviceID < uninstall.DeviceID {
			fix = NEWINSTALL
		} else if install.DeviceID > uninstall.DeviceID {
			fix = NEWUNINSTALL
		}
		return false, fix
	}
	if install.Datetime.Before(uninstall.Datetime) {
		return true, NEWBOTH
	} else if install.Datetime.Before(uninstall.Datetime) == false {
		return false, NEWUNINSTALL
	}
	fmt.Println("SHOULD NOT RUN")
	return false, fix
}

func parseTimestamp(s string) time.Time {
	parsed, err := time.Parse("Jan 2, 2006 15:04:05", s)
	if err != nil {
		fmt.Println(err)
	}
	return parsed
}

func openCSVReader(filename string) (*os.File, *csv.Reader) {
	f, err := os.Open(filename)
	if err != nil {
		exit(fmt.Sprintf("failed to open file: %s\n", filename))
	}
	return f, csv.NewReader(f)
}

func exit(msg string) {
	fmt.Printf(msg)
	os.Exit(1)
}
