package trygo

import (
	"github.com/pkg/errors"
	"os"
	"path/filepath"
	"strings"
)

var cwd string

func init() {
	var err error
	if cwd, err = os.Getwd(); err != nil {
		panic(err)
	}
}

type Gen struct {
	FilePaths []string
	OutDir    string
}

func NewGenGoGenerate(outDir string) (*Gen, error) {
	gofile, ok := os.LookupEnv("GOFILE")
	if !ok {
		return nil, errors.New("`trygo` was not run from `go generate` and no path is given. Nothing to generate")
	}

	paths := []string{filepath.Join(cwd, gofile)}

	if outDir != "" {
		return &Gen{paths, outDir}, nil
	}

	root := cwd
	for {
		// TODO: .git is a file. In the case, .git file contains the path to $GIT_DIR
		if s, err := os.Stat(filepath.Join(root, ".git")); err == nil && s.IsDir() {
			break
		}
		prev := root
		root = filepath.Dir(root)
		if prev == root {
			return nil, errors.Errorf("File %q is not inside Git repository at %q", gofile, cwd)
		}
	}

	return &Gen{
		FilePaths: []string{filepath.Join(cwd, gofile)},
		OutDir:    filepath.Join(root, "gen"),
	}, nil
}

func NewGenWithPaths(paths []string, outDir string) (*Gen, error) {
	if outDir == "" {
		return nil, errors.New("Output directory must be given when paths are given")
	}

	files := []string{}
	for _, path := range paths {
		s, err := os.Stat(path)
		if err != nil {
			return nil, errors.Wrapf(err, "Cannot read %q", path)
		}
		if !s.IsDir() {
			if strings.HasSuffix(path, ".go") {
				files = append(files, path)
			}
			continue
		}
		if err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() || !strings.HasSuffix(p, ".go") {
				return nil
			}
			files = append(files, p)
			return nil
		}); err != nil {
			return nil, errors.Wrapf(err, "Cannot read directory %q", path)
		}
	}
	if len(files) == 0 {
		return nil, errors.New("No Go source is included in given paths")
	}
	return &Gen{
		FilePaths: files,
		OutDir:    outDir,
	}, nil
}

func NewGen(paths []string, outDir string) (*Gen, error) {
	if len(paths) == 0 {
		return NewGenGoGenerate(outDir)
	}
	return NewGenWithPaths(paths, outDir)
}
