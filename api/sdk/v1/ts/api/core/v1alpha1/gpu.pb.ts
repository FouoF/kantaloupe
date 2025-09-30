/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/

export enum GPUModel {
  GPU_MODEL_UNSPECIFIED = "GPU_MODEL_UNSPECIFIED",
  GPU_MODEL_MIG = "GPU_MODEL_MIG",
  GPU_MODEL_GPU = "GPU_MODEL_GPU",
  GPU_MODEL_VGPU = "GPU_MODEL_VGPU",
}

export enum MIGStrategy {
  MIG_STRATEGY_UNSPECIFIED = "MIG_STRATEGY_UNSPECIFIED",
  MIG_STRATEGY_SINGLE = "MIG_STRATEGY_SINGLE",
  MIG_STRATEGY_MIXED = "MIG_STRATEGY_MIXED",
}

export type GPU = {
  uid?: string
  name?: string
  gpuMode?: GPUModel
  gpuInfo?: GPUInfo
  migSpecList?: MigSpec[]
}

export type GPUInfo = {
  modelName?: string
  device?: string
  gpuMemory?: string
}

export type MigSpec = {
  gpuIId?: string
  gpuIProfile?: string
}

export type GPUSummary = {
  node?: string
  vgpuTypes?: string[]
}

export type ListNodeGPURequest = {
  cluster?: string
  node?: string
}

export type ListNodeGPUResponse = {
  items?: GPU[]
}

export type UpdateNodeGPUModeRequest = {
  cluster?: string
  node?: string
  mode?: GPUModel
  migSpec?: MIGModeSpec
}

export type UpdateNodeGPUModeResponse = {
  mode?: GPUModel
}

export type MIGModeSpec = {
  config?: string
  strategy?: MIGStrategy
}

export type ListClusterGPUSummaryRequest = {
  cluster?: string
}

export type ListClusterGPUSummaryResponse = {
  summary?: GPUSummary[]
}

export type GetNodeGPUStatsRequest = {
  cluster?: string
  node?: string
}

export type GetNodeGPUStatsResponse = {
  mode?: GPUModel
  fullGpuStats?: FullGPUNodeStats
  vgpuStats?: VGPUNodeStats
  migStats?: MIGNodeStats
}

export type FullGPUNodeStats = {
  totalGpuNumber?: number
  allocatedGpuNumber?: number
}

export type VGPUNodeStats = {
  physicalGpuNumber?: number
  totalVirtualGpuNumber?: number
  allocatedVirtualGpuNumber?: number
  allocatedComputePower?: string
  totalComputePower?: string
  totalGpuMemory?: string
  allocatedGpuMemory?: string
}

export type MIGNodeStats = {
  totalGpuNumber?: number
  allocatedGpuNumber?: number
  totalMigNumber?: number
}