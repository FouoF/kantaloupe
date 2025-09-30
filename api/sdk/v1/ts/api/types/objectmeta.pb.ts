/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/

export enum WorkloadState {
  WORKLOAD_STATE_UNSPECIFIED = "WORKLOAD_STATE_UNSPECIFIED",
  Running = "Running",
  Deleting = "Deleting",
  Not_Ready = "Not_Ready",
  Stopped = "Stopped",
  Waiting = "Waiting",
}

export type OwnerReference = {
  uid?: string
  controller?: boolean
  name?: string
  kind?: string
  apiVersion?: string
  blockOwnerDeletion?: boolean
}

export type ObjectMeta = {
  name?: string
  namespace?: string
  uid?: string
  resourceVersion?: string
  creationTimestamp?: string
  deletionTimestamp?: string
  labels?: {[key: string]: string}
  annotations?: {[key: string]: string}
  ownerReferences?: OwnerReference[]
}

export type Selector = {
  matchLabels?: {[key: string]: string}
}

export type LabelSelector = {
  matchLabels?: {[key: string]: string}
  matchExpressions?: LabelSelectorRequirement[]
}

export type LabelSelectorRequirement = {
  key?: string
  operator?: string
  values?: string[]
}

export type RollingUpdate = {
  maxSurge?: string
  maxUnavailable?: string
}

export type UpdateStrategy = {
  rollingUpdate?: RollingUpdate
  type?: string
}

export type Condition = {
  lastTransitionTime?: string
  lastUpdateTime?: string
  message?: string
  reason?: string
  status?: string
  type?: string
}