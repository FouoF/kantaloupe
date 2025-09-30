/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/

import * as KantaloupeDynamiaAiApiTypesObjectmeta from "../../types/objectmeta.pb"
import * as KantaloupeDynamiaAiApiTypesPage from "../../types/page.pb"
export type ListNodesRequest = {
  cluster?: string
  page?: number
  pageSize?: number
  sortOption?: KantaloupeDynamiaAiApiTypesPage.SortOption
}

export type ListNodesResponse = {
  items?: Node[]
  pagination?: KantaloupeDynamiaAiApiTypesPage.Pagination
}

export type GetNodeSummaryRequest = {
  cluster?: string
  nodeID?: string
}

export type GetNodeSummaryResponse = {
  inform?: NodeInformation
  cpuCount?: string
  memoryCapacity?: string
  gpuCount?: string
  gpuMemoryCapacity?: string
}

export type GetNodeEventsRequest = {
  cluster?: string
  nodeID?: string
}

export type GetNodeEventsResponse = {
  events?: NodeResourceEvents[]
}

export type GetNodeStatusRequest = {
  cluster?: string
  nodeID?: string
}

export type GetNodeStatusResponse = {
  events?: NodeStatus[]
}

export type NodeStatus = {
  type?: string
  status?: string
  updateTimestamp?: string
  message?: string
}

export type NodeResourceEvents = {
  timestamp?: string
  type?: string
  reason?: string
  message?: string
}

export type NodeResourceTrend = {
  metrics?: string
  points?: NodeResourceTimestamp[]
}

export type NodeResourceTimestamp = {
  timestamp?: string
  value?: number
}

export type GetNodeKVRequest = {
  cluster?: string
  nodeID?: string
  kvName?: string
}

export type GetNodeKVResponse = {
  data?: KvValues[]
}

export type KvValues = {
  key?: string
  value?: string
}

export type GPUUsage = {
  gpuIndex?: string
  workloadCount?: string
}

export type NodeInformation = {
  kubeletVersion?: string
  os?: string
  ctrVersion?: string
  kernelVersion?: string
  arch?: string
  cudaVersion?: string
  nvDriverVersion?: string
  createTimestamp?: string
}

export type Node = {
  metadata?: KantaloupeDynamiaAiApiTypesObjectmeta.ObjectMeta
  name?: string
  health?: boolean
  character?: string
  cpucore?: ResourceUsage
  memory?: ResourceUsage
  gpuutil?: ResourceUsage
  gpumem?: ResourceUsage
  ipaddr?: string
  createtimestamp?: string
}

export type ResourceUsage = {
  used?: number
  allocated?: number
  total?: number
}