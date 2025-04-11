package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"regexp"
	"runtime/pprof"
	"strings"

	"github.com/antonmedv/fx/display"
	"github.com/antonmedv/fx/internal/complete"
	"github.com/antonmedv/fx/internal/theme"
	"github.com/goccy/go-yaml"
	"github.com/mattn/go-isatty"
)

var (
	flagYaml bool
	flagComp bool
)

func main() {
	if _, ok := os.LookupEnv("FX_PPROF"); ok {
		f, err := os.Create("cpu.prof")
		if err != nil {
			panic(err)
		}
		err = pprof.StartCPUProfile(f)
		if err != nil {
			panic(err)
		}
		defer f.Close()
		defer pprof.StopCPUProfile()
		memProf, err := os.Create("mem.prof")
		if err != nil {
			panic(err)
		}
		defer memProf.Close()
		defer pprof.WriteHeapProfile(memProf)
	}

	if complete.Complete() {
		os.Exit(0)
		return
	}

	var args []string
	for _, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, "--comp") {
			flagComp = true
			continue
		}
		switch arg {
		case "-h", "--help":
			fmt.Println(display.Usage(version))
			return
		case "-v", "-V", "--version":
			fmt.Println(version)
			return
		case "--themes":
			theme.ThemeTester()
			return
		case "--export-themes":
			theme.ExportThemes()
			return
		default:
			args = append(args, arg)
		}
	}

	if flagComp {
		shell := flag.String("comp", "", "")
		flag.Parse()
		switch *shell {
		case "bash":
			fmt.Print(complete.Bash())
		case "zsh":
			fmt.Print(complete.Zsh())
		case "fish":
			fmt.Print(complete.Fish())
		default:
			fmt.Println("unknown shell type")
		}
		return
	}

	fd := os.Stdin.Fd()
	stdinIsTty := isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
	var fileName string
	var src io.Reader

	if stdinIsTty && len(args) == 0 {
		fmt.Println(display.Usage(version))
		return
	} else if stdinIsTty && len(args) == 1 {
		filePath := args[0]
		f, err := os.Open(filePath)
		if err != nil {
			var pathError *fs.PathError
			if errors.As(err, &pathError) {
				fmt.Println(err)
				os.Exit(1)
			} else {
				panic(err)
			}
		}
		fileName = path.Base(filePath)
		src = f
		hasYamlExt, _ := regexp.MatchString(`(?i)\.ya?ml$`, fileName)
		if !flagYaml && hasYamlExt {
			flagYaml = true
		}
	} else if !stdinIsTty && len(args) == 0 {
		src = os.Stdin
	} else {
		reduce(os.Args[1:])
		return
	}

	data, err := io.ReadAll(src)
	if err != nil {
		panic(err)
	}

	if flagYaml {
		data, err = yaml.YAMLToJSON(data)
		if err != nil {
			fmt.Print(err.Error())
			os.Exit(1)
			return
		}
	}

	display.Display(data, fileName)
}
