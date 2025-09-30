/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/

import * as KantaloupeDynamiaAiApiTypesObjectmeta from "../../types/objectmeta.pb"
import * as KantaloupeDynamiaAiApiTypesPage from "../../types/page.pb"

export enum StorageClassReclaimPolicy {
  RECLAIM_AOLICY_UNSPECIFIED = "RECLAIM_AOLICY_UNSPECIFIED",
  Delete = "Delete",
  Retain = "Retain",
}

export enum StorageClassVolumeBindingMode {
  VOLUME_BINDING_MODE_UNSPECIFIED = "VOLUME_BINDING_MODE_UNSPECIFIED",
  Immediate = "Immediate",
  WaitForFirstConsumer = "WaitForFirstConsumer",
}

export type StorageClass = {
  metadata?: KantaloupeDynamiaAiApiTypesObjectmeta.ObjectMeta
  provisioner?: string
  reclaimPolicy?: StorageClassReclaimPolicy
  storageClassName?: string
  volumeBindingMode?: StorageClassVolumeBindingMode
  mountOptions?: string[]
  parameters?: {[key: string]: string}
  allowVolumeExpansion?: boolean
}

export type ListStorageClassesRequest = {
  cluster?: string
  page?: number
  pageSize?: number
  name?: string
  sortOption?: KantaloupeDynamiaAiApiTypesPage.SortOption
  labelSelector?: string
  fieldSelector?: string
  provisioner?: string
  reclaimPolicy?: string
  fuzzyName?: string
}

export type ListStorageClassesResponse = {
  items?: StorageClass[]
  pagination?: KantaloupeDynamiaAiApiTypesPage.Pagination
}