/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/

import * as KantaloupeDynamiaAiApiTypesObjectmeta from "../../types/objectmeta.pb"
import * as KantaloupeDynamiaAiApiTypesPage from "../../types/page.pb"

export enum NamespacePhase {
  NAMESPACE_PHASE_UNSPECIFIED = "NAMESPACE_PHASE_UNSPECIFIED",
  Active = "Active",
  Terminating = "Terminating",
}

export enum Mode {
  MODE_UNSPECIFIED = "MODE_UNSPECIFIED",
  enforce = "enforce",
  audit = "audit",
  warn = "warn",
}

export type Namespace = {
  name?: string
  resourceQuotas?: string[]
}

export type NamespaceSpec = {
  finalizers?: string[]
}

export type NamespaceStatus = {
  phase?: NamespacePhase
  readyPodNumber?: number
  totalPodNumber?: number
  conditions?: KantaloupeDynamiaAiApiTypesObjectmeta.Condition[]
  podSecurityEnabled?: boolean
  cpuUsage?: number
  memoryUsage?: number
}

export type NamespaceCondition = {
  type?: string
  status?: string
  lastTransitionTime?: string
  reason?: string
  message?: string
}

export type ListClusterNamespacesRequest = {
  cluster?: string
  workspaceId?: number
  workspaceAlias?: string
  name?: string
  page?: number
  pageSize?: number
  sortOption?: KantaloupeDynamiaAiApiTypesPage.SortOption
  labelSelector?: string
  fieldSelector?: string
  fuzzyName?: string
  phase?: NamespacePhase
  excludeSystem?: boolean
  resourceQuota?: boolean
}

export type ListClusterNamespacesResponse = {
  items?: Namespace[]
  pagination?: KantaloupeDynamiaAiApiTypesPage.Pagination
}