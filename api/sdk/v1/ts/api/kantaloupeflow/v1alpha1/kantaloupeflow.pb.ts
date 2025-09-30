/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/

import * as KantaloupeDynamiaAiApiTypesObjectmeta from "../../types/objectmeta.pb"
import * as KantaloupeDynamiaAiApiTypesPage from "../../types/page.pb"

export enum PluginType {
  PLUGIN_TYPE_UNSPECIFIED = "PLUGIN_TYPE_UNSPECIFIED",
  ssh = "ssh",
  vscode = "vscode",
  jupyter = "jupyter",
}

export enum WorkloadType {
  WORKLOAD_TYPE_UNSPECIFIED = "WORKLOAD_TYPE_UNSPECIFIED",
  Pod = "Pod",
  Deployment = "Deployment",
}

export enum KantaloupeflowState {
  KANTALOUPEFLOW_STATE_UNSPECIFIED = "KANTALOUPEFLOW_STATE_UNSPECIFIED",
  Unknow = "Unknow",
  Progressing = "Progressing",
  Running = "Running",
  Falied = "Falied",
}

export type Kantaloupeflow = {
  metadata?: KantaloupeDynamiaAiApiTypesObjectmeta.ObjectMeta
  spec?: KantaloupeflowSpec
  status?: KantaloupeflowStatus
}

export type KantaloupeflowSpec = {
  plugins?: PluginType[]
  replicas?: number
  template?: PodTemplateSpec
  paused?: boolean
  workload?: WorkloadType
}

export type KantaloupeflowStatus = {
  replicas?: number
  readyReplicas?: number
  networks?: Network[]
  state?: KantaloupeflowState
  conditions?: KantaloupeDynamiaAiApiTypesObjectmeta.Condition[]
  gpus?: GPU[]
}

export type Network = {
  name?: string
  url?: string
}

export type PodTemplateSpec = {
  metadata?: KantaloupeDynamiaAiApiTypesObjectmeta.ObjectMeta
  spec?: PodSpec
}

export type PodSpec = {
  volumes?: Volume[]
  containers?: Container[]
}

export type Volume = {
  name?: string
  hostPath?: HostPathVolumeSource
  emptyDir?: EmptyDirVolumeSource
  secret?: SecretVolumeSource
  persistentVolumeClaim?: PersistentVolumeClaimVolumeSource
  configMap?: ConfigMapVolumeSource
}

export type Container = {
  name?: string
  image?: string
  command?: string[]
  args?: string[]
  workingDir?: string
  ports?: Ports[]
  env?: EnvVar[]
  resources?: ResourceRequirements
  volumeMounts?: VolumeMount[]
  imagePullPolicy?: string
}

export type VolumeMount = {
  name?: string
  readOnly?: boolean
  mountPath?: string
  subPath?: string
  mountPropagation?: string
  subPathExpr?: string
}

export type EnvVar = {
  name?: string
  value?: string
}

export type Ports = {
  containerPort?: number
  hostPort?: number
  name?: string
  protocol?: string
}

export type ResourceList = {
  cpu?: string
  memory?: string
  storage?: string
  resources?: {[key: string]: string}
}

export type ResourceRequirements = {
  limits?: ResourceList
  requests?: ResourceList
}

export type HostPathVolumeSource = {
  path?: string
  type?: string
}

export type EmptyDirVolumeSource = {
  medium?: string
  sizeLimit?: string
}

export type SecretVolumeSource = {
  secretName?: string
  items?: KeyToPath[]
  defaultMode?: number
  optional?: boolean
}

export type KeyToPath = {
  key?: string
  path?: string
  mode?: number
}

export type PersistentVolumeClaimVolumeSource = {
  claimName?: string
  readOnly?: boolean
}

export type ConfigMapVolumeSource = {
  name?: string
  items?: KeyToPath[]
  defaultMode?: number
  optional?: boolean
}

export type KantaloupeTree = {
  data?: KantaloupeTreeNode[]
}

export type KantaloupeTreeNode = {
  name?: string
  value?: number
  children?: KantaloupeTreeNode[]
}

export type CreateKantaloupeflowRequest = {
  cluster?: string
  data?: Kantaloupeflow
}

export type GetKantaloupeflowRequest = {
  cluster?: string
  namespace?: string
  name?: string
}

export type ListKantaloupeflowsRequest = {
  name?: string
  cluster?: string
  namespace?: string
  status?: KantaloupeflowState
  page?: number
  pageSize?: number
  sortBy?: KantaloupeDynamiaAiApiTypesPage.SortBy
  sortDir?: KantaloupeDynamiaAiApiTypesPage.SortDir
}

export type ListKantaloupeflowsResponse = {
  items?: Kantaloupeflow[]
  pagination?: KantaloupeDynamiaAiApiTypesPage.Pagination
}

export type DeleteKantaloupeflowRequest = {
  cluster?: string
  namespace?: string
  name?: string
}

export type UpdateKantaloupeflowGPUMemoryRequest = {
  cluster?: string
  namespace?: string
  name?: string
  gpumemory?: number
}

export type GPU = {
  uuid?: string
  model?: string
  memory?: number
  core?: number
}

export type GetKantaloupeflowResponse = {
  kantaloupeflow?: Kantaloupeflow
  node?: string
}

export type GetKantaloupeflowConditionsRequest = {
  cluster?: string
  namespace?: string
  name?: string
}

export type ConditionStrings = {
  type?: string
  status?: string
  message?: string
}

export type GetKantaloupeflowConditionsResponse = {
  conditions?: ConditionStrings[]
}