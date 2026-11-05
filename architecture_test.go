package wechatease

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode"
)

const maxGoFileLines = 500

func TestGoFilesStayMaintainable(t *testing.T) {
	err := filepath.WalkDir(".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if path == ".git" || strings.HasPrefix(path, ".git"+string(filepath.Separator)) {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		lines, err := countFileLines(path)
		if err != nil {
			return err
		}
		if lines > maxGoFileLines {
			t.Errorf("%s has %d lines; maximum is %d", path, lines, maxGoFileLines)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk Go files: %v", err)
	}
}

func TestMethodsHaveUnitTests(t *testing.T) {
	testNamesByDir := make(map[string]map[string]struct{})
	methodsByDir := make(map[string][]methodName)
	fset := token.NewFileSet()
	err := filepath.WalkDir(".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if path == ".git" || strings.HasPrefix(path, ".git"+string(filepath.Separator)) {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
		dir := filepath.Dir(path)
		if strings.HasSuffix(path, "_test.go") {
			if testNamesByDir[dir] == nil {
				testNamesByDir[dir] = make(map[string]struct{})
			}
			collectTestNames(file, testNamesByDir[dir])
			return nil
		}
		methodsByDir[dir] = append(methodsByDir[dir], collectMethods(file)...)
		return nil
	})
	if err != nil {
		t.Fatalf("walk Go files: %v", err)
	}

	for dir, methods := range methodsByDir {
		testNames := testNamesByDir[dir]
		for _, method := range methods {
			methodCandidate := "Test" + upperFirst(method.name)
			receiverCandidate := "Test" + upperFirst(method.receiver) + upperFirst(method.name)
			if !hasTestName(testNames, receiverCandidate) && !hasTestName(testNames, methodCandidate) {
				t.Errorf("%s: method %s.%s has no unit test (expected %s or %s)", dir, method.receiver, method.name, receiverCandidate, methodCandidate)
			}
		}
	}
}

type methodName struct {
	receiver string
	name     string
}

func countFileLines(path string) (int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	lines := 0
	for scanner.Scan() {
		lines++
	}
	return lines, scanner.Err()
}

func collectTestNames(file *ast.File, names map[string]struct{}) {
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if ok && fn.Recv == nil && strings.HasPrefix(fn.Name.Name, "Test") {
			names[fn.Name.Name] = struct{}{}
		}
	}
}

func collectMethods(file *ast.File) []methodName {
	var methods []methodName
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv == nil || len(fn.Recv.List) == 0 {
			continue
		}
		receiver, err := receiverTypeName(fn.Recv.List[0].Type)
		if err != nil {
			continue
		}
		methods = append(methods, methodName{receiver: receiver, name: fn.Name.Name})
	}
	return methods
}

func receiverTypeName(expr ast.Expr) (string, error) {
	switch value := expr.(type) {
	case *ast.Ident:
		return value.Name, nil
	case *ast.StarExpr:
		return receiverTypeName(value.X)
	default:
		return "", fmt.Errorf("unsupported receiver %T", expr)
	}
}

func hasTestName(names map[string]struct{}, name string) bool {
	_, ok := names[name]
	return ok
}

func upperFirst(value string) string {
	runes := []rune(value)
	if len(runes) == 0 {
		return value
	}
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}
