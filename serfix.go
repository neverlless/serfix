package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"runtime"
)

const (
	helpFlagUsage  = "Help and usage instructions"
	forceFlagUsage = "Force overwrite of destination file if it exists"
	readBuffer     = 2 * 2 * 2 * 2 * 1024 * 1024
)

var helpPtr = flag.Bool("help", false, helpFlagUsage)
var forcePtr = flag.Bool("force", false, forceFlagUsage)
var counter int = 0
var lexer = regexp.MustCompile(`s:\d+:\\?\".*?\\?\";`)
var re = regexp.MustCompile(`(s:)(\d+)(:\\?\")(.*?)(\\?\";)`)
var esc = regexp.MustCompile(`(\\"|\\'|\\\\|\\a|\\b|\\f|\\n|\\r|\\s|\\t|\\v|\\0)`)

func init() {
	// Short flags too
	flag.BoolVar(helpPtr, "h", false, helpFlagUsage)
	flag.BoolVar(forcePtr, "f", false, forceFlagUsage)
}

func main() {
	numCPU := runtime.NumCPU()
	runtime.GOMAXPROCS(numCPU)

	flag.Parse()
	args := flag.Args()

	if *helpPtr {
		PrintUsage()
		return
	}

	if len(args) > 0 {
		processFile(args)
	} else {
		processStdin()
	}
}

func processFile(args []string) {
	if len(args) < 1 {
		fmt.Println("No input file provided")
		return
	}

	filename := args[0]
	infile, err := os.Open(filename)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer infile.Close()

	outfilename := getOutFilename(args)
	if outfilename == "" {
		return
	}

	tempfilename := outfilename + "~"
	tempfile, err := os.Create(tempfilename)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer tempfile.Close()

	if err := processLines(infile, tempfile); err != nil {
		fmt.Println(err)
		return
	}

	if err := os.Rename(tempfilename, outfilename); err != nil {
		fmt.Println(err)
		return
	}

	if len(args) == 1 {
		if err := os.Remove(filename); err != nil {
			fmt.Println(err)
			return
		}
	}
}

func getOutFilename(args []string) string {
	if len(args) < 2 {
		fmt.Println("No output file provided")
		return ""
	}

	outfilename := args[1]
	if !*forcePtr {
		if _, err := os.Stat(outfilename); err == nil {
			fmt.Println("Destination file already exists, aborting serfix.")
			return ""
		}
	}
	return outfilename
}

func processLines(infile *os.File, tempfile *os.File) error {
	r := bufio.NewReaderSize(infile, readBuffer)
	for {
		line, err := r.ReadString('\n')
		if err != nil && err != io.EOF {
			return err
		}

		if err == io.EOF {
			break
		}

		if _, err := tempfile.WriteString(lexer.ReplaceAllStringFunc(string(line), Replace)); err != nil {
			return err
		}
	}
	return nil
}

func processStdin() {
	r := bufio.NewReaderSize(os.Stdin, readBuffer)
	for {
		line, isPrefix, err := r.ReadLine()
		if err != nil && err != io.EOF {
			fmt.Println(err)
			return
		}

		if isPrefix {
			fmt.Println(errors.New("serfix: buffer size too small"))
			return
		}

		if err == io.EOF {
			break
		}

		fmt.Println(lexer.ReplaceAllStringFunc(string(line), Replace))
	}
}

func Replace(matches string) string {
	parts := re.FindStringSubmatch(matches)
	stringLength := len(parts[4]) - len(esc.FindAllString(parts[4], -1))
	return fmt.Sprintf("%s%d%s%s%s", parts[1], stringLength, parts[3], parts[4], parts[5])
}

func PrintUsage() {
	fmt.Println("Usage: serfix [flags] filename [outfilename]")
	fmt.Println("Alt. Usage: cat filename | serfix")
	fmt.Println("")
	fmt.Println("\t -f, --force \t\t\t Force overwrite of destination file if it exists.")
	fmt.Println("\t -h, --help  \t\t\t Print serfix help.")
	fmt.Println("")
}
