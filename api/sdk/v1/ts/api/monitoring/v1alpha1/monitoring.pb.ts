/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/

import * as GoogleProtobufWrappers from "../../../google/protobuf/wrappers.pb"
import * as KantaloupeDynamiaAiApiTypesPage from "../../types/page.pb"

export enum ResourceType {
  RESOURCE_TYPE_UNSPECIFIED = "RESOURCE_TYPE_UNSPECIFIED",
  RESOURCE_TYPE_CPU = "RESOURCE_TYPE_CPU",
  RESOURCE_TYPE_MEMORY = "RESOURCE_TYPE_MEMORY",
  RESOURCE_TYPE_GPU_CORE = "RESOURCE_TYPE_GPU_CORE",
  RESOURCE_TYPE_GPU_MEMORY = "RESOURCE_TYPE_GPU_MEMORY",
  RESOURCE_TYPE_STORAGE = "RESOURCE_TYPE_STORAGE",
  RESOURCE_TYPE_NETWORK = "RESOURCE_TYPE_NETWORK",
  RESOURCE_TYPE_TEMP = "RESOURCE_TYPE_TEMP",
  RESOURCE_TYPE_POWER = "RESOURCE_TYPE_POWER",
}

export enum RankingType {
  RANKING_TYPE_UNSPECIFIED = "RANKING_TYPE_UNSPECIFIED",
  RANKING_TYPE_ALLOCATED = "RANKING_TYPE_ALLOCATED",
  RANKING_TYPE_USED = "RANKING_TYPE_USED",
}

export enum RequstType {
  UNSPECIFIED = "UNSPECIFIED",
  CORE = "CORE",
  MEMORY = "MEMORY",
}

export type Monitoring = {
  podName?: string
  podNamespace?: string
  gpuUuid?: string
  nodeName?: string
  gpuUtilization?: number
  modelName?: string
}

export type ListMonitoringsRequest = {
  cluster?: string
  podName?: string
  podNamespace?: string
  gpuUuid?: string
  nodeName?: string
  modelName?: string
  sortOption?: KantaloupeDynamiaAiApiTypesPage.SortOption
}

export type ListMonitoringsResponse = {
  items?: Monitoring[]
  pagination?: KantaloupeDynamiaAiApiTypesPage.Pagination
}

export type TimeSeriesPoint = {
  timestamp?: string
  value?: GoogleProtobufWrappers.DoubleValue
}

export type TimeSeries = {
  metric?: string
  points?: TimeSeriesPoint[]
}

export type ResourceTrendRequest = {
  cluster?: string
  resourceType?: ResourceType
  range?: string
  start?: string
  end?: string
  step?: string
}

export type ResourceTrendResponse = {
  data?: TimeSeries[]
}

export type NodeResourceTrendRequest = {
  cluster?: string
  node?: string
  resourceType?: ResourceType
  range?: string
  start?: string
  end?: string
  step?: string
}

export type KantaloupeflowResourceTrendRequest = {
  cluster?: string
  namespace?: string
  name?: string
  resourceType?: ResourceType
  range?: string
  start?: string
  end?: string
  step?: string
}

export type DistributionPoint = {
  name?: string
  value?: number
}

export type WorkloadDistributionResponse = {
  data?: DistributionPoint[]
}

export type NodeWorkloadDistributionRequest = {
  cluster?: string
  node?: string
}

export type ClusterWorkloadDistributionRequest = {
  cluster?: string
}

export type TopNodeRequest = {
  cluster?: string
  resourceType?: ResourceType
  rankingType?: RankingType
  limit?: number
}

export type TopNodeWorkloadRequest = {
  cluster?: string
  limit?: number
}

export type TopNodeResponse = {
  data?: DistributionPoint[]
}

export type MemoryDistributionRequest = {
  cluster?: string
  namespace?: string
  name?: string
}

export type MemoryDistributionResponse = {
  data?: DistributionPoint64[]
}

export type DistributionPoint64 = {
  name?: string
  value?: string
}

export type WorkloadInfo = {
  name?: string
  coreAllocated?: number
  memoryAllocated?: number
  coreUsage?: number
  memoryUsage?: number
}

export type CardTopWorkloadsRequest = {
  cluster?: string
  node?: string
  uuid?: string
  limit?: number
  type?: RequstType
}

export type CardTopWorkloadsResponse = {
  workloads?: WorkloadInfo[]
  total?: number
}

export type GpuResourceTrendRequest = {
  cluster?: string
  node?: string
  uuid?: string
  resourceType?: ResourceType
  range?: string
  start?: string
  end?: string
  step?: string
}

export type GetClusterWorkloadsTopRequest = {
  cluster?: string
  limit?: number
  type?: RequstType
  range?: string
  step?: string
}

export type GetClusterWorkloadsTopResponse = {
  data?: TimeSeries[]
}