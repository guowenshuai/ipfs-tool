package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	shell "github.com/ipfs/go-ipfs-api"
	"github.com/urfave/cli/v2" // imports as package "cli"
)

const (
	FILE = 2
	DIR  = 1
)

var log *os.File
var existFiles []string

func main() {
	var err error
	log, err = os.OpenFile("ipfs-tool.log", os.O_CREATE, 0666)
	if err != nil {
		panic(err.Error())
	}
	buf := bufio.NewReader(log)
	for {
		line, err := buf.ReadString('\n')
		line = strings.TrimSpace(line)
		if len(line) != 0 {
			fields := strings.Fields(line)
			if len(fields) > 1 {
				existFiles = append(existFiles, fields[0])
			}
		}
		// fmt.Println(line)
		if err != nil {
			if err == io.EOF {
				// fmt.Println("File read ok!")
				break
			} else {
				fmt.Println("Read file error!", err)
				panic(err)
			}
		}
	}
	log.Close()
	log, err = os.OpenFile("ipfs-tool.log", os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		panic(err.Error())
	}
	defer log.Close()

	app := &cli.App{
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "server",
				Usage: "ipfs server",
				Value: "127.0.0.1",
			}, &cli.IntFlag{
				Name:  "port",
				Usage: "ipfs port",
				Value: 5001,
			},
		},
		Commands: []*cli.Command{
			{
				Name:    "add",
				Aliases: []string{"a"},
				Usage:   "Add a file or dir to ipfs",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "recursive",
						Aliases: []string{"r"},
						Usage:   "add recursive for dirs, not entire dir",
						Value:   false,
					},
				},
				Action: func(c *cli.Context) error {
					sh := NewSH(c.String("server"), c.Int("port"))
					push(sh, c.Args().First(), c.Bool("recursive"))
					// fmt.Println("added task: ", c.Args().First())
					return nil
				},
			},
			{
				Name:    "list",
				Aliases: []string{"l"},
				Usage:   "List links from an cid",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "recursive",
						Aliases: []string{"r"},
						Usage:   "list recursive for dirs",
						Value:   false,
					},
				},
				Action: func(c *cli.Context) error {
					sh := NewSH(c.String("server"), c.Int("port"))
					list(sh, ".", c.Args().First(), c.Bool("recursive"))
					return nil
				},
			},
		},
	}

	err = app.Run(os.Args)
	if err != nil {
		panic(err)
	}
}

func NewSH(server string, port int) *shell.Shell {
	return shell.NewShell(fmt.Sprintf("%s:%d", server, port))
}

func push(sh *shell.Shell, path string, recursive bool) {
	fileinfo, err := os.Stat(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s", err.Error())
		os.Exit(1)
	}
	if fileinfo.IsDir() {
		if !recursive {  // 上传一个目录
			cid, err := sh.AddDir(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %s", err)
				os.Exit(1)
			}
			write2log(path, cid, 0, DIR)
			// fmt.Printf("%s %s %d %d\n", path, cid, 0, DIR)
			os.Exit(0)
		} else { // 递归上传下面的文件
			files := walkDirs(path)
			for _, f := range files {
				pushOneFile(sh, f.path, f.info)
			}
		}
	} else {
		pushOneFile(sh, path, fileinfo)
	}
	return
}

func pushOneFile(sh *shell.Shell, path string, fileinfo os.FileInfo) {
	for _, eachItem := range existFiles { // 存在，则不继续上传
		if eachItem == path {
			fmt.Printf("path %s already pushed in log\n", path)
			return
		}
	}
	fh, err := os.Open(path)
	defer fh.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s", err)
		os.Exit(-1)
	}
	cid, err := sh.Add(fh)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s", err)
		os.Exit(1)
	}
	write2log(path, cid, fileinfo.Size(), FILE)
	// fmt.Printf("%s %s %d %d\n", path, cid, fileinfo.Size(), FILE)
	return
}

func list(sh *shell.Shell, base, cid string, recursive bool) {
	links, err := sh.List(cid)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s", err.Error())
		os.Exit(1)
	}
	if links == nil {
		os.Exit(0)
	}
	for _, l := range links {
		fmt.Printf("%s %s %d %d\n", path.Join(base, l.Name), l.Hash, l.Size, l.Type)
		if recursive && l.Type == 1 {
			list(sh, path.Join(base, l.Name), l.Hash, recursive)
		}
	}
}

type fileInfo struct {
	path string
	info os.FileInfo
}

func walkDirs(dir string) []*fileInfo {
	files := make([]*fileInfo, 0)
	err := filepath.Walk(dir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				files = append(files, &fileInfo{
					path: path,
					info: info,
				})
				// fmt.Println(path, info.Size())
			}
			return nil
		})
	if err != nil {
		fmt.Println(err)
	}
	return files
}

func write2log(fullname, hash string, size, ftype int64) {
	if log != nil {
		log.WriteString(fmt.Sprintf("%s %s %d %d\n", fullname, hash, size, ftype))
	}
	fmt.Printf("%s %s %d %d\n", fullname, hash, size, ftype)
}
