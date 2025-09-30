/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/

import * as KantaloupeDynamiaAiApiTypesObjectmeta from "../../types/objectmeta.pb"
import * as KantaloupeDynamiaAiApiTypesPage from "../../types/page.pb"
export type Secret = {
  metadata?: KantaloupeDynamiaAiApiTypesObjectmeta.ObjectMeta
  immutable?: boolean
  data?: {[key: string]: string}
  stringData?: {[key: string]: string}
  type?: string
}

export type ListSecretsRequest = {
  cluster?: string
  page?: number
  pageSize?: number
  namespace?: string
  name?: string
  sortOption?: KantaloupeDynamiaAiApiTypesPage.SortOption
  labelSelector?: string
  fieldSelector?: string
  type?: string
  onlyMetadata?: boolean
  fuzzyName?: string
}

export type ListSecretsResponse = {
  items?: Secret[]
  pagination?: KantaloupeDynamiaAiApiTypesPage.Pagination
}

export type GetSecretRequest = {
  cluster?: string
  namespace?: string
  name?: string
}

export type GetSecretResponse = {
  data?: Secret
}

export type CreateSecretRequest = {
  cluster?: string
  namespace?: string
  data?: Secret
}

export type CreateSecretResponse = {
  data?: Secret
}

export type DeleteSecretRequest = {
  cluster?: string
  namespace?: string
  name?: string
}