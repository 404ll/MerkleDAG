package main

import (
	"fmt"
	"os"
	"path/filepath"

	"merkledag/importer"
	"merkledag/object"
	"merkledag/resolver"
	"merkledag/store"
)

const defaultObjectDir = "data/objects"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "错误:", err)
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
			return fmt.Errorf("用法: mdag add <本地路径>")
		}
		rootCID, err := importer.AddPath(args[1], st)
		if err != nil {
			return err
		}
		fmt.Println("根 CID:", rootCID)
		return nil
	case "resolve":
		if len(args) != 3 {
			return fmt.Errorf("用法: mdag resolve <根CID> <路径>")
		}
		result, err := resolver.Resolve(args[1], args[2], st)
		if err != nil {
			return err
		}
		fmt.Println("目标 CID:", result.CID)
		fmt.Println("类型:", displayType(result.Type))
		return nil
	case "cat":
		if len(args) != 3 {
			return fmt.Errorf("用法: mdag cat <根CID> <路径>")
		}
		data, err := resolver.ReadFile(args[1], args[2], st)
		if err != nil {
			return err
		}
		fmt.Print(string(data))
		return nil
	case "ls":
		if len(args) != 3 {
			return fmt.Errorf("用法: mdag ls <根CID> <路径>")
		}
		entries, err := resolver.List(args[1], args[2], st)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			fmt.Printf("%s\t%s\t%s\t%d\n", displayType(entry.Type), entry.Name, entry.CID, entry.Size)
		}
		return nil
	default:
		return usage()
	}
}

func usage() error {
	return fmt.Errorf("用法: mdag <add|resolve|cat|ls> [参数]")
}

func displayType(objectType object.ObjectType) string {
	switch objectType {
	case object.BlobType:
		return "Blob（文件）"
	case object.TreeType:
		return "Tree（目录）"
	case object.ListType:
		return "List（分块文件）"
	default:
		return string(objectType)
	}
}
