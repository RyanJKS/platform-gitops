package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"sigs.k8s.io/yaml"
)

func clusterSet(t *testing.T, cluster string) appSet {
	t.Helper()
	data, err := exec.Command("kubectl", "kustomize", filepath.Join("../..", cluster, "argocd")).Output()
	if err != nil {
		t.Fatal(err)
	}
	for _, doc := range strings.Split(string(data), "\n---\n") {
		var kind struct {
			Kind string `json:"kind"`
		}
		if err := yaml.Unmarshal([]byte(doc), &kind); err != nil {
			t.Fatal(err)
		}
		if kind.Kind == "ApplicationSet" {
			var set appSet
			if err := yaml.UnmarshalStrict([]byte(doc), &set); err != nil {
				t.Fatal(err)
			}
			return set
		}
	}
	t.Fatal("no ApplicationSet")
	return appSet{}
}

func TestClusterSelections(t *testing.T) {
	for _, cluster := range []string{
		"clusters/azure/DEV-JKS/dev/eus2/spoke-atlas/aks-shared",
		"clusters/azure/DEV-JKS/dev/eus2/spoke-atlas/aks-atlas-market",
		"clusters/local/kind/kind-platform",
	} {
		t.Run(filepath.Base(cluster), func(t *testing.T) {
			set := clusterSet(t, cluster)
			if set.Spec.SyncPolicy["applicationsSync"] != "create-update" || set.Spec.SyncPolicy["preserveResourcesOnDeletion"] != true {
				t.Fatal("deletion safeguards missing")
			}
			apps, err := expand(set, "../..", cluster)
			if err != nil {
				t.Fatal(err)
			}
			expected := 7
			if strings.Contains(cluster, "/local/") {
				expected = 8
			}
			if len(apps) != expected {
				t.Fatalf("got %d selections, want %d", len(apps), expected)
			}
			for _, app := range apps {
				metadata := app["metadata"].(map[string]interface{})
				spec := app["spec"].(map[string]interface{})
				if metadata["namespace"] != "argocd" || spec["project"] != "platform" {
					t.Fatal("unexpected namespace or project")
				}
				if strings.Contains(cluster, "/azure/") && metadata["name"] == "platform-argocd" {
					t.Fatal("extension ownership violated")
				}
				if spec["destination"].(map[string]interface{})["server"] != "https://kubernetes.default.svc" {
					t.Fatal("unexpected destination")
				}
				if source, ok := spec["source"].(map[string]interface{}); ok {
					path := source["path"].(string)
					if !strings.HasPrefix(path, cluster+"/") {
						t.Fatalf("cross-cluster source: %s", path)
					}
					if _, err := os.Stat(filepath.Join("../..", path, "kustomization.yaml")); err != nil {
						t.Fatal(err)
					}
				}
				if _, ok := metadata["finalizers"]; ok {
					t.Fatal("unexpected deletion finalizer")
				}
				if _, ok := spec["syncPolicy"].(map[string]interface{})["automated"]; ok {
					t.Fatal("auto-sync enabled")
				}
			}
		})
	}
}

func TestOverridesAndInvalidSelections(t *testing.T) {
	cluster := "clusters/local/kind/kind-platform"
	t.Run("version and ordered values", func(t *testing.T) {
		set := clusterSet(t, cluster)
		set.Spec.Generators[0].Matrix.Generators[0].List.Elements = []object{{"component": "monitoring", "versionOverride": "70.4.1", "overrideValues": []interface{}{"platform/monitoring/values.yaml"}}}
		apps, err := expand(set, "../..", cluster)
		if err != nil {
			t.Fatal(err)
		}
		source := apps[0]["spec"].(map[string]interface{})["sources"].([]interface{})[0].(map[string]interface{})
		if source["targetRevision"] != "70.4.1" {
			t.Fatal("version override ignored")
		}
		values := source["helm"].(map[string]interface{})["valueFiles"].([]interface{})
		if len(values) != 2 || values[0] != "$values/catalog/platform/monitoring/values.yaml" || values[1] != "$values/"+cluster+"/platform/monitoring/values.yaml" {
			t.Fatalf("incorrect precedence: %v", values)
		}
	})
	for _, component := range []string{"*", "../argocd", "does-not-exist", ""} {
		t.Run("reject "+component, func(t *testing.T) {
			set := clusterSet(t, cluster)
			set.Spec.Generators[0].Matrix.Generators[0].List.Elements = []object{{"component": component}}
			if _, err := expand(set, "../..", cluster); err == nil {
				t.Fatal("invalid selection accepted")
			}
		})
	}
	t.Run("reject other cluster", func(t *testing.T) {
		if _, err := expand(clusterSet(t, cluster), "../..", "clusters/other"); err == nil {
			t.Fatal("cross-cluster generator accepted")
		}
	})
	t.Run("strict missing keys", func(t *testing.T) {
		if _, err := render("{{ .missing }}", object{}); err == nil {
			t.Fatal("missing parameter accepted")
		}
	})
}
