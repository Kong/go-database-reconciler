package file

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"dario.cat/mergo"
	"github.com/kong/go-database-reconciler/pkg/utils"
	"golang.org/x/term"
	"sigs.k8s.io/yaml"
)

// getContent reads all the YAML and JSON files in the directory or the
// file, depending on the type of each item in filenames, merges the content of
// these files and renders a Content.
func getContent(filenames []string, mockEnvVars bool) (*Content, error) {
	var workspaces, runtimeGroups []string
	var res Content
	var errs []error
	for _, fileOrDir := range filenames {
		readers, err := getReaders(fileOrDir)
		if err != nil {
			return nil, err
		}

		for filename, r := range readers {
			content, err := readContent(r, mockEnvVars)
			if err != nil {
				errs = append(errs, fmt.Errorf("reading file %s: %w", filename, err))
				continue
			}
			if content.Workspace != "" {
				workspaces = append(workspaces, content.Workspace)
			}
			if content.Konnect != nil && len(content.Konnect.RuntimeGroupName) > 0 {
				runtimeGroups = append(runtimeGroups, content.Konnect.RuntimeGroupName)
			}
			err = mergo.Merge(&res, content, mergo.WithAppendSlice)
			if err != nil {
				return nil, fmt.Errorf("merging file contents: %w", err)
			}
		}
	}
	if len(errs) > 0 {
		return nil, utils.ErrArray{Errors: errs}
	}
	if err := validateWorkspaces(workspaces); err != nil {
		return nil, err
	}
	if err := validateRuntimeGroups(runtimeGroups); err != nil {
		return nil, err
	}
	if err := validateEmptyContent(res); err != nil {
		return &Content{}, err
	}
	return &res, nil
}

// getReaders returns back a map of filename:io.Reader representing all the
// YAML and JSON files in a directory. If fileOrDir is a single file, then it
// returns back the reader for the file.
// If fileOrDir is equal to "-" string, then it returns back a io.Reader
// for the os.Stdin file descriptor.
func getReaders(fileOrDir string) (map[string]io.Reader, error) {
	// special case where `-` means stdin
	if fileOrDir == "-" {
		if term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stderr.Fd())) {
			fmt.Fprintf(os.Stderr, "reading input from stdin...\n")
		}
		return map[string]io.Reader{"STDIN": os.Stdin}, nil
	}

	finfo, err := os.Stat(fileOrDir)
	if err != nil {
		return nil, fmt.Errorf("reading state file: %w", err)
	}

	var files []string
	if finfo.IsDir() {
		files, err = utils.ConfigFilesInDir(fileOrDir)
		if err != nil {
			return nil, fmt.Errorf("getting files from directory: %w", err)
		}
	} else {
		files = append(files, fileOrDir)
	}

	res := make(map[string]io.Reader, len(files))
	for _, file := range files {
		f, err := os.Open(file)
		if err != nil {
			return nil, fmt.Errorf("opening file: %w", err)
		}
		res[file] = bufio.NewReader(f)
	}
	return res, nil
}

func hasLeadingSpace(fileContent string) bool {
	if fileContent != "" && string(fileContent[0]) == " " {
		return true
	}
	return false
}

// readContent reads all the byes until io.EOF and unmarshals the read
// bytes into Content.
func readContent(reader io.Reader, mockEnvVars bool) (*Content, error) {
	var err error
	contentBytes, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	renderedContent, err := renderTemplate(string(contentBytes), mockEnvVars)
	if err != nil {
		return nil, fmt.Errorf("parsing file: %w", err)
	}
	// go-yaml implementation fails at correctly parsing a file whose first
	// character is a space, as shown in https://github.com/Kong/deck/issues/578
	// If that is the case here, raise an error.
	if hasLeadingSpace(renderedContent) {
		return nil, fmt.Errorf("file must not begin with a whitespace")
	}
	renderedContentBytes := []byte(renderedContent)
	err = validate(renderedContentBytes)
	if err != nil {
		return nil, fmt.Errorf("validating file content: %w", err)
	}
	var result Content
	err = yamlUnmarshal(renderedContentBytes, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// yamlUnmarshal is a wrapper around yaml.Unmarshal to ensure that the right
// yaml package is in use. Using ghodss/yaml ensures that no
// `map[interface{}]interface{}` is present in go-kong.Plugin.Configuration.
// If it is present, then it leads to a silent error. See Github Issue #144.
// The verification for this is done using a test.
func yamlUnmarshal(bytes []byte, v interface{}) error {
	return yaml.Unmarshal(bytes, v)
}
