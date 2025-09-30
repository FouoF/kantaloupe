/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/

import * as KantaloupeDynamiaAiApiClustersV1alpha1Cluster from "../../clusters/v1alpha1/cluster.pb"
import * as KantaloupeDynamiaAiApiCoreV1alpha1Node from "../../core/v1alpha1/node.pb"
import * as KantaloupeDynamiaAiApiTypesPage from "../../types/page.pb"

export enum AcceleratorCardState {
  ACCELERATORCARD_STATE_UNSPECIFIED = "ACCELERATORCARD_STATE_UNSPECIFIED",
  HEALTH = "HEALTH",
  ERROR = "ERROR",
}

export type AcceleratorCard = {
  uuid?: string
  node?: string
  model?: string
  state?: AcceleratorCardState
  temperature?: number
  power?: number
  provider?: KantaloupeDynamiaAiApiClustersV1alpha1Cluster.ClusterProvider
  type?: KantaloupeDynamiaAiApiClustersV1alpha1Cluster.ClusterType
  nodeAddresses?: KantaloupeDynamiaAiApiCoreV1alpha1Node.NodeAddress[]
  gpuCoreTotal?: number
  gpuCoreAllocated?: number
  gpuCoreUsage?: number
  gpuMemoryTotal?: string
  gpuMemoryAllocated?: string
  gpuMemoryAllocatable?: string
  gpuMemoryUsage?: string
  workloadLimit?: number
}

export type ListAcceleratorCardsRequest = {
  cluster?: string
  uuid?: string
  model?: string
  node?: string
  state?: AcceleratorCardState
  page?: number
  pageSize?: number
  sortOption?: KantaloupeDynamiaAiApiTypesPage.SortOption
}

export type ListAcceleratorCardsResponse = {
  items?: AcceleratorCard[]
  pagination?: KantaloupeDynamiaAiApiTypesPage.Pagination
}

export type GetAcceleratorCardRequest = {
  cluster?: string
  node?: string
  uuid?: string
}

export type ListModelNamesRequest = {
  cluster?: string
}

export type ListModelNamesResponse = {
  modelNames?: string[]
}