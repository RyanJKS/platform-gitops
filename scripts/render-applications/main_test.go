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
		"clusters/azure/DEV-JKS/dev/eus2/spoke-atlas/aks-atlas-market",
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
	cluster := "clusters/azure/DEV-JKS/dev/eus2/spoke-atlas/aks-atlas-market"
	t.Run("version and ordered values", func(t *testing.T) {
		repo := t.TempDir()
		component, err := os.ReadFile("../../catalog/platform/components/monitoring.yaml")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(repo, "catalog/platform/components"), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(repo, "catalog/platform/components/monitoring.yaml"), component, 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(repo, cluster, "platform/monitoring"), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(repo, cluster, "platform/monitoring/values.yaml"), []byte("{}\n"), 0644); err != nil {
			t.Fatal(err)
		}

		set := clusterSet(t, cluster)
		set.Spec.Generators[0].Matrix.Generators[0].List.Elements = []object{{"component": "monitoring", "versionOverride": "70.4.1", "overrideValues": []interface{}{"platform/monitoring/values.yaml"}}}
		apps, err := expand(set, repo, cluster)
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

// Roots owned by Terraform have no bootstrap/root.yaml, but still need validation.
func TestSharedClusterRoots(t *testing.T) {
	for _, region := range []string{"uks", "eus2"} {
		t.Run(region, func(t *testing.T) {
			path := "clusters/azure/DEV-JKS/dev/" + region + "/spoke-atlas/aks-shared/argocd"
			data, err := exec.Command("kubectl", "kustomize", "../../"+path).Output()
			if err != nil {
				t.Fatal(err)
			}
			apps := map[string]object{}
			projects := map[string]bool{}
			for _, doc := range strings.Split(string(data), "\n---\n") {
				var resource object
				if err := yaml.Unmarshal([]byte(doc), &resource); err != nil {
					t.Fatal(err)
				}
				name := resource["metadata"].(map[string]interface{})["name"].(string)
				switch resource["kind"] {
				case "Application":
					apps[name] = resource
				case "AppProject":
					projects[name] = true
				default:
					t.Fatalf("unexpected root resource: %v", resource["kind"])
				}
			}
			if apps["chaos-generator"] == nil {
				t.Fatal("chaos-generator missing")
			}
			if region == "eus2" {
				if len(apps) != 1 || len(projects) != 0 {
					t.Fatal("East US 2 selections changed")
				}
				return
			}
			if len(apps) != 2 || !projects["cert-manager"] {
				t.Fatal("UK South platform selection incorrect")
			}
			cert := apps["platform-cert-manager"]
			metadata := cert["metadata"].(map[string]interface{})
			if _, ok := metadata["finalizers"]; ok {
				t.Fatal("unexpected cascading deletion")
			}
			spec := cert["spec"].(map[string]interface{})
			if spec["project"] != "cert-manager" {
				t.Fatal("wrong project")
			}
			sources := spec["sources"].([]interface{})
			if len(sources) != 2 {
				t.Fatal("chart and issuer must share one Application")
			}
			chart := sources[0].(map[string]interface{})
			git := sources[1].(map[string]interface{})
			if chart["chart"] != "cert-manager" || chart["targetRevision"] != "v1.20.4" || git["path"] != "catalog/platform/cert-manager/issuers" || git["ref"] != "values" {
				t.Fatal("chart/issuer source contract changed")
			}
			policy := spec["syncPolicy"].(map[string]interface{})["automated"].(map[string]interface{})
			if policy["prune"] != false || policy["selfHeal"] != true {
				t.Fatal("unsafe reconciliation policy")
			}
		})
	}
}

func TestLocalKindRoot(t *testing.T) {
	cluster := "../../clusters/local/kind/kind-platform"
	entries, err := os.ReadDir(cluster)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || entries[0].Name() != "applications" || entries[1].Name() != "argocd" {
		t.Fatal("kind must contain only applications/ and argocd/")
	}
	data, err := exec.Command("kubectl", "kustomize", cluster+"/argocd").Output()
	if err != nil {
		t.Fatal(err)
	}
	var app object
	if err := yaml.UnmarshalStrict(data, &app); err != nil {
		t.Fatal(err)
	}
	if app["kind"] != "Application" || app["metadata"].(map[string]interface{})["name"] != "platform-argocd" {
		t.Fatal("kind must select only its existing Argo CD Application")
	}
	spec := app["spec"].(map[string]interface{})
	if spec["project"] != "default" {
		t.Fatal("kind still depends on removed platform project")
	}
	if _, ok := app["metadata"].(map[string]interface{})["finalizers"]; ok {
		t.Fatal("unexpected cascading deletion")
	}
	if _, ok := spec["syncPolicy"].(map[string]interface{})["automated"]; ok {
		t.Fatal("unexpected automatic sync")
	}
	var root object
	if err := readYAML("../../bootstrap/local-kind.yaml", &root); err != nil {
		t.Fatal(err)
	}
	rootSpec := root["spec"].(map[string]interface{})
	if rootSpec["source"].(map[string]interface{})["path"] != "clusters/local/kind/kind-platform/argocd" {
		t.Fatal("local-kind root points to the wrong entry point")
	}
}
