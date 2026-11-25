package operators

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/samber/lo"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"
)

// clusterSnapshot holds a deep copy of every object in a client, keyed by kind, namespace and name.
type clusterSnapshot map[string]runtime.Object

// snapshotCluster lists every object kind registered in scheme, so built-in resources and the
// operator's CRs are all included.
func snapshotCluster(c client.Client, scheme *runtime.Scheme) (clusterSnapshot, error) {
	snapshot := clusterSnapshot{}
	for listGVK := range scheme.AllKnownTypes() {
		itemGVK := listGVK.GroupVersion().WithKind(strings.TrimSuffix(listGVK.Kind, "List"))
		if itemGVK.Kind == listGVK.Kind || !isObjectKind(scheme, itemGVK) {
			continue // not a list of API objects
		}

		list, err := scheme.New(listGVK)
		if err != nil {
			return nil, fmt.Errorf("create list for %s: %w", listGVK, err)
		}
		if err := c.List(context.Background(), list); err != nil {
			return nil, fmt.Errorf("list %s: %w", listGVK, err)
		}
		items, err := meta.ExtractList(list)
		if err != nil {
			return nil, fmt.Errorf("extract items of %s: %w", listGVK, err)
		}

		for _, item := range items {
			obj := item.DeepCopyObject()
			accessor, err := meta.Accessor(obj)
			if err != nil {
				return nil, fmt.Errorf("access metadata of %s: %w", itemGVK, err)
			}
			// Bumped by the fake client on every write, so it isn't part of the cluster state
			accessor.SetResourceVersion("")
			snapshot[fmt.Sprintf("%s %s/%s", itemGVK, accessor.GetNamespace(), accessor.GetName())] = obj
		}
	}

	return snapshot, nil
}

func isObjectKind(scheme *runtime.Scheme, gvk schema.GroupVersionKind) bool {
	if !scheme.Recognizes(gvk) {
		return false
	}
	obj, err := scheme.New(gvk)
	if err != nil {
		return false
	}
	_, err = meta.Accessor(obj)
	return err == nil
}

// copyClient returns a new fake client holding a deep copy of everything in src.
func copyClient(src client.Client, scheme *runtime.Scheme) (client.Client, error) {
	snapshot, err := snapshotCluster(src, scheme)
	if err != nil {
		return nil, err
	}
	return fake.NewFakeClientWithScheme(scheme, lo.Values(snapshot)...), nil
}

// diffClusters describes how actual differs from expected, or returns nil if they hold the same objects.
func diffClusters(scheme *runtime.Scheme, expected, actual client.Client) ([]string, error) {
	expectedSnapshot, err := snapshotCluster(expected, scheme)
	if err != nil {
		return nil, fmt.Errorf("snapshot expected cluster: %w", err)
	}
	actualSnapshot, err := snapshotCluster(actual, scheme)
	if err != nil {
		return nil, fmt.Errorf("snapshot actual cluster: %w", err)
	}

	keys := lo.Union(lo.Keys(expectedSnapshot), lo.Keys(actualSnapshot))
	sort.Strings(keys)

	var diffs []string
	for _, key := range keys {
		expectedObj, inExpected := expectedSnapshot[key]
		actualObj, inActual := actualSnapshot[key]
		switch {
		case !inActual:
			diffs = append(diffs, fmt.Sprintf("%s: missing", key))
		case !inExpected:
			diffs = append(diffs, fmt.Sprintf("%s: unexpected", key))
		case !equality.Semantic.DeepEqual(expectedObj, actualObj):
			diffs = append(diffs, fmt.Sprintf("%s: differs\nexpected:\n%s\nactual:\n%s", key, toYAML(expectedObj), toYAML(actualObj)))
		}
	}

	return diffs, nil
}

func toYAML(obj runtime.Object) string {
	out, err := yaml.Marshal(obj)
	if err != nil {
		return fmt.Sprintf("%+v", obj)
	}
	return string(out)
}
