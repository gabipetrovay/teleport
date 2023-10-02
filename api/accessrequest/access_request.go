/*
Copyright 2023 Gravitational, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package accessrequest

import (
	"context"
	"fmt"
	"strings"

	"github.com/gravitational/teleport/api/client/proto"
	"github.com/gravitational/teleport/api/types"
	"github.com/gravitational/trace"
)

// FriendlyName will return the friendly name for a resource if it has one. Otherwise, it
// will return an empty string.
func FriendlyName(resource types.ResourceWithLabels) string {
	// Right now, only resources sourced from Okta and nodes have friendly names.
	if resource.Origin() == types.OriginOkta {
		return resource.GetMetadata().Description
	}

	if hn, ok := resource.(interface{ GetHostname() string }); ok {
		return hn.GetHostname()
	}

	return ""
}

// ResourceLister is an interface which can list resources.
type ResourceLister interface {
	ListResources(ctx context.Context, req proto.ListResourcesRequest) (*types.ListResourcesResponse, error)
}

type ListResourcesRequestOption func(*proto.ListResourcesRequest)

func GetResourceDetails(ctx context.Context, clusterName string, lister ResourceLister, ids []types.ResourceID) (map[string]types.ResourceDetails, error) {
	var resourceIDs []types.ResourceID
	for _, resourceID := range ids {
		// We're interested in hostname or friendly name details. These apply to
		// nodes, app servers, and user groups.
		switch resourceID.Kind {
		case types.KindNode, types.KindApp, types.KindUserGroup:
			resourceIDs = append(resourceIDs, resourceID)
		}
	}

	withExtraRoles := func(req *proto.ListResourcesRequest) {
		req.UseSearchAsRoles = true
		req.UsePreviewAsRoles = true
	}

	resources, err := GetResourcesByResourceIDs(ctx, lister, resourceIDs, withExtraRoles)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	result := make(map[string]types.ResourceDetails)
	for _, resource := range resources {
		friendlyName := FriendlyName(resource)

		// No friendly name was found, so skip to the next resource.
		if friendlyName == "" {
			continue
		}

		id := types.ResourceID{
			ClusterName: clusterName,
			Kind:        resource.GetKind(),
			Name:        resource.GetName(),
		}
		result[types.ResourceIDToString(id)] = types.ResourceDetails{
			FriendlyName: friendlyName,
		}
	}

	return result, nil
}

// GetResourceIDsByCluster will return resource IDs grouped by cluster.
func GetResourceIDsByCluster(r types.AccessRequest) map[string][]types.ResourceID {
	resourceIDsByCluster := make(map[string][]types.ResourceID)
	for _, resourceID := range r.GetRequestedResourceIDs() {
		resourceIDsByCluster[resourceID.ClusterName] = append(resourceIDsByCluster[resourceID.ClusterName], resourceID)
	}
	return resourceIDsByCluster
}

func GetResourcesByResourceIDs(ctx context.Context, lister ResourceLister, resourceIDs []types.ResourceID, opts ...ListResourcesRequestOption) ([]types.ResourceWithLabels, error) {
	resourceNamesByKind := make(map[string][]string)
	for _, resourceID := range resourceIDs {
		resourceNamesByKind[resourceID.Kind] = append(resourceNamesByKind[resourceID.Kind], resourceID.Name)
	}
	var resources []types.ResourceWithLabels
	for kind, resourceNames := range resourceNamesByKind {
		req := proto.ListResourcesRequest{
			ResourceType:        MapResourceKindToListResourcesType(kind),
			PredicateExpression: anyNameMatcher(resourceNames),
			Limit:               int32(len(resourceNames)),
		}
		for _, opt := range opts {
			opt(&req)
		}
		resp, err := lister.ListResources(ctx, req)
		if err != nil {
			return nil, trace.Wrap(err)
		}

		for _, result := range resp.Resources {
			leafResources, err := MapListResourcesResultToLeafResource(result, kind)
			if err != nil {
				return nil, trace.Wrap(err)
			}
			resources = append(resources, leafResources...)
		}
	}
	return resources, nil
}

// anyNameMatcher returns a PredicateExpression which matches any of a given list
// of names. Given names will be escaped and quoted when building the expression.
func anyNameMatcher(names []string) string {
	matchers := make([]string, len(names))
	for i := range names {
		matchers[i] = fmt.Sprintf(`resource.metadata.name == %q`, names[i])
	}
	return strings.Join(matchers, " || ")
}

// MapResourceKindToListResourcesType returns the value to use for ResourceType in a
// ListResourcesRequest based on the kind of resource you're searching for.
// Necessary because some resource kinds don't support ListResources directly,
// so you have to list the parent kind. Use MapListResourcesResultToLeafResource to map back
// to the given kind.
func MapResourceKindToListResourcesType(kind string) string {
	switch kind {
	case types.KindApp:
		return types.KindAppServer
	case types.KindDatabase:
		return types.KindDatabaseServer
	case types.KindKubernetesCluster:
		return types.KindKubeServer
	default:
		return kind
	}
}

// MapListResourcesResultToLeafResource is the inverse of
// MapResourceKindToListResourcesType, after the ListResources call it maps the
// result back to the kind we really want. `hint` should be the name of the
// desired resource kind, used to disambiguate normal SSH nodes and kubernetes
// services which are both returned as `types.Server`.
func MapListResourcesResultToLeafResource(resource types.ResourceWithLabels, hint string) (types.ResourcesWithLabels, error) {
	switch r := resource.(type) {
	case types.AppServer:
		return types.ResourcesWithLabels{r.GetApp()}, nil
	case types.KubeServer:
		return types.ResourcesWithLabels{r.GetCluster()}, nil
	case types.DatabaseServer:
		return types.ResourcesWithLabels{r.GetDatabase()}, nil
	case types.Server:
		if hint == types.KindKubernetesCluster {
			return nil, trace.BadParameter("expected kubernetes server, got server")
		}
	default:
	}
	return types.ResourcesWithLabels{resource}, nil
}
