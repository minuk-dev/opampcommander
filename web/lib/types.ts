// Type definitions mirroring api/v1/*.go on the server.

export interface ListMeta {
  continue: string;
  remainingItemCount: number;
}

export interface ListResponse<T> {
  kind: string;
  apiVersion: string;
  metadata: ListMeta;
  items: T[];
}

export interface Condition {
  type: string;
  lastTransitionTime: string;
  status: 'True' | 'False' | 'Unknown';
  reason: string;
  message?: string;
}

// ---------- Agent ----------
export interface AgentDescription {
  identifyingAttributes?: Record<string, string>;
  nonIdentifyingAttributes?: Record<string, string>;
}

export interface AgentCustomCapabilities {
  capabilities?: string[];
}

export interface AgentMetadata {
  instanceUid: string;
  namespace: string;
  description?: AgentDescription;
  capabilities?: number;
  customCapabilities?: AgentCustomCapabilities;
}

export interface OpAMPConnectionSettings {
  destinationEndpoint: string;
  headers?: Record<string, string[]>;
  certificateName?: string | null;
}

export interface TelemetryConnectionSettings {
  destinationEndpoint: string;
  headers?: Record<string, string[]>;
  certificateName?: string | null;
}

export interface OtherConnectionSettings {
  destinationEndpoint: string;
  headers?: Record<string, string[]>;
  certificateName?: string | null;
}

export interface ConnectionSettings {
  opamp?: OpAMPConnectionSettings;
  ownMetrics?: TelemetryConnectionSettings;
  ownLogs?: TelemetryConnectionSettings;
  ownTraces?: TelemetryConnectionSettings;
  otherConnections?: Record<string, OtherConnectionSettings>;
}

export interface AgentSpecRemoteConfig {
  remoteConfigNames?: string[];
}

export interface AgentSpecPackages {
  packages?: string[];
}

export interface AgentSpec {
  newInstanceUid?: string;
  connectionSettings?: ConnectionSettings;
  remoteConfig?: AgentSpecRemoteConfig;
  packagesAvailable?: AgentSpecPackages;
  restartRequiredAt?: string | null;
}

export interface AgentConfigFile {
  body: string;
  contentType: string;
}

export interface AgentConfigMap {
  configMap?: Record<string, AgentConfigFile>;
}

export interface AgentEffectiveConfig {
  configMap: AgentConfigMap;
}

export interface AgentComponentHealth {
  healthy: boolean;
  startTime?: string;
  lastError?: string;
  status?: string;
  statusTime?: string;
  componentsMap?: Record<string, string>;
}

export interface AgentStatus {
  effectiveConfig?: AgentEffectiveConfig;
  packageStatuses?: unknown;
  componentHealth: AgentComponentHealth;
  availableComponents?: unknown;
  conditions?: Condition[];
  connected: boolean;
  connectionType?: string;
  sequenceNum?: number;
  lastReportedAt?: string;
}

export interface Agent {
  metadata: AgentMetadata;
  spec?: AgentSpec;
  status: AgentStatus;
}

// ---------- AgentGroup ----------
export type Attributes = Record<string, string>;

export interface AgentSelector {
  identifyingAttributes?: Record<string, string>;
  nonIdentifyingAttributes?: Record<string, string>;
}

export interface AgentRemoteConfigSpec {
  value: string;
  contentType: string;
}

export interface AgentGroupRemoteConfig {
  agentRemoteConfigName?: string;
  agentRemoteConfigSpec?: AgentRemoteConfigSpec;
  agentRemoteConfigRef?: string;
}

export interface AgentGroupAgentConfig {
  agentRemoteConfig?: AgentGroupRemoteConfig;
  connectionSettings?: ConnectionSettings;
}

export interface AgentGroupMetadata {
  namespace: string;
  name: string;
  attributes: Attributes;
  createdAt: string;
  deletedAt?: string | null;
}

export interface AgentGroupSpec {
  priority: number;
  selector: AgentSelector;
  agentConfig?: AgentGroupAgentConfig;
}

export interface AgentGroupStatus {
  numAgents: number;
  numConnectedAgents: number;
  numHealthyAgents: number;
  numUnhealthyAgents: number;
  numNotConnectedAgents: number;
  conditions?: Condition[];
}

export interface AgentGroup {
  metadata: AgentGroupMetadata;
  spec: AgentGroupSpec;
  status: AgentGroupStatus;
}

// ---------- Namespace ----------
export interface NamespaceMetadata {
  name: string;
  labels?: Record<string, string>;
  annotations?: Record<string, string>;
  createdAt: string;
  deletedAt?: string | null;
}

export interface NamespaceStatus {
  conditions?: Condition[];
}

export interface Namespace {
  metadata: NamespaceMetadata;
  status: NamespaceStatus;
}

// ---------- Connection ----------
export interface Connection {
  id: string;
  instanceUid: string;
  namespace: string;
  type: string;
  lastCommunicatedAt: string;
  alive: boolean;
}

// ---------- Server ----------
export interface ServerCondition {
  type: 'Registered' | 'Alive' | string;
  lastTransitionTime: string;
  status: 'True' | 'False' | 'Unknown';
  reason: string;
  message?: string;
}

export interface Server {
  id: string;
  lastHeartbeatAt: string;
  conditions?: ServerCondition[];
}

// ---------- Certificate ----------
export interface CertificateMetadata {
  name: string;
  namespace: string;
  attributes?: Attributes;
  createdAt: string;
  deletedAt?: string | null;
}

export interface CertificateSpec {
  cert?: string;
  privateKey?: string;
  caCert?: string;
}

export interface Certificate {
  kind: string;
  apiVersion: string;
  metadata: CertificateMetadata;
  spec: CertificateSpec;
  status?: { conditions?: Condition[] };
}

// ---------- AgentPackage ----------
export interface AgentPackageMetadata {
  name: string;
  namespace: string;
  attributes?: Attributes;
  createdAt: string;
  deletedAt?: string | null;
}

export interface AgentPackageSpec {
  packageType: string;
  version: string;
  downloadUrl: string;
  contentHash?: string;
  signature?: string;
  headers?: Record<string, string>;
  hash?: string;
}

export interface AgentPackage {
  metadata: AgentPackageMetadata;
  spec: AgentPackageSpec;
  status?: { conditions?: Condition[] };
}

// ---------- AgentRemoteConfig ----------
export interface AgentRemoteConfigMetadata {
  name: string;
  namespace: string;
  attributes?: Attributes;
  createdAt: string;
}

export interface AgentRemoteConfig {
  metadata: AgentRemoteConfigMetadata;
  spec: AgentRemoteConfigSpec;
  status?: { conditions?: Condition[] };
}

// ---------- Role / RoleBinding / User ----------
export interface RoleMetadata {
  uid: string;
  createdAt: string;
  updatedAt: string;
  deletedAt?: string | null;
}

export interface RoleSpec {
  displayName: string;
  description: string;
  permissions?: string[];
  isBuiltIn: boolean;
}

export interface Role {
  kind: string;
  apiVersion: string;
  metadata: RoleMetadata;
  spec: RoleSpec;
  status?: { conditions?: Condition[] };
}

export interface RoleBindingMetadata {
  namespace: string;
  name: string;
  createdAt?: string;
  updatedAt?: string;
  deletedAt?: string | null;
}

export interface RoleBindingRoleRef {
  kind: string;
  name: string;
}

export interface RoleBindingSubject {
  kind: string;
  name: string;
  apiVersion?: string;
}

export interface RoleBindingSpec {
  roleRef: RoleBindingRoleRef;
  subjects?: RoleBindingSubject[];
}

export interface RoleBinding {
  kind: string;
  apiVersion: string;
  metadata: RoleBindingMetadata;
  spec: RoleBindingSpec;
  status?: { conditions?: Condition[] };
}

export interface UserMetadata {
  uid: string;
  createdAt: string;
  updatedAt: string;
  deletedAt?: string | null;
  labels?: Record<string, string>;
}

export interface UserSpec {
  email: string;
  username: string;
  isActive: boolean;
}

export interface User {
  kind: string;
  apiVersion: string;
  metadata: UserMetadata;
  spec: UserSpec;
  status?: { conditions?: Condition[]; roles?: string[] };
}

export interface UserRoleEntry {
  role: Role;
  roleBinding?: RoleBinding | null;
}

export interface UserProfileResponse {
  user: User;
  roles?: UserRoleEntry[];
}

// ---------- Auth ----------
export interface AuthnTokenResponse {
  token: string;
  refreshToken?: string;
  expiresAt?: string;
}

export interface AuthInfo {
  authenticated: boolean;
  email?: string | null;
}

export interface OAuth2AuthCodeURLResponse {
  url: string;
}

export interface VersionInfo {
  major?: string;
  minor?: string;
  gitVersion?: string;
  gitCommit?: string;
  gitTreeState?: string;
  buildDate?: string;
  goVersion?: string;
  compiler?: string;
  platform?: string;
}
