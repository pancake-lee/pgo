package genCURD

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var (
	serviceDeclPattern = regexp.MustCompile(`\bservice\s+([A-Za-z_][A-Za-z0-9_]*)\s*\{`)
	tableOptionPattern = regexp.MustCompile(`\boption\s+\(pgo\.tables\)\s*=\s*"([^"]+)"\s*;`)
)

func readTableToServiceMap(tableNameList []string) (map[string]string, error) {
	knownTableMap := make(map[string]struct{}, len(tableNameList))
	for _, tableName := range tableNameList {
		knownTableMap[tableName] = struct{}{}
	}

	tableToServiceMap := make(map[string]string)
	err := filepath.Walk("proto", func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			if os.IsNotExist(walkErr) {
				return nil
			}
			return walkErr
		}
		if info.IsDir() || filepath.Ext(path) != ".proto" || path == pbTplPath {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		serviceToTableListMap, err := parseProtoTableMappings(string(content), path)
		if err != nil {
			return err
		}
		if len(serviceToTableListMap) > 0 && !strings.Contains(string(content), `import "pgo/options.proto";`) {
			return fmt.Errorf("proto mapping %s must import \"pgo/options.proto\"", path)
		}
		for serviceName, mappedTableList := range serviceToTableListMap {
			for _, tableName := range mappedTableList {
				if _, ok := knownTableMap[tableName]; !ok {
					return fmt.Errorf("proto mapping %s declares unknown table %q", path, tableName)
				}
				if previousService, ok := tableToServiceMap[tableName]; ok {
					return fmt.Errorf("table %q is mapped to both %q and %q", tableName, previousService, serviceName)
				}
				tableToServiceMap[tableName] = serviceName
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return tableToServiceMap, nil
}

func parseProtoTableMappings(content, path string) (map[string][]string, error) {
	serviceToTableListMap := make(map[string][]string)
	matchList := serviceDeclPattern.FindAllStringSubmatchIndex(content, -1)
	for _, match := range matchList {
		serviceName := content[match[2]:match[3]]
		serviceBody, err := readProtoBlock(content, match[1]-1)
		if err != nil {
			return nil, fmt.Errorf("parse service %q in %s: %w", serviceName, path, err)
		}
		mappedOptionList := tableOptionPattern.FindAllStringSubmatch(serviceBody, -1)
		for _, option := range mappedOptionList {
			tableName := strings.TrimSpace(option[1])
			if tableName == "" {
				return nil, fmt.Errorf("service %q in %s declares an empty pgo.tables option", serviceName, path)
			}
			serviceToTableListMap[serviceNameToOutputName(serviceName)] = append(
				serviceToTableListMap[serviceNameToOutputName(serviceName)], tableName)
		}
	}
	return serviceToTableListMap, nil
}

func readProtoBlock(content string, openBraceOffset int) (string, error) {
	depth := 0
	for i := openBraceOffset; i < len(content); i++ {
		switch content[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return content[openBraceOffset+1 : i], nil
			}
		}
	}
	return "", fmt.Errorf("unclosed block")
}

func serviceNameToOutputName(serviceName string) string {
	serviceName = strings.TrimSuffix(serviceName, "CURD")
	if serviceName == "" {
		return "default"
	}
	return strings.ToLower(serviceName[:1]) + serviceName[1:]
}

func tableNameListForService(tblToSvrMap map[string]*Table, serviceName string) []string {
	var tableNameList []string
	for tableName, tbl := range tblToSvrMap {
		if tbl.ServiceName == serviceName {
			tableNameList = append(tableNameList, tableName)
		}
	}
	sort.Strings(tableNameList)
	return tableNameList
}
