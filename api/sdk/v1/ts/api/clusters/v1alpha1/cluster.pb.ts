/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/

import * as KantaloupeDynamiaAiApiTypesObjectmeta from "../../types/objectmeta.pb"
import * as KantaloupeDynamiaAiApiTypesPage from "../../types/page.pb"

export enum ClusterProvider {
  CLUSTER_PROVIDER_UNSPECIFIED = "CLUSTER_PROVIDER_UNSPECIFIED",
  GENERIC = "GENERIC",
  REDHAT_OPENSHIFT4 = "REDHAT_OPENSHIFT4",
  SUSE_RANCHER = "SUSE_RANCHER",
  VMWARE_TANZU = "VMWARE_TANZU",
  AWS_EKS = "AWS_EKS",
  ALIYUN_ACK = "ALIYUN_ACK",
  HUAWEI_CCE = "HUAWEI_CCE",
  GCP_GKE = "GCP_GKE",
}

export enum ClusterType {
  CLUSTER_TYPE_UNSPECIFIED = "CLUSTER_TYPE_UNSPECIFIED",
  NVIDIA = "NVIDIA",
  METAX = "METAX",
  CAMBRICON = "CAMBRICON",
  MOORE_THREADS = "MOORE_THREADS",
  ILUVATAR_COREX = "ILUVATAR_COREX",
  HYGON = "HYGON",
  ASCEND = "ASCEND",
  NEURON = "NEURON",
}

export enum ClusterState {
  UNSPECIFED = "UNSPECIFED",
  RUNNING = "RUNNING",
  UNHEALTH = "UNHEALTH",
}

export enum RankOption {
  RANK_OPTION_UNSPECIFIED = "RANK_OPTION_UNSPECIFIED",
  RANK_OPTION_MEMORY = "RANK_OPTION_MEMORY",
  RANK_OPTION_CORE = "RANK_OPTION_CORE",
}

export enum KantaloupePluginName {
  KANTALOUPE_PLUGIN_NAME_UNSPECIFED = "KANTALOUPE_PLUGIN_NAME_UNSPECIFED",
  HAMI = "HAMI",
}

export type Cluster = {
  metadata?: KantaloupeDynamiaAiApiTypesObjectmeta.ObjectMeta
  spec?: ClusterSpec
  status?: ClusterStatus
}

export type ClusterSpec = {
  provider?: ClusterProvider
  type?: ClusterType
  apiEndpoint?: string
  region?: string
  zone?: string
  aliasName?: string
  description?: string
  prometheusAddress?: string
  gatewayAddress?: string
}

export type ClusterStatus = {
  kubernetesVersion?: string
  kubeSystemID?: string
  clusterVersion?: string
  nodeSummary?: ResourceSummary
  podSummary?: ResourceSummary
  kantaloupeflowSummary?: ResourceSummary
  cpuUsage?: number
  memoryUsage?: number
  gpuTotal?: number
  gpuMemoryTotal?: string
  cpuTotal?: number
  memoryTotal?: string
  gpuCoreUsage?: number
  gpuMemoryUsage?: number
  gpuCoreAllocated?: number
  gpuMemoryAllocated?: number
  state?: ClusterState
  conditions?: KantaloupeDynamiaAiApiTypesObjectmeta.Condition[]
}

export type ResourceSummary = {
  totalNum?: number
  readyNum?: number
}

export type PlatformSummury = {
  clusterNum?: number
  nodeNum?: number
  acceleratorCardNum?: number
  kantaloupeflowNum?: number
  acceleratorCardSummury?: AcceleratorCardSummury[]
}

export type AcceleratorCardSummury = {
  mode?: string
  totalNum?: number
  useageNum?: number
  idelNum?: number
}

export type GetPlatformSummuryRequest = {
  threshold?: number
}

export type ListClustersRequest = {
  name?: string
  page?: number
  pageSize?: number
  type?: ClusterType
  provider?: ClusterProvider
  state?: ClusterState
  sortOption?: KantaloupeDynamiaAiApiTypesPage.SortOption
}

export type ListClustersResponse = {
  items?: Cluster[]
  pagination?: KantaloupeDynamiaAiApiTypesPage.Pagination
}

export type IntegrateClusterRequest = {
  name?: string
  aliasName?: string
  provider?: ClusterProvider
  labels?: {[key: string]: string}
  annotations?: {[key: string]: string}
  description?: string
  kubeConfig?: string
  prometheusAddress?: string
  gatewayAddress?: string
  type?: ClusterType
}

export type DeleteClusterRequest = {
  name?: string
}

export type GetClusterRequest = {
  name?: string
}

export type ValidateKubeconfigRequest = {
  kubeconfig?: string
}

export type ValidateKubeconfigResponse = {
  validate?: boolean
}

export type UpdateClusterRequest = {
  name?: string
  aliasName?: string
  labels?: {[key: string]: string}
  annotations?: {[key: string]: string}
  description?: string
  prometheusAddress?: string
  gatewayAddress?: string
  kubeConfig?: string
}

export type ListClusterVersionsResponse = {
  versions?: string[]
}

export type GPUSummary = {
  model?: string
  total?: number
  memAllocated?: number
  memUsage?: number
  coreAllocated?: number
  coreUsage?: number
}

export type GetPlatformGPUTopRequest = {
  topn?: number
  rankOption?: RankOption
}

export type GetPlatformGPUTopResponse = {
  gpus?: GPUSummary[]
}

export type KantaloupePlugin = {
  name?: KantaloupePluginName
  namespace?: string
}

export type GetClusterPluginsRequest = {
  name?: string
}

export type GetClusterPluginsResponse = {
  plugins?: KantaloupePlugin[]
}

export type ResourceName = {
  cardModel?: string
  resourceKeys?: string[]
}

export type CardRequestType = {
  requestType?: string
  resourceNames?: ResourceName[]
}

export type GetClusterCardRequestTypeRequest = {
  name?: string
}

export type GetClusterCardRequestTypeResponse = {
  requestTypes?: CardRequestType[]
}