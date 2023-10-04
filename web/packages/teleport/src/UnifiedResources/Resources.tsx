/*
Copyright 2019-2022 Gravitational, Inc.

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

import React, { useEffect, useState } from 'react';

import styled from 'styled-components';
import {
  Box,
  Flex,
  ButtonLink,
  ButtonSecondary,
  Text,
  ButtonBorder,
  Popover,
} from 'design';
import { Magnifier, PushPin } from 'design/Icon';

import { Danger } from 'design/Alert';

import { TextIcon } from 'teleport/Discover/Shared';

import {
  FeatureBox,
  FeatureHeader,
  FeatureHeaderTitle,
} from 'teleport/components/Layout';
import Empty, { EmptyStateInfo } from 'teleport/components/Empty';
import useTeleport from 'teleport/useTeleport';
import cfg from 'teleport/config';
import history from 'teleport/services/history/history';
import localStorage from 'teleport/services/localStorage';
import useStickyClusterId from 'teleport/useStickyClusterId';
import AgentButtonAdd from 'teleport/components/AgentButtonAdd';
import { SearchResource } from 'teleport/Discover/SelectResource';
import { useUrlFiltering, useInfiniteScroll } from 'teleport/components/hooks';
import { UnifiedResource } from 'teleport/services/agents';
import { useUser } from 'teleport/User/UserContext';
import { encodeUrlQueryParams } from 'teleport/components/hooks/useUrlFiltering';

import { ResourceCard, LoadingCard } from './ResourceCard';
import SearchPanel from './SearchPanel';
import { FilterPanel } from './FilterPanel';
import './unifiedStyles.css';

function mergeResourceIds(arr1: string[], arr2: string[]) {
  const mergedArray = [...arr1];
  for (const item of arr2) {
    if (!mergedArray.includes(item)) {
      mergedArray.push(item);
    }
  }
  return mergedArray;
}

const RESOURCES_MAX_WIDTH = '1800px';
// get 48 resources to start
const INITIAL_FETCH_SIZE = 48;
// increment by 24 every fetch
const FETCH_MORE_SIZE = 24;

const loadingCardArray = new Array(FETCH_MORE_SIZE).fill(undefined);

const tabs: { label: string; value: string }[] = [
  {
    label: 'All Resources',
    value: 'all',
  },
  {
    label: 'Pinned Resources',
    value: 'pinned',
  },
];

export function Resources() {
  const { isLeafCluster, clusterId } = useStickyClusterId();
  const enabled = localStorage.areUnifiedResourcesEnabled();
  const teleCtx = useTeleport();
  const canCreate = teleCtx.storeUser.getTokenAccess().create;
  const [selectedResources, setSelectedResources] = useState<string[]>([]);

  const { updateClusterPreferences, clusterPreferences } = useUser();
  const pinnedResources = clusterPreferences.pinnedResources || [];

  useEffect(() => {
    const handleKeyDown = event => {
      if (event.key === 'Escape') {
        setSelectedResources([]);
      }
    };

    document.addEventListener('keydown', handleKeyDown);

    return () => {
      document.removeEventListener('keydown', handleKeyDown);
    };
  }, []);

  const handlePinResource = (resourceId: string) => {
    if (pinnedResources.includes(resourceId)) {
      updateClusterPreferences({
        pinnedResources: pinnedResources.filter(i => i !== resourceId),
      });
      return;
    }
    updateClusterPreferences({
      pinnedResources: [...pinnedResources, resourceId],
    });
  };

  // if every selected resource is already pinned, the bulk action
  // should be to unpin those resources
  const shouldUnpin = selectedResources.every(resource =>
    pinnedResources.includes(resource)
  );

  const handleSelectResources = (resourceId: string) => {
    if (selectedResources.includes(resourceId)) {
      setSelectedResources(selectedResources.filter(i => i !== resourceId));
      return;
    }
    setSelectedResources([...selectedResources, resourceId]);
  };

  const handlePinSelected = (unpin: boolean) => {
    let newPinned = [];
    if (unpin) {
      newPinned = pinnedResources.filter(i => !selectedResources.includes(i));
    } else {
      newPinned = mergeResourceIds(
        [...pinnedResources],
        [...selectedResources]
      );
    }

    updateClusterPreferences({
      pinnedResources: newPinned,
    });
  };

  const { params, setParams, replaceHistory, pathname, setSort, onLabelClick } =
    useUrlFiltering({
      fieldName: 'name',
      dir: 'ASC',
    });

  const {
    setTrigger: setScrollDetector,
    forceFetch,
    resources,
    attempt,
  } = useInfiniteScroll({
    fetchFunc: teleCtx.resourceService.fetchUnifiedResources,
    clusterId,
    filter: params,
    initialFetchSize: INITIAL_FETCH_SIZE,
    fetchMoreSize: FETCH_MORE_SIZE,
  });

  const noResults = attempt.status === 'success' && resources.length === 0;

  const [isSearchEmpty, setIsSearchEmpty] = useState(true);

  // Using a useEffect for this prevents the "Add your first resource" component from being
  // shown for a split second when making a search after a search that yielded no results.
  useEffect(() => {
    setIsSearchEmpty(!params?.query && !params?.search);
  }, [params.query, params.search]);

  if (!enabled) {
    history.replace(cfg.getNodesRoute(clusterId));
  }

  const onRetryClicked = () => {
    forceFetch();
  };

  const allSelected =
    resources.length > 0 &&
    resources.every(resource =>
      selectedResources.includes(resourceKey(resource))
    );

  const selectAll = () => {
    if (allSelected) {
      setSelectedResources([]);
      return;
    }
    setSelectedResources(resources.map(resource => resourceKey(resource)));
  };

  const selectTab = (value: string) => {
    const pinnedResourcesOnly = value === 'pinned' ? true : null;
    setParams({
      ...params,
      pinnedResourcesOnly,
    });
    replaceHistory(
      encodeUrlQueryParams(
        pathname,
        params.search,
        params.sort,
        params.kinds,
        !!params.query,
        pinnedResourcesOnly
      )
    );
  };

  return (
    <FeatureBox
      className="ContainerContext"
      px={4}
      css={`
        max-width: ${RESOURCES_MAX_WIDTH};
        margin: auto;
      `}
    >
      {attempt.status === 'failed' && (
        <ErrorBox>
          <ErrorBoxInternal>
            <Danger>
              {attempt.statusText}
              <Box flex="0 0 auto" ml={2}>
                <ButtonLink onClick={onRetryClicked}>Retry</ButtonLink>
              </Box>
            </Danger>
          </ErrorBoxInternal>
        </ErrorBox>
      )}
      <FeatureHeader
        css={`
          border-bottom: none;
        `}
        mb={1}
        alignItems="center"
        justifyContent="space-between"
      >
        <FeatureHeaderTitle>Resources</FeatureHeaderTitle>
        <Flex alignItems="center">
          <AgentButtonAdd
            agent={SearchResource.UNIFIED_RESOURCE}
            beginsWithVowel={false}
            isLeafCluster={isLeafCluster}
            canCreate={canCreate}
          />
        </Flex>
      </FeatureHeader>
      <Flex alignItems="center" justifyContent="space-between">
        <SearchPanel
          params={params}
          setParams={setParams}
          pathname={pathname}
          replaceHistory={replaceHistory}
        />
        {selectedResources.length > 0 && (
          <ButtonBorder
            onClick={() => handlePinSelected(shouldUnpin)}
            textTransform="none"
            css={`
              border: none;
              color: ${props => props.theme.colors.brand};
            `}
          >
            <PushPin color="brand" size={16} mr={2} />
            {shouldUnpin ? 'Unpin ' : 'Pin '}
            Selected
          </ButtonBorder>
        )}
      </Flex>
      <FilterPanel
        params={params}
        setParams={setParams}
        setSort={setSort}
        pathname={pathname}
        replaceHistory={replaceHistory}
        selectAll={selectAll}
        selected={allSelected}
        shouldUnpin={shouldUnpin}
      />
      <Flex gap={4} mb={3}>
        {tabs.map(tab => (
          <ResourceTab
            key={tab.value}
            title={tab.label}
            value={tab.value}
            selectedTab={params.pinnedResourcesOnly ? 'pinned' : 'all'}
            onChange={selectTab}
          />
        ))}
      </Flex>
      <ResourcesContainer className="ResourcesContainer" gap={2}>
        {resources.map(res => {
          const key = resourceKey(res);
          return (
            <ResourceCard
              key={key}
              resource={res}
              onLabelClick={onLabelClick}
              pinResource={handlePinResource}
              pinned={pinnedResources.includes(key)}
              selected={selectedResources.includes(key)}
              selectResource={handleSelectResources}
            />
          );
        })}
        {/* Using index as key here is ok because these elements never change order */}
        {attempt.status === 'processing' &&
          loadingCardArray.map((_, i) => <LoadingCard delay="short" key={i} />)}
      </ResourcesContainer>
      <div ref={setScrollDetector} />
      <ListFooter>
        {attempt.status === 'failed' && resources.length > 0 && (
          <ButtonSecondary onClick={onRetryClicked}>Load more</ButtonSecondary>
        )}
        {noResults && isSearchEmpty && (
          <Empty
            clusterId={clusterId}
            canCreate={canCreate && !isLeafCluster}
            emptyStateInfo={emptyStateInfo}
          />
        )}
        {noResults && !isSearchEmpty && (
          <NoResults query={params?.query || params?.search} />
        )}
      </ListFooter>
    </FeatureBox>
  );
}

export function resourceKey(resource: UnifiedResource) {
  if (resource.kind === 'node') {
    return `${resource.id}/node`;
  }
  return `${resource.name}/${resource.kind}`;
}

export function resourceName(resource: UnifiedResource) {
  if (resource.kind === 'app' && resource.friendlyName) {
    return resource.friendlyName;
  }
  if (resource.kind === 'node') {
    return resource.hostname;
  }
  return resource.name;
}

function NoResults({ query }: { query: string }) {
  // Prevent `No resources were found for ""` flicker.
  if (query) {
    return (
      <Box p={8} mt={3} mx="auto" maxWidth="720px" textAlign="center">
        <TextIcon typography="h3">
          <Magnifier />
          No resources were found for&nbsp;
          <Text
            as="span"
            bold
            css={`
              max-width: 270px;
              overflow: hidden;
              text-overflow: ellipsis;
            `}
          >
            {query}
          </Text>
        </TextIcon>
      </Box>
    );
  }
  return null;
}

const ResourcesContainer = styled(Flex)`
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(400px, 1fr));
`;

const ErrorBox = styled(Box)`
  position: sticky;
  top: 0;
  z-index: 1;
`;

const ErrorBoxInternal = styled(Box)`
  position: absolute;
  left: 0;
  right: 0;
  margin: ${props => props.theme.space[1]}px 10% 0 10%;
`;

const INDICATOR_SIZE = '48px';

// It's important to make the footer at least as big as the loading indicator,
// since in the typical case, we want to avoid UI "jumping" when loading the
// final fragment finishes, and the final fragment is just one element in the
// final row (i.e. the number of rows doesn't change). It's then important to
// keep the same amount of whitespace below the resource list.
const ListFooter = styled.div`
  margin-top: ${props => props.theme.space[2]}px;
  min-height: ${INDICATOR_SIZE};
  text-align: center;
`;

const emptyStateInfo: EmptyStateInfo = {
  title: 'Add your first resource to Teleport',
  byline:
    'Connect SSH servers, Kubernetes clusters, Windows Desktops, Databases, Web apps and more from our integrations catalog.',
  readOnly: {
    title: 'No Resources Found',
    resource: 'resources',
  },
  resourceType: 'unified_resource',
};

type ResourceTabProps = {
  title: string;
  value: string;
  selectedTab: string;
  onChange: (value: string) => void;
};

const ResourceTab = ({
  title,
  value,
  selectedTab,
  onChange,
}: ResourceTabProps) => {
  const selectTab = () => {
    onChange(value);
  };

  const selected = value === selectedTab;

  return (
    <Box
      css={`
        cursor: pointer;
      `}
      onClick={selectTab}
    >
      <TabText selected={selected}>{title}</TabText>
      <TabTextUnderline selected={selected} />
    </Box>
  );
};

const TabText = styled(Text)`
  font-size: ${props => props.theme.fontSizes[2]};
  font-weight: ${props =>
    props.selected
      ? props.theme.fontWeights.bold
      : props.theme.fontWeights.regular};
  line-height: 20px;

  color: ${props =>
    props.selected ? props.theme.colors.brand : props.theme.colors.main};
`;

const TabTextUnderline = styled(Box)`
  height: 2px;
  // transparent background if not selected to preserve layout
  background-color: ${props =>
    props.selected ? props.theme.colors.brand : 'transparent'};
`;

export const HoverTooltip: React.FC<{
  tipContent: React.ReactElement;
  fontSize?: number;
}> = ({ tipContent, fontSize = 10, children }) => {
  const [anchorEl, setAnchorEl] = useState();
  const open = Boolean(anchorEl);

  function handlePopoverOpen(event) {
    setAnchorEl(event.currentTarget);
  }

  function handlePopoverClose() {
    setAnchorEl(null);
  }

  return (
    <Flex
      aria-owns={open ? 'mouse-over-popover' : undefined}
      onMouseEnter={handlePopoverOpen}
      onMouseLeave={handlePopoverClose}
    >
      {children}
      <Popover
        modalCss={modalCss}
        onClose={handlePopoverClose}
        open={open}
        anchorEl={anchorEl}
        anchorOrigin={{
          vertical: 'top',
          horizontal: 'center',
        }}
        transformOrigin={{
          vertical: 'bottom',
          horizontal: 'center',
        }}
      >
        <StyledOnHover px={2} py={1} fontSize={`${fontSize}px`}>
          {tipContent}
        </StyledOnHover>
      </Popover>
    </Flex>
  );
};

const modalCss = () => `
  pointer-events: none;
`;

const StyledOnHover = styled(Text)`
  color: ${props => props.theme.colors.text.main};
  background-color: ${props => props.theme.colors.tooltip.background};
  max-width: 350px;
`;
