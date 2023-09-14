/**
 * Copyright 2020-2022 Gravitational, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { ResourceLabel } from 'teleport/services/agents';

export type GuessedAppType =
  | 'Grafana'
  | 'Slack'
  | 'Jenkins'
  | 'Application'
  | 'Aws';
export interface App {
  kind: 'app';
  id: string;
  name: string;
  description: string;
  uri: string;
  publicAddr: string;
  labels: ResourceLabel[];
  clusterId: string;
  launchUrl: string;
  fqdn: string;
  awsRoles: AwsRole[];
  awsConsole: boolean;
  isCloudOrTcpEndpoint?: boolean;
  // addrWithProtocol can either be a public address or
  // if public address wasn't defined, fallback to uri
  addrWithProtocol?: string;
  friendlyName?: string;
  userGroups: UserGroupAndDescription[];
  // samlApp is whether the application is a SAML Application (Service Provider).
  samlApp: boolean;
  // samlAppSsoUrl is the URL that triggers IdP-initiated SSO for SAML Application;
  samlAppSsoUrl?: string;
  // guessedAppIconName is our best guess at what type of app this is based on factors like name and labels
  guessedAppIconName?: GuessedAppType;
}

export type AwsRole = {
  arn: string;
  display: string;
};

export type UserGroupAndDescription = {
  name: string;
  description: string;
};
