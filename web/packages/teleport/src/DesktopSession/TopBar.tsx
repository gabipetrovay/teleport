/*
Copyright 2021 Gravitational, Inc.

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
import React from 'react';
import { useTheme } from 'styled-components';
import { Text, TopNav, Flex } from 'design';
import { Clipboard, FolderShared } from 'design/Icon';

import ActionMenu from './ActionMenu';
import { WarningDropdown } from './WarningDropdown';

import type { NotificationItem } from 'shared/components/Notification';

export default function TopBar(props: Props) {
  const {
    userHost,
    clipboardSharingEnabled,
    onDisconnect,
    canShareDirectory,
    isSharingDirectory,
    onShareDirectory,
    warnings,
    onRemoveWarning,
  } = props;
  const theme = useTheme();

  const primaryOnTrue = (b: boolean): any => {
    return {
      color: b ? theme.colors.text.main : theme.colors.text.slightlyMuted,
    };
  };

  return (
    <TopNav
      height={`${TopBarHeight}px`}
      bg="levels.deep"
      style={{
        justifyContent: 'space-between',
      }}
    >
      <Text px={3} style={{ color: theme.colors.text.slightlyMuted }}>
        {userHost}
      </Text>

      <Flex px={3}>
        <Flex alignItems="center">
          <FolderShared
            style={primaryOnTrue(isSharingDirectory)}
            pr={3}
            title={
              isSharingDirectory
                ? 'Directory Sharing Enabled'
                : 'Directory Sharing Disabled'
            }
          />
          <Clipboard
            style={primaryOnTrue(clipboardSharingEnabled)}
            pr={3}
            title={
              clipboardSharingEnabled
                ? 'Clipboard Sharing Enabled'
                : 'Clipboard Sharing Disabled'
            }
          />
          <WarningDropdown
            warnings={warnings}
            onRemoveWarning={onRemoveWarning}
          />
        </Flex>
        <ActionMenu
          onDisconnect={onDisconnect}
          showShareDirectory={canShareDirectory && !isSharingDirectory}
          onShareDirectory={onShareDirectory}
        />
      </Flex>
    </TopNav>
  );
}

export const TopBarHeight = 40;

type Props = {
  userHost: string;
  clipboardSharingEnabled: boolean;
  canShareDirectory: boolean;
  isSharingDirectory: boolean;
  onDisconnect: VoidFunction;
  onShareDirectory: VoidFunction;
  warnings: NotificationItem[];
  onRemoveWarning(id: string): void;
};
