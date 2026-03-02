// roomctl handles room content validation and bundle publishing.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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
		runBuild()
	case "publish":
		runPublish()
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
	fmt.Fprintln(os.Stderr, "  build --room <slug|path> --version <N>")
	fmt.Fprintln(os.Stderr, "  publish --room <slug|path> --version <N> [--activate] [--bff-url URL] [--s3-endpoint URL] [--s3-bucket NAME] [--s3-access-key KEY] [--s3-secret-key KEY] [--admin-api-key KEY]")
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
	roomPath := resolveRoomPath(roomArg)

	if err := roomctl.ValidateRoomDir(roomPath); err != nil {
		fmt.Fprintf(os.Stderr, "validate failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("validation passed")
}

func runBuild() {
	if len(os.Args) != 6 {
		fmt.Fprintln(os.Stderr, "Usage: roomctl build --room <slug|path> --version <N>")
		os.Exit(1)
	}

	var roomArg string
	var versionArg string
	for i := 2; i < len(os.Args); i += 2 {
		if i+1 >= len(os.Args) {
			fmt.Fprintln(os.Stderr, "Usage: roomctl build --room <slug|path> --version <N>")
			os.Exit(1)
		}
		switch os.Args[i] {
		case "--room":
			roomArg = os.Args[i+1]
		case "--version":
			versionArg = os.Args[i+1]
		default:
			fmt.Fprintln(os.Stderr, "Usage: roomctl build --room <slug|path> --version <N>")
			os.Exit(1)
		}
	}
	if roomArg == "" || versionArg == "" {
		fmt.Fprintln(os.Stderr, "Usage: roomctl build --room <slug|path> --version <N>")
		os.Exit(1)
	}

	version, err := strconv.Atoi(versionArg)
	if err != nil || version < 1 {
		fmt.Fprintln(os.Stderr, "--version must be a positive integer")
		os.Exit(1)
	}

	res, err := roomctl.BuildRoom(resolveRoomPath(roomArg), version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "build failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("build succeeded: %s (%s)\n", res.BundlePath, res.Manifest.BundleHashSha256)
}

func runPublish() {
	usage := "Usage: roomctl publish --room <slug|path> --version <N> [--activate] [--bff-url URL] [--s3-endpoint URL] [--s3-bucket NAME] [--s3-access-key KEY] [--s3-secret-key KEY] [--admin-api-key KEY]"
	if len(os.Args) < 6 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(1)
	}

	var roomArg string
	var versionArg string
	opts := roomctl.PublishOptions{}

	for i := 2; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--activate":
			opts.Activate = true
		case "--room", "--version", "--bff-url", "--s3-endpoint", "--s3-bucket", "--s3-access-key", "--s3-secret-key", "--admin-api-key":
			if i+1 >= len(os.Args) {
				fmt.Fprintln(os.Stderr, usage)
				os.Exit(1)
			}
			val := os.Args[i+1]
			i++
			switch os.Args[i-1] {
			case "--room":
				roomArg = val
			case "--version":
				versionArg = val
			case "--bff-url":
				opts.BFFURL = val
			case "--s3-endpoint":
				opts.S3Endpoint = val
			case "--s3-bucket":
				opts.S3Bucket = val
			case "--s3-access-key":
				opts.S3AccessKey = val
			case "--s3-secret-key":
				opts.S3SecretKey = val
			case "--admin-api-key":
				opts.AdminAPIKey = val
			}
		default:
			fmt.Fprintln(os.Stderr, usage)
			os.Exit(1)
		}
	}

	if roomArg == "" || versionArg == "" {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(1)
	}

	version, err := strconv.Atoi(versionArg)
	if err != nil || version < 1 {
		fmt.Fprintln(os.Stderr, "--version must be a positive integer")
		os.Exit(1)
	}

	opts.RoomDir = resolveRoomPath(roomArg)
	opts.Version = version

	res, err := roomctl.PublishRoom(context.Background(), opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "publish failed: %v\n", err)
		os.Exit(1)
	}

	shortHash := res.Hash
	if len(shortHash) > 12 {
		shortHash = shortHash[:12]
	}
	if res.Activated {
		fmt.Printf("Published %s v%d (hash: %s) [activated]\n", res.Slug, res.Version, shortHash)
		return
	}
	fmt.Printf("Published %s v%d (hash: %s)\n", res.Slug, res.Version, shortHash)
}

func resolveRoomPath(roomArg string) string {
	roomPath := roomArg
	if !strings.ContainsRune(roomArg, filepath.Separator) {
		if _, err := os.Stat(roomArg); err != nil {
			roomPath = filepath.Join("rooms", roomArg)
		}
	}
	return roomPath
}
