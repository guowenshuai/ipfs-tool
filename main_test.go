package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"
)

func Test_Main(t *testing.T) {
	err := filepath.Walk(".",
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				fmt.Println(path, info.Size())
			}
			return nil
		})
	if err != nil {
		log.Println(err)
	}
}