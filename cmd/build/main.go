// Command build builds the module programs and the application.
//
// Run it from the repository root:
//
//	go run ./cmd/build [flags]
//
// It builds every modules/<id> whose metadata.json lists the current OS in
// "platforms" into build/bin/modules/<id>.<GOOS>.<GOARCH>.vfgmod, writes
// the installer's module sections (build/windows/nsis/modules.nsh)
// from their metadata.json, regenerates the build assets (version info,
// manifests, installer defines) from build/config.yml with "wails3 task
// common:update:build-assets", and then builds the application with the
// Wails v3 CLI ("wails3 build", or "wails3 package" for the NSIS installer
// and the macOS app bundle). The NSIS installer (-nsis) is only built on Windows,
// one per architecture (-arch). The Linux packages (-linux-packages: AppImage,
// deb, rpm and Arch Linux) are only built on Linux.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/SayukiDev/VRCFaceTrackingGO/internal/manager"
	"github.com/SayukiDev/VRCFaceTrackingGO/internal/metadata"
	"github.com/SayukiDev/VRCFaceTrackingGO/pkg/module/adapter"
	"github.com/SayukiDev/VRCFaceTrackingGO/pkg/module/plugin"
)

const (
	modulesDir         = "modules"
	moduleMetadataFile = "metadata.json"
	moduleOutDir       = "build/bin/modules"
	nsisInclude        = "build/windows/nsis/modules.nsh"
	metadataFile       = "build/config.yml"
	moduleLDFlags      = "-s -w"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func run() error {
	app := flag.Bool("app", true, "build the application with wails after the modules")
	assets := flag.Bool("assets", true, "regenerate the build assets from build/config.yml before building the application (overwrites manual edits to them)")
	nsis := flag.Bool("nsis", runtime.GOOS == "windows", "also build the NSIS installer (needs -app, Windows only)")
	linuxPackages := flag.Bool("linux-packages", false, "also build the AppImage and the deb, rpm and Arch Linux packages (needs -app, Linux only)")
	install := flag.Bool("install", false, "copy the built modules into the current user's module folder")
	tags := flag.String("tags", "", "extra build tags for the application (e.g. gtk3 for GTK3 and WebKitGTK 4.1 on Linux)")
	arch := flag.String("arch", runtime.GOARCH, "target architecture (GOARCH: amd64 or arm64) for the modules and the application")
	flag.Parse()

	raw, err := os.ReadFile(metadataFile)
	if err != nil {
		return fmt.Errorf("run from the repository root: %w", err)
	}
	metadata.Init(raw)
	if *nsis && runtime.GOOS != "windows" {
		return fmt.Errorf("the NSIS installer can only be built on Windows")
	}
	if *linuxPackages && runtime.GOOS != "linux" {
		return fmt.Errorf("the Linux packages can only be built on Linux")
	}
	if *install && *arch != runtime.GOARCH {
		return fmt.Errorf("-install needs modules for this machine's architecture (%s), not %s", runtime.GOARCH, *arch)
	}

	modules, err := buildModules(*arch)
	if err != nil {
		return err
	}
	if *install {
		if err := installModules(modules); err != nil {
			return err
		}
	}
	if !*app {
		return nil
	}
	if *assets {
		// The task passes -config and -name; without them the assets get
		// the CLI's placeholder values ("My Product", version 0.1.0).
		if err := runCmd("wails3", "task", "common:update:build-assets"); err != nil {
			return fmt.Errorf("update build assets: %w", err)
		}
	}
	// "wails3 package" builds the application and then packages it: the NSIS
	// installer on Windows, the .app bundle on macOS, the AppImage and the
	// deb, rpm and Arch Linux packages on Linux.
	cmd := "build"
	if *nsis {
		if err := writeNSISInclude(modules); err != nil {
			return err
		}
		cmd = "package"
	}
	if runtime.GOOS == "darwin" || *linuxPackages {
		cmd = "package"
	}
	args := []string{cmd, "GOARCH=" + *arch}
	if *tags != "" {
		args = append(args, "EXTRA_TAGS="+*tags)
	}
	return runCmd("wails3", args...)
}

// module is one built module program. Its ID is the directory name, which
// is also the name the host installs it under.
type module struct {
	id   string
	path string
	meta adapter.Metadata
}

// buildModules builds every modules/<id> that is a main package and whose
// metadata.json supports the current OS, for the architecture arch.
func buildModules(arch string) ([]module, error) {
	entries, err := os.ReadDir(modulesDir)
	if err != nil {
		return nil, err
	}
	var built []module
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		id := e.Name()
		src := filepath.Join(modulesDir, id)
		if _, err := os.Stat(filepath.Join(src, "main.go")); err != nil {
			continue
		}
		meta, err := adapter.LoadMetadata(os.DirFS(src), moduleMetadataFile)
		if err != nil {
			return nil, fmt.Errorf("module %s: %w", id, err)
		}
		if !meta.SupportsPlatform(runtime.GOOS) {
			fmt.Printf("Skipping module %s (platforms: %s)\n", id, strings.Join(meta.Platforms, ", "))
			continue
		}
		out := filepath.Join(moduleOutDir, id+plugin.PlatformSuffix(runtime.GOOS, arch))
		fmt.Printf("Building module %s -> %s\n", id, out)
		build := exec.Command("go", "build", "-ldflags="+moduleLDFlags, "-o", out, "./"+modulesDir+"/"+id)
		build.Env = append(os.Environ(), "GOARCH="+arch)
		if err := runExec(build); err != nil {
			return nil, fmt.Errorf("build module %s: %w", id, err)
		}
		built = append(built, module{id: id, path: out, meta: meta})
	}
	return built, nil
}

// installModules copies the modules to where the application looks for
// them. The application must not be running.
func installModules(modules []module) error {
	dataDir, err := metadata.Info.PersistentDataDir()
	if err != nil {
		return err
	}
	dir := manager.DefaultDir(dataDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for _, m := range modules {
		dst := filepath.Join(dir, m.id+plugin.HostSuffix)
		fmt.Printf("Installing module %s -> %s\n", m.id, dst)
		data, err := os.ReadFile(m.path)
		if err != nil {
			return err
		}
		if err := os.WriteFile(dst, data, 0o755); err != nil {
			return fmt.Errorf("install module %s (is the application running?): %w", m.id, err)
		}
	}
	return nil
}

// nsisString escapes s for a double quoted NSIS string.
var nsisString = strings.NewReplacer(`$`, `$$`, `"`, `$\"`, "\r", "", "\n", `$\n`)

// writeNSISInclude writes the installer sections and component page
// descriptions for the modules; project.nsi includes the file.
func writeNSISInclude(modules []module) error {
	var b strings.Builder
	b.WriteString("; Generated by cmd/build. Do not edit.\n\n")
	b.WriteString("!macro moduleSections\n")
	for _, m := range modules {
		fmt.Fprintf(&b, "    Section \"%s\" %s\n", nsisString.Replace(m.meta.Name), sectionID(m.id))
		fmt.Fprintf(&b, "        !insertmacro installModule \"%s\"\n", filepath.Base(m.path))
		b.WriteString("    SectionEnd\n")
	}
	b.WriteString("!macroend\n\n")
	b.WriteString("!macro moduleDescriptions\n")
	for _, m := range modules {
		fmt.Fprintf(&b, "    !insertmacro MUI_DESCRIPTION_TEXT ${%s} \"%s\"\n", sectionID(m.id), nsisString.Replace(m.meta.Description))
	}
	b.WriteString("!macroend\n")
	return os.WriteFile(nsisInclude, []byte(b.String()), 0o644)
}

// sectionID turns a module ID into an NSIS section index name, which may
// only contain letters, digits and underscores.
func sectionID(id string) string {
	return "SecMod_" + strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			return r
		}
		return '_'
	}, id)
}

func runCmd(name string, args ...string) error {
	return runExec(exec.Command(name, args...))
}

func runExec(cmd *exec.Cmd) error {
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
