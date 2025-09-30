/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/

import * as KantaloupeDynamiaAiApiTypesObjectmeta from "../../types/objectmeta.pb"
import * as KantaloupeDynamiaAiApiTypesPage from "../../types/page.pb"
export type QuotaSpec = {
  hard?: {[key: string]: string}
}

export type QuotaStatus = {
  hard?: {[key: string]: string}
  used?: {[key: string]: string}
}

export type Quota = {
  metadata?: KantaloupeDynamiaAiApiTypesObjectmeta.ObjectMeta
  spec?: QuotaSpec
  status?: QuotaStatus
}

export type QuotaResponse = {
  name?: string
  namespace?: string
  createdTime?: string
  hard?: {[key: string]: string}
  used?: {[key: string]: string}
  labels?: {[key: string]: string}
  workload?: string[]
  isManaged?: boolean
}

export type ListQuotasRequest = {
  name?: string
  namespace?: string
  page?: number
  pageSize?: number
  cluster?: string
  sortOption?: KantaloupeDynamiaAiApiTypesPage.SortOption
}

export type ListQuotasResponse = {
  items?: QuotaResponse[]
  pagination?: KantaloupeDynamiaAiApiTypesPage.Pagination
}

export type DeleteQuotaRequest = {
  name?: string
  namespace?: string
  cluster?: string
}

export type CreateQuotaRequest = {
  name?: string
  namespace?: string
  cluster?: string
  hard?: {[key: string]: string}
}

export type UpdateQuotaRequest = {
  name?: string
  namespace?: string
  cluster?: string
  hard?: {[key: string]: string}
}

export type GetQuotaRequest = {
  name?: string
  namespace?: string
  cluster?: string
}