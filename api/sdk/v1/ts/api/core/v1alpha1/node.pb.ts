/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/

import * as KantaloupeDynamiaAiApiTypesObjectmeta from "../../types/objectmeta.pb"
import * as KantaloupeDynamiaAiApiTypesPage from "../../types/page.pb"
import * as KantaloupeDynamiaAiApiTypesTypes from "../../types/types.pb"

export enum TaintEffect {
  TAINT_EFFECT_UNSPECIFIED = "TAINT_EFFECT_UNSPECIFIED",
  NoSchedule = "NoSchedule",
  PreferNoSchedule = "PreferNoSchedule",
  NoExecute = "NoExecute",
}

export enum NodePhase {
  NODE_PHASE_UNSPECIFIED = "NODE_PHASE_UNSPECIFIED",
  Ready = "Ready",
  Not_Ready = "Not_Ready",
  Unknown = "Unknown",
}

export enum Role {
  NODE_ROLE_UNSPECIFIED = "NODE_ROLE_UNSPECIFIED",
  CONTROL_PLANE = "CONTROL_PLANE",
  WORKER = "WORKER",
}

export type Node = {
  metadata?: KantaloupeDynamiaAiApiTypesObjectmeta.ObjectMeta
  spec?: NodeSpec
  status?: NodeStatus
}

export type NodeSpec = {
  podCIDR?: string
  unschedulable?: boolean
  taints?: Taint[]
}

export type Taint = {
  key?: string
  value?: string
  effect?: TaintEffect
}

export type NodeStatusStatus = {
  phase?: NodePhase
  conditions?: NodeCondition[]
}

export type NodeStatus = {
  status?: NodeStatusStatus
  addresses?: NodeAddress[]
  cpuCapacity?: string
  cpuAllocated?: number
  cpuUsage?: number
  memoryCapacity?: string
  memoryAllocated?: number
  memoryUsage?: number
  gpuCount?: number
  gpuCoreTotal?: number
  gpuCoreAllocated?: number
  gpuCoreUsage?: number
  gpuMemoryTotal?: string
  gpuMemoryAllocatable?: string
  gpuMemoryAllocated?: string
  gpuMemoryUsage?: string
  systemInfo?: NodeSystemInfo
  roles?: Role[]
}

export type NodeCondition = {
  type?: string
  status?: KantaloupeDynamiaAiApiTypesTypes.ConditionStatus
  reason?: string
  message?: string
  updateTimestamp?: string
}

export type NodeSystemInfo = {
  kernelVersion?: string
  osImage?: string
  containerRuntimeVersion?: string
  kubeletVersion?: string
  architecture?: string
  cudaVersion?: string
  nvidiaVersion?: string
}

export type NodeAddress = {
  type?: string
  address?: string
}

export type ListNodesRequest = {
  cluster?: string
  name?: string
  role?: Role
  phase?: NodePhase
  page?: number
  pageSize?: number
  sortOption?: KantaloupeDynamiaAiApiTypesPage.SortOption
}

export type ListNodesResponse = {
  items?: Node[]
  pagination?: KantaloupeDynamiaAiApiTypesPage.Pagination
}

export type GetNodeRequest = {
  cluster?: string
  name?: string
}

export type PutNodeLabelsRequest = {
  cluster?: string
  node?: string
  labels?: {[key: string]: string}
}

export type PutNodeLabelsResponse = {
  labels?: {[key: string]: string}
}

export type PutNodeTaintsRequest = {
  cluster?: string
  node?: string
  taints?: Taint[]
}

export type PutNodeTaintsResponse = {
  taints?: Taint[]
}

export type UpdateNodeAnnotationsRequest = {
  cluster?: string
  node?: string
  annotations?: {[key: string]: string}
}

export type UpdateNodeAnnotationsResponse = {
  annotations?: {[key: string]: string}
}

export type ScheduleNodeRequest = {
  cluster?: string
  node?: string
}