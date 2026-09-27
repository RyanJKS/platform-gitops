// Render the repository's explicit list + Git-file ApplicationSet matrix offline.
// This validates templates with Go/Sprig, not controller adoption or reconciliation.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	jsonpatch "github.com/evanphx/json-patch/v5"
	"sigs.k8s.io/yaml"
)

type object = map[string]interface{}
type appSet struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Metadata   object `json:"metadata"`
	Spec       struct {
		GoTemplate        bool     `json:"goTemplate"`
		GoTemplateOptions []string `json:"goTemplateOptions"`
		Generators        []struct {
			Matrix struct {
				Generators []struct {
					List *struct {
						Elements []object `json:"elements"`
					} `json:"list"`
					Git *struct {
						RepoURL  string `json:"repoURL"`
						Revision string `json:"revision"`
						Files    []struct {
							Path string `json:"path"`
						} `json:"files"`
						Values map[string]string `json:"values"`
					} `json:"git"`
				} `json:"generators"`
			} `json:"matrix"`
		} `json:"generators"`
		SyncPolicy    object `json:"syncPolicy"`
		Template      object `json:"template"`
		TemplatePatch string `json:"templatePatch"`
	} `json:"spec"`
}

func render(s string, params object) (string, error) {
	funcs := sprig.TxtFuncMap()
	for _, name := range []string{"env", "expandenv", "getHostByName"} {
		delete(funcs, name)
	}
	t, err := template.New("application").Funcs(funcs).Option("missingkey=error").Parse(s)
	if err != nil {
		return "", err
	}
	var out bytes.Buffer
	err = t.Execute(&out, params)
	return out.String(), err
}

func renderFields(value interface{}, params object) (interface{}, error) {
	switch v := value.(type) {
	case string:
		return render(v, params)
	case map[string]interface{}:
		out := object{}
		for k, x := range v {
			r, e := renderFields(x, params)
			if e != nil {
				return nil, e
			}
			out[k] = r
		}
		return out, nil
	case []interface{}:
		out := make([]interface{}, len(v))
		for i, x := range v {
			r, e := renderFields(x, params)
			if e != nil {
				return nil, e
			}
			out[i] = r
		}
		return out, nil
	default:
		return value, nil
	}
}

func readYAML(path string, target interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.UnmarshalStrict(data, target)
}

func expand(set appSet, repo, cluster string) ([]object, error) {
	if !set.Spec.GoTemplate || len(set.Spec.GoTemplateOptions) != 1 || set.Spec.GoTemplateOptions[0] != "missingkey=error" || len(set.Spec.Generators) != 1 {
		return nil, fmt.Errorf("unsupported ApplicationSet template or generators")
	}
	matrix := set.Spec.Generators[0].Matrix.Generators
	if len(matrix) != 2 || matrix[0].List == nil || matrix[1].Git == nil {
		return nil, fmt.Errorf("expected explicit list + Git-file matrix")
	}
	git := matrix[1].Git
	if git.RepoURL != "https://github.com/RyanJKS/platform-gitops.git" || git.Revision != "main" || len(git.Files) != 1 || git.Files[0].Path != "catalog/platform/components/{{ .component }}.yaml" || len(git.Values) != 1 || git.Values["clusterPath"] != cluster {
		return nil, fmt.Errorf("generator must select this cluster and exact catalog component files")
	}
	var apps []object
	seen := map[string]bool{}
	for _, element := range matrix[0].List.Elements {
		component, ok := element["component"].(string)
		if !ok || component == "" || strings.ContainsAny(component, "/.\\*?[]") {
			return nil, fmt.Errorf("invalid explicit component: %v", element["component"])
		}
		if seen[component] {
			return nil, fmt.Errorf("duplicate component %s", component)
		}
		seen[component] = true
		params := object{}
		if err := readYAML(filepath.Join(repo, "catalog/platform/components", component+".yaml"), &params); err != nil {
			return nil, err
		}
		for k, v := range element {
			if k != "component" && k != "overrideValues" && k != "versionOverride" && k != "namespaceOverride" {
				return nil, fmt.Errorf("unsupported component override %s", k)
			}
			if _, exists := params[k]; exists {
				return nil, fmt.Errorf("matrix parameter collision: %s", k)
			}
			params[k] = v
		}
		if overrides, ok := params["overrideValues"].([]interface{}); ok {
			for _, v := range overrides {
				path, ok := v.(string)
				if !ok || filepath.IsAbs(path) || strings.Contains(path, "..") {
					return nil, fmt.Errorf("override must stay within cluster")
				}
				if _, err := os.Stat(filepath.Join(repo, cluster, path)); err != nil {
					return nil, err
				}
			}
		}
		params["values"] = object{"clusterPath": cluster}
		rendered, err := renderFields(set.Spec.Template, params)
		if err != nil {
			return nil, err
		}
		base, err := json.Marshal(rendered)
		if err != nil {
			return nil, err
		}
		patch, err := render(set.Spec.TemplatePatch, params)
		if err != nil {
			return nil, err
		}
		patchJSON, err := yaml.YAMLToJSON([]byte(patch))
		if err != nil {
			return nil, err
		}
		merged, err := jsonpatch.MergePatch(base, patchJSON)
		if err != nil {
			return nil, err
		}
		app := object{}
		if err = json.Unmarshal(merged, &app); err != nil {
			return nil, err
		}
		app["apiVersion"] = "argoproj.io/v1alpha1"
		app["kind"] = "Application"
		apps = append(apps, app)
	}
	if len(apps) == 0 {
		return nil, fmt.Errorf("cluster selects no components")
	}
	return apps, nil
}

func run() error {
	if len(os.Args) != 4 {
		return fmt.Errorf("usage: render-applications REPO CLUSTER APPLICATIONSET.yaml")
	}
	var set appSet
	if err := readYAML(os.Args[3], &set); err != nil {
		return err
	}
	apps, err := expand(set, os.Args[1], filepath.ToSlash(os.Args[2]))
	if err != nil {
		return err
	}
	for _, app := range apps {
		data, err := yaml.Marshal(app)
		if err != nil {
			return err
		}
		fmt.Printf("---\n%s", data)
	}
	return nil
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
