package main

import (
	"fmt"
	"os"
	"path/filepath"

	"merkledag/importer"
	"merkledag/resolver"
	"merkledag/store"
)

const defaultObjectDir = "data/objects"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return usage()
	}

	st := store.NewFileStore(filepath.FromSlash(defaultObjectDir))

	switch args[0] {
	case "add":
		if len(args) != 2 {
			return fmt.Errorf("usage: mdag add <local-path>")
		}
		rootCID, err := importer.AddPath(args[1], st)
		if err != nil {
			return err
		}
		fmt.Println("Root CID:", rootCID)
		return nil
	case "resolve":
		if len(args) != 3 {
			return fmt.Errorf("usage: mdag resolve <root-cid> <path>")
		}
		result, err := resolver.Resolve(args[1], args[2], st)
		if err != nil {
			return err
		}
		fmt.Println("Target CID:", result.CID)
		fmt.Println("Type:", result.Type)
		return nil
	case "cat":
		if len(args) != 3 {
			return fmt.Errorf("usage: mdag cat <root-cid> <path>")
		}
		data, err := resolver.ReadFile(args[1], args[2], st)
		if err != nil {
			return err
		}
		fmt.Print(string(data))
		return nil
	case "ls":
		if len(args) != 3 {
			return fmt.Errorf("usage: mdag ls <root-cid> <path>")
		}
		entries, err := resolver.List(args[1], args[2], st)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			fmt.Printf("%s\t%s\t%s\t%d\n", entry.Type, entry.Name, entry.CID, entry.Size)
		}
		return nil
	default:
		return usage()
	}
}

func usage() error {
	return fmt.Errorf("usage: mdag <add|resolve|cat|ls> [args]")
}
