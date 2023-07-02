package binaries

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"

	"github.com/vnsoft2014/prisma-client-go/binaries/platform"
	"github.com/vnsoft2014/prisma-client-go/logger"
)

// PrismaVersion is a hardcoded version of the Prisma CLI.
const PrismaVersion = "4.16.0"

// EngineVersion is a hardcoded version of the Prisma Engine.
// The versions can be found under https://github.com/prisma/prisma-engines/commits/main
const EngineVersion = "b20ead4d3ab9e78ac112966e242ded703f4a052c"

// PrismaURL points to an S3 bucket URL where the CLI binaries are stored.
var PrismaURL = "https://packaged-cli.prisma.sh/%s-%s-%s-%s.gz"

// EngineURL points to an S3 bucket URL where the Prisma engines are stored.
var EngineURL = "https://binaries.prisma.sh/all_commits/%s/%s/%s.gz"

type Engine struct {
	Name string
	Env  string
}

var Engines = []Engine{{
	"query-engine",
	"PRISMA_QUERY_ENGINE_BINARY",
}, {
	"migration-engine",
	"PRISMA_MIGRATION_ENGINE_BINARY",
}}

// init overrides URLs if env variables are specific for debugging purposes and to
// be able to provide a fallback if the links above should go down
func init() {
	if prismaURL, ok := os.LookupEnv("PRISMA_CLI_URL"); ok {
		PrismaURL = prismaURL
	}
	if engineURL, ok := os.LookupEnv("PRISMA_ENGINE_URL"); ok {
		EngineURL = engineURL
	}
}

// PrismaCLIName returns the local file path of where the CLI lives
func PrismaCLIName() string {
	variation := platform.Name()
	arch := platform.Arch()
	return fmt.Sprintf("prisma-cli-%s-%s", variation, arch)
}

var baseDirName = path.Join("prisma", "binaries")

// GlobalTempDir returns the path of where the engines live
// internally, this is the global temp dir
func GlobalTempDir(version string) string {
	temp := os.TempDir()
	logger.Debug.Printf("temp dir: %s", temp)

	return path.Join(temp, baseDirName, "engines", version)
}

func GlobalUnpackDir(version string) string {
	return path.Join(GlobalTempDir(version), "unpacked", "v2")
}

// GlobalCacheDir returns the path of where the CLI lives
// internally, this is the global temp dir
func GlobalCacheDir() string {
	cache, err := os.UserCacheDir()
	if err != nil {
		panic(fmt.Errorf("could not read user cache dir: %w", err))
	}

	logger.Debug.Printf("global cache dir: %s", cache)

	return path.Join(cache, baseDirName, "cli", PrismaVersion)
}

func FetchEngine(dir string, engineName string, binaryName string) error {
	logger.Debug.Printf("checking %s %s...", engineName, binaryName)

	to := GetEnginePath(dir, engineName, binaryName)

	if _, err := os.Stat(to); !os.IsNotExist(err) {
		logger.Debug.Printf("%s is cached at %s", engineName, to)
		return nil
	}

	url := platform.CheckForExtension(binaryName, fmt.Sprintf(EngineURL, EngineVersion, binaryName, engineName))

	logger.Debug.Printf("%s is missing, downloading...", engineName)

	logger.Debug.Printf("downloading %s from %s to %s", engineName, url, to)

	if err := download(url, to); err != nil {
		return fmt.Errorf("could not download %s to %s: %w", url, to, err)
	}

	logger.Debug.Printf("%s done", engineName)

	return nil
}

// FetchNative fetches the Prisma binaries needed for the generator to a given directory
func FetchNative(toDir string) error {
	if toDir == "" {
		return fmt.Errorf("toDir must be provided")
	}

	if !filepath.IsAbs(toDir) {
		return fmt.Errorf("toDir must be absolute")
	}

	if err := DownloadCLI(toDir); err != nil {
		return fmt.Errorf("could not download engines: %w", err)
	}

	for _, e := range Engines {
		if err := FetchEngine(toDir, e.Name, platform.BinaryPlatformNameStatic()); err != nil {
			return fmt.Errorf("could not download engines: %w", err)
		}
	}

	return nil
}

func DownloadCLI(toDir string) error {
	cli := PrismaCLIName()
	to := platform.CheckForExtension(platform.Name(), path.Join(toDir, cli))
	url := platform.CheckForExtension(platform.Name(), fmt.Sprintf(PrismaURL, "prisma-cli", PrismaVersion, platform.Name(), platform.Arch()))

	logger.Debug.Printf("ensuring CLI %s from %s to %s", cli, url, to)

	if _, err := os.Stat(to); os.IsNotExist(err) {
		logger.Info.Printf("prisma cli doesn't exist, fetching... (this might take a few minutes)")

		if err := download(url, to); err != nil {
			return fmt.Errorf("could not download %s to %s: %w", url, to, err)
		}

		logger.Info.Printf("prisma cli fetched successfully.")
	} else {
		logger.Debug.Printf("prisma cli is cached")
	}

	return nil
}

func GetEnginePath(dir, engineName, binaryName string) string {
	return platform.CheckForExtension(binaryName, path.Join(dir, EngineVersion, fmt.Sprintf("prisma-%s-%s", engineName, binaryName)))
}

func download(url string, to string) error {
	if err := os.MkdirAll(path.Dir(to), os.ModePerm); err != nil {
		return fmt.Errorf("could not run MkdirAll on path %s: %w", to, err)
	}

	// copy to temp file first
	dest := to + ".tmp"

	resp, err := http.Get(url) //nolint:gosec
	if err != nil {
		return fmt.Errorf("could not get %s: %w", url, err)
	}
	//goland:noinspection GoUnhandledErrorResult
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		out, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("received code %d from %s: %+v", resp.StatusCode, url, string(out))
	}

	out, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("could not create %s: %w", dest, err)
	}
	//goland:noinspection GoUnhandledErrorResult
	defer out.Close()

	if err := os.Chmod(dest, os.ModePerm); err != nil {
		return fmt.Errorf("could not chmod +x %s: %w", url, err)
	}

	g, err := gzip.NewReader(resp.Body)
	if err != nil {
		return fmt.Errorf("could not create gzip reader: %w", err)
	}
	//goland:noinspection GoUnhandledErrorResult
	defer g.Close()

	if _, err := io.Copy(out, g); err != nil { //nolint:gosec
		return fmt.Errorf("could not copy %s: %w", url, err)
	}

	// temp file is ready, now copy to the original destination
	if err := copyFile(dest, to); err != nil {
		return fmt.Errorf("copy temp file: %w", err)
	}

	return nil
}

func copyFile(from string, to string) error {
	input, err := os.ReadFile(from)
	if err != nil {
		return fmt.Errorf("readfile: %w", err)
	}

	if err := os.WriteFile(to, input, os.ModePerm); err != nil {
		return fmt.Errorf("writefile: %w", err)
	}

	return nil
}
