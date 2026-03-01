// roomctl handles room content validation and bundle publishing.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sidsharma96/SysEscape/internal/roomctl"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "validate":
		runValidate()
	case "build":
		fmt.Fprintln(os.Stderr, "build not implemented")
		os.Exit(1)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "Usage: roomctl <command> [args]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  validate --room <slug|path>")
	fmt.Fprintln(os.Stderr, "  validate --all")
	fmt.Fprintln(os.Stderr, "  build ... (not implemented)")
}

func runValidate() {
	if len(os.Args) != 4 && len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "Usage: roomctl validate --room <slug|path> | --all")
		os.Exit(1)
	}

	if len(os.Args) == 3 && os.Args[2] == "--all" {
		if err := roomctl.ValidateAllRooms("rooms"); err != nil {
			fmt.Fprintf(os.Stderr, "validate failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("validation passed")
		return
	}

	if len(os.Args) != 4 || os.Args[2] != "--room" {
		fmt.Fprintln(os.Stderr, "Usage: roomctl validate --room <slug|path> | --all")
		os.Exit(1)
	}

	roomArg := os.Args[3]
	roomPath := roomArg
	if !strings.ContainsRune(roomArg, filepath.Separator) {
		if _, err := os.Stat(roomArg); err != nil {
			roomPath = filepath.Join("rooms", roomArg)
		}
	}

	if err := roomctl.ValidateRoomDir(roomPath); err != nil {
		fmt.Fprintf(os.Stderr, "validate failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("validation passed")
}
