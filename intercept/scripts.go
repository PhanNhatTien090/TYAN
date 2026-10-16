package intercept

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"mvdan.cc/sh/v3/syntax"
)

var scriptInterpreter = regexp.MustCompile(`^(python[0-9.]*|bash|sh|zsh|fish|node|nodejs|ruby|perl|php|pwsh|powershell)$`)

type scriptReference struct {
	path, problem string
	direct        bool
	inline        bool
	generated     bool
	content       string
}

// Only statically resolvable shell words are used. No environment expansion,
// command substitution, interpreter or shell is executed to discover a path.
func staticWord(w *syntax.Word) (string, bool) {
	if w == nil {
		return "", false
	}
	return staticParts(w.Parts)
}

// staticCommandName is like staticWord, but also accepts the POSIX test
// builtin `[` as a command name. `[` is a valid executable name, while the
// wildcard guard in staticParts intentionally rejects brackets in paths.
func staticCommandName(w *syntax.Word) (string, bool) {
	if value, ok := staticWord(w); ok {
		return value, true
	}
	if w != nil && len(w.Parts) == 1 {
		if lit, ok := w.Parts[0].(*syntax.Lit); ok && (lit.Value == "[" || lit.Value == "[[") {
			return lit.Value, true
		}
	}
	return "", false
}
func staticParts(parts []syntax.WordPart) (string, bool) {
	return literalShellParts(parts)
}

func scriptReferences(command, dir string, depth int) []scriptReference {
	remaining := maxInlineParses
	return scriptReferencesBounded(command, dir, depth, &remaining)
}

func scriptReferencesBounded(command, dir string, depth int, remaining *int) []scriptReference {
	if *remaining <= 0 || len(command) > maxReviewScriptBytes {
		return []scriptReference{{problem: "inline shell execution content exceeds static analysis total limit"}}
	}
	*remaining--
	if depth > 4 {
		return []scriptReference{{problem: "nested shell commands exceed static analysis depth"}}
	}
	tree, err := syntax.NewParser().Parse(strings.NewReader(command), "review")
	if err != nil {
		return []scriptReference{{problem: "cannot statically parse shell command: " + err.Error()}}
	}
	var refs []scriptReference
	sourceInputs := map[*syntax.CallExpr]reviewSourceInput{}
	generated := map[string]scriptReference{}
	topStatements := map[*syntax.Stmt]bool{}
	var markStraightLine func(*syntax.Stmt)
	markStraightLine = func(stmt *syntax.Stmt) {
		if stmt.Background {
			return
		}
		if binary, ok := stmt.Cmd.(*syntax.BinaryCmd); ok {
			// A foreground pipeline finishes before the next statement. A cat
			// producer with its own literal input/output redirections can be
			// captured independently of the consumer's stdout.
			if binary.Op == syntax.Pipe || binary.Op == syntax.PipeAll {
				if call, ok := binary.X.Cmd.(*syntax.CallExpr); ok && len(call.Args) == 1 {
					if name, known := staticCommandName(call.Args[0]); known && name == "cat" {
						topStatements[binary.X] = true
					}
				}
				return
			}
			if binary.Op != syntax.AndStmt {
				return
			}
			markStraightLine(binary.X)
			// `cd dir && cat ...` is the common success path. The script cannot
			// be created outside that directory by this branch.
			if staticScriptPreparation(binary.X) {
				markStraightLine(binary.Y)
			}
			return
		}
		topStatements[stmt] = true
	}
	for _, stmt := range tree.Stmts {
		markStraightLine(stmt)
	}
	constants := shellConstants(tree, topStatements)
	allowSed := staticSedCommandUnshadowed(tree)
	sedAnalysisRemaining, envAnalysisRemaining := 8192, 8192
	trackWrites := func(stmt *syntax.Stmt, args []*syntax.Word) {
		sedKnown := false
		if allowSed && len(args) == 3 && len(stmt.Redirs) > 0 {
			if name, known := staticCommandName(args[0]); known && filepath.Base(name) == "sed" {
				sedKnown = staticSedPrefixKnown(tree, stmt, &sedAnalysisRemaining)
			}
		}
		for _, ref := range generatedScriptWrites(stmt, args, dir, topStatements[stmt], generated, sedKnown) {
			generated[ref.path] = ref
		}
	}
	add := func(word *syntax.Word, direct bool) {
		path, ok := staticWord(word)
		if !ok {
			if fields, known := constantWordFields(word, constants); known && len(fields) == 1 {
				path, ok = fields[0], true
			}
		}
		if !ok || path == "" {
			refs = append(refs, scriptReference{problem: "script path contains dynamic expansion, cannot determine actual file"})
			return
		}
		if !filepath.IsAbs(path) {
			if dir == "" {
				if ref, known := generated[filepath.Clean(path)]; known {
					ref.direct = direct
					refs = append(refs, ref)
					return
				}

				refs = append(refs, scriptReference{path: path, problem: "working directory unknown, cannot resolve relative script path"})
				return
			}
			path = filepath.Join(dir, path)
		}
		absolute, err := filepath.Abs(path)
		if err != nil {
			refs = append(refs, scriptReference{path: path, problem: "cannot resolve absolute path"})
			return
		}
		if ref, ok := generated[absolute]; ok {
			ref.direct = direct
			refs = append(refs, ref)
			return
		}
		refs = append(refs, scriptReference{path: absolute, direct: direct})
	}
	// Walking the AST visits calls inside command substitutions as well, but not
	// string literals, comments or heredoc bodies. Function bodies/conditional cd
	// cannot establish a reliable working directory for subsequent calls.
	writes, unknownWrites := loopBindingWrites(tree, constants)
	var bindings map[string][]string
	var scopes []map[string][]string
	syntax.Walk(tree, func(node syntax.Node) bool {
		if node == nil {
			bindings = scopes[len(scopes)-1]
			scopes = scopes[:len(scopes)-1]
			return true
		}
		if _, ok := node.(*syntax.FuncDecl); ok {
			return false
		}
		scopes = append(scopes, bindings)
		if loop, ok := node.(*syntax.ForClause); ok {
			bindings = forLoopBindings(loop, bindings, writes, unknownWrites)
		}
		stmt, ok := node.(*syntax.Stmt)
		if !ok {
			return true
		}
		if stmt.Cmd == nil {
			trackWrites(stmt, nil)
			return true
		}
		call, ok := stmt.Cmd.(*syntax.CallExpr)
		if !ok {
			return true
		}
		args := call.Args
		if len(args) == 0 {
			trackWrites(stmt, args)
			return true
		}
		args, resolved := constantCallArgs(args, constants)
		name, ok := staticCommandName(args[0])
		if !resolved || !ok {
			refs = append(refs, scriptReference{problem: "dynamic command entry, cannot determine whether a script is executed"})
			return true
		}
		wrappedArgs := args
		args, name, ok = unwrapScriptCall(args)
		if !ok {
			refs = append(refs, scriptReference{problem: "wrapper command entry point is indeterminate"})
			return true
		}
		trackWrites(stmt, args)
		base := filepath.Base(name)
		if base == "cd" {
			for path := range generated {
				if !filepath.IsAbs(path) {
					delete(generated, path)
				}
			}
			if len(args) == 2 {
				path, known := staticWord(args[1])
				if known && path != "-" {
					if filepath.IsAbs(path) {
						dir = path
					} else if dir != "" {
						dir = filepath.Join(dir, path)
					}
				} else {
					dir = ""
				}
			} else {
				dir = ""
			}
			return true
		}
		if base == "source" || base == "." {
			if len(args) > 1 {
				before := len(refs)
				add(args[1], false)
				if len(refs) > before {
					index := len(refs) - 1
					ref := refs[index]
					// Reuse the same bytes for environment analysis and the model's
					// snapshot; a second filesystem read could see a newer version.
					if ref.problem == "" && !ref.inline {
						raw, err := readScript(ref.path)
						if err != nil {
							ref.problem = err.Error()
						} else {
							ref.inline = true
							ref.content = string(raw)
						}
					}
					refs[index] = ref
					sourceInputs[call] = reviewSourceInput{ref: ref}
				}
			} else {
				refs = append(refs, scriptReference{problem: "source missing script path"})
			}
			return true
		}
		if scriptInterpreter.MatchString(base) {
			shell := base == "bash" || base == "sh" || base == "zsh" || base == "fish"
			for i := 1; i < len(args); i++ {
				value, literal := staticWord(args[i])
				if !literal {
					add(args[i], false)
					return true
				}
				inlineOption := (!shell && (value == "-c" || value == "-e" || value == "--eval" || value == "--print")) || (shell && (value == "--command" || (strings.HasPrefix(value, "-") && !strings.HasPrefix(value, "--") && strings.Contains(value, "c"))))
				if inlineOption {
					if shell && i+1 < len(args) {
						variants, known := inlineShellWords(args[i+1], inlineBindingsAt(args[i+1], bindings, constants))
						if known {
							for _, inline := range variants {
								refs = append(refs, scriptReferencesBounded(inline, dir, depth+1, remaining)...)
							}
						} else {
							refs = append(refs, scriptReference{problem: "inline shell execution content is indeterminate"})
						}
					} else if shell {
						refs = append(refs, scriptReference{problem: "inline shell execution is missing command content"})
					}
					return true
				}
				if interpreterInfoOnly(base, value) {
					return true
				}
				if value == "-m" && strings.HasPrefix(base, "python") {
					if pythonModuleShellEnvironmentUnknown(tree, call, name, sourceInputs, constants, &envAnalysisRemaining) {
						refs = append(refs, scriptReference{problem: "preceding shell operations may have altered Python module environment, cannot confirm source entry"})
						return true
					}
					if i+1 >= len(args) {
						refs = append(refs, scriptReference{problem: "Python module execution missing module name"})
						return true
					}
					module, known := staticWord(args[i+1])
					if !known {
						if fields, ok := constantWordFields(args[i+1], constants); ok && len(fields) == 1 {
							module, known = fields[0], true
						}
					}
					if !known {
						refs = append(refs, scriptReference{problem: "Python module name contains dynamic expansion"})
						return true
					}
					refs = append(refs, pythonModuleReferences(name, module, dir, args, wrappedArgs[:len(wrappedArgs)-len(args)], call, generated, constants)...)
					return true
				}
				if value == "-" {
					break
				}
				if value == "--" {
					if i+1 < len(args) {
						add(args[i+1], false)
						return true
					}
					break
				}
				if shell && (value == "--rcfile" || value == "--init-file") {
					if i+1 < len(args) {
						add(args[i+1], false)
						i++
						continue
					}
					refs = append(refs, scriptReference{problem: "shell init option missing script path"})
					return true
				}
				if strings.HasPrefix(value, "-") {
					if value == "-W" || value == "-X" || value == "--require" || value == "-r" || (shell && (value == "-o" || value == "-O")) {
						i++
					}
					continue
				}
				add(args[i], false)
				return true
			}
			for _, r := range stmt.Redirs {
				if r.Op == syntax.RdrIn {
					add(r.Word, false)
					return true
				}
				if r.Hdoc != nil || r.Op == syntax.Hdoc || r.Op == syntax.DashHdoc || r.Op == syntax.WordHdoc {
					return true
				}
			}
			if len(args) > 1 || len(tree.Stmts) > 1 || strings.Contains(command, "|") {
				refs = append(refs, scriptReference{problem: "interpreter reads code from external stdin, cannot obtain file snapshot"})
			}
			return true
		}
		if strings.Contains(name, "/") || knownScriptExtension(name) {
			add(args[0], true)
		}
		return true
	})
	return refs
}
func knownScriptExtension(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".sh", ".bash", ".zsh", ".fish", ".py", ".py3", ".js", ".mjs", ".cjs", ".ts", ".tsx", ".rb", ".pl", ".php", ".ps1", ".lua":
		return true
	}
	return false
}

func captureScripts(workingDir, tool string, arguments json.RawMessage) ([]ReviewScript, string) {
	switch tool {
	case "Bash", "shell_open", "shell_send", "command", "script":
	default:
		return nil, "not_applicable"
	}
	var fields map[string]any
	if json.Unmarshal(arguments, &fields) != nil {
		return nil, "not_applicable"
	}
	key := "command"
	if tool == "shell_send" {
		key = "text"
	}
	command, _ := fields[key].(string)
	if strings.TrimSpace(command) == "" {
		for _, candidate := range []string{"script", "code", "cmd"} {
			if value, ok := fields[candidate].(string); ok {
				command = value
				break
			}
		}
	}
	if strings.TrimSpace(command) == "" {
		return nil, "not_applicable"
	}
	refs := scriptReferences(command, workingDir, 0)
	var files []ReviewScript
	seen := map[string]bool{}
	status := "complete"
	total := 0
	for _, ref := range refs {
		key := ref.path + ref.problem
		if ref.inline {
			hash := sha256.Sum256([]byte(ref.content))
			key += fmt.Sprintf("%x", hash)
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		file := ReviewScript{Path: ref.path, ReadError: ref.problem}
		if ref.generated {
			file.Source = "command"
		} else if ref.path != "" {
			file.Source = "file"
		}
		if file.Path == "" {
			file.Path = "<undetermined script>"
		}
		if ref.problem != "" {
			file.Truncated = strings.Contains(ref.problem, "exceeds")
			status = "incomplete"
			files = append(files, file)
			continue
		}
		if len(files) >= 8 {
			files = append(files, ReviewScript{Path: file.Path, ReadError: "this call exceeds 8 script files"})
			status = "incomplete"
			break
		}
		content := []byte(ref.content)
		var err error
		if !ref.inline {
			content, err = readScript(ref.path)
		}
		if err != nil {
			file.ReadError = err.Error()
			if strings.Contains(file.ReadError, "exceeds") {
				file.Truncated = true
			}
			status = "incomplete"
			files = append(files, file)
			continue
		}
		file.Bytes = int64(len(content))
		// A directly invoked native binary is not a script. Interpreters, however,
		// must never be permitted to execute an opaque/non-text file automatically.
		if ref.direct && (bytes.HasPrefix(content, []byte{0x7f, 'E', 'L', 'F'}) || bytes.HasPrefix(content, []byte{'M', 'Z'})) {
			continue
		}
		if tool == "shell_send" {
			file.ReadError = "interactive session may have changed directory or connected to a remote host; cannot confirm local file is the actual executed script"
			status = "incomplete"
			files = append(files, file)
			continue
		}
		if !utf8.Valid(content) || bytes.ContainsRune(content, 0) {
			file.ReadError = "script is not valid UTF-8 text"
			status = "incomplete"
			files = append(files, file)
			continue
		}
		if len(content) > maxReviewScriptBytes || total+len(content) > maxReviewScriptBytes {
			file.ReadError = "script exceeds total snapshot limit (1 MiB); partial content not attached"
			file.Truncated = true
			status = "incomplete"
			files = append(files, file)
			continue
		}
		hash := sha256.Sum256(content)
		file.SHA256 = fmt.Sprintf("%x", hash)
		file.Content = string(content)
		file.Complete = true
		total += len(content)
		files = append(files, file)
	}
	if len(files) == 0 {
		return nil, "not_applicable"
	}
	return files, status
}
func readScript(path string) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("not a regular file")
	}
	if info.Size() > maxReviewScriptBytes {
		return nil, fmt.Errorf("script exceeds snapshot limit (1 MiB)")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(io.LimitReader(f, maxReviewScriptBytes+1))
}
