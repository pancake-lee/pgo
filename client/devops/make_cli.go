package devops

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/pancake-lee/pgo/pkg/pconfig"
	"github.com/pancake-lee/pgo/pkg/putil"
)

type MakeTarget struct {
	Name string
	Msg  string
}

type MakeVar struct {
	Name  string
	Value string
}

func MakeCli() {
	makefile := "Makefile"
	if _, err := os.Stat(makefile); os.IsNotExist(err) {
		// Try parent directory
		if _, err := os.Stat("../Makefile"); err == nil {
			makefile = "../Makefile"
		} else {
			putil.Interact.Errorf("Makefile not found in current directory or parent")
			return
		}
	}

	targets, vars := parseMakefile(makefile)
	if len(targets) == 0 {
		putil.Interact.Errorf("No targets found in %s", makefile)
		return
	}

	// 1. Configure Variables (First)
	currentVars := make(map[string]string)
	cachePath := pconfig.GetDefaultCachePath()

	if len(vars) > 0 {
		putil.Interact.Infof("Configure Variables (Press Enter to use Default)")
		for _, v := range vars {

			// 先赋值为默认值
			val := v.Value

			// 尝试从缓存读取，覆盖默认值
			key := fmt.Sprintf("client.make.%s", v.Name)
			cachedVal := pconfig.GetCacheValue(cachePath, key)
			if cachedVal != "" {
				val = cachedVal
				currentVars[v.Name] = val
			}

			prompt := fmt.Sprintf("%s (Default: %s)", v.Name, val)
			input := putil.Interact.Input(prompt)

			if input != "" {
				val = input
				currentVars[v.Name] = val
				pconfig.SetCacheValue(cachePath, key, val)
			}
		}
	}

	// 2. Select Target
	sel := putil.Interact.NewSelector(fmt.Sprintf("Select Make Target (%s)", makefile))

	for _, t := range targets {
		tName := t.Name // capture
		desc := tName
		if t.Msg != "" {
			desc = fmt.Sprintf("%-20s # %s", tName, t.Msg)
		}
		sel.Reg(desc, func() {
			// 3. Assemble Command
			args := []string{tName}
			for name, val := range currentVars {
				args = append(args, fmt.Sprintf("%s=%s", name, val))
			}
			putil.Interact.Infof("Executing: make %s", putil.StrListToStr(args, " "))

			// 4. Execute Command
			out, err := putil.Exec("make", args...)
			putil.Interact.Infof("%s", out)
			if err != nil {
				putil.Interact.Errorf("Execution failed: %v", err)
			}

			putil.Interact.Input("Execution completed. Press Enter to continue...")
		})
	}

	sel.Loop()
}

func parseMakefile(path string) ([]MakeTarget, []MakeVar) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil
	}
	defer file.Close()

	var targets []MakeTarget
	var vars []MakeVar

	reTarget := regexp.MustCompile(`^([a-zA-Z0-9_-]+):`)
	reVar := regexp.MustCompile(`^([a-zA-Z0-9_-]+)\?=(.*)`)

	scanner := bufio.NewScanner(file)
	var lastComment string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "#") {
			lastComment = strings.TrimSpace(strings.TrimPrefix(line, "#"))
			continue
		}

		if line == "" {
			continue // Maintain comment across empty lines
		}
		targetMsg := lastComment
		lastComment = ""

		if matches := reTarget.FindStringSubmatch(line); len(matches) > 1 {
			t := matches[1]
			// Exclude variable assignment using :=
			if strings.HasPrefix(line[len(matches[0]):], "=") {
				continue
			}
			if t == ".PHONY" {
				continue
			}

			targets = append(targets, MakeTarget{Name: t, Msg: targetMsg})

		} else if matches := reVar.FindStringSubmatch(line); len(matches) > 1 {
			v := matches[1]
			val := strings.TrimSpace(matches[2])
			vars = append(vars, MakeVar{Name: v, Value: val})
		}
	}
	return targets, vars
}
