package main

import (
	"os"
	"os/exec"
	"sigs.k8s.io/yaml"
	"strings"
	"testing"
)

type object = map[string]interface{}

func readYAML(path string, target interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.UnmarshalStrict(data, target)
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

func TestMarketRoot(t *testing.T) {
	data, err := exec.Command("kubectl", "kustomize", "../../clusters/azure/DEV-JKS/dev/eus2/spoke-atlas/aks-atlas-market/argocd").CombinedOutput()
	if err != nil {
		t.Fatalf("%v: %s", err, data)
	}
	seen := map[string]bool{}
	for _, doc := range strings.Split(string(data), "\n---\n") {
		var resource object
		if err := yaml.UnmarshalStrict([]byte(doc), &resource); err != nil {
			t.Fatal(err)
		}
		kind := resource["kind"].(string)
		name := resource["metadata"].(map[string]interface{})["name"].(string)
		seen[kind+"/"+name] = true
		if kind == "Application" {
			spec := resource["spec"].(map[string]interface{})
			if spec["project"] != "atlas-market" || spec["destination"].(map[string]interface{})["namespace"] != "atlas-market" {
				t.Fatal("workload permissions changed")
			}
			if _, ok := spec["syncPolicy"].(map[string]interface{})["automated"]; ok {
				t.Fatal("unexpected automatic sync")
			}
		}
	}
	for _, expected := range []string{"Application/atlas-market-api", "Application/atlas-market-worker", "AppProject/atlas-market", "Namespace/atlas-market"} {
		if !seen[expected] {
			t.Fatalf("missing %s", expected)
		}
	}
	if len(seen) != 4 {
		t.Fatalf("unexpected market selections: %v", seen)
	}
}
