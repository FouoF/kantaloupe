/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/

import * as KantaloupeDynamiaAiApiTypesObjectmeta from "../../types/objectmeta.pb"
import * as KantaloupeDynamiaAiApiTypesPage from "../../types/page.pb"

export enum CredentialType {
  CREDENTIAL_TYPE_UNSPECIFIED = "CREDENTIAL_TYPE_UNSPECIFIED",
  DOCKER_REGISTRY = "DOCKER_REGISTRY",
  ACCESS_KEY = "ACCESS_KEY",
}

export type CredentialSpec = {
  type?: CredentialType
  data?: {[key: string]: string}
}

export type Credential = {
  metadata?: KantaloupeDynamiaAiApiTypesObjectmeta.ObjectMeta
  spec?: CredentialSpec
}

export type CredentialResponse = {
  name?: string
  type?: CredentialType
  namespace?: string
  createdTime?: string
  labels?: {[key: string]: string}
}

export type ListCredentialsRequest = {
  type?: CredentialType
  namespace?: string
  page?: number
  pageSize?: number
  cluster?: string
}

export type ListCredentialsResponse = {
  items?: CredentialResponse[]
  pagination?: KantaloupeDynamiaAiApiTypesPage.Pagination
}

export type DeleteCredentialRequest = {
  name?: string
  namespace?: string
  cluster?: string
}

export type CreateCredentialRequest = {
  name?: string
  namespace?: string
  cluster?: string
  type?: CredentialType
  data?: {[key: string]: string}
}

export type UpdateCredentialRequest = {
  name?: string
  namespace?: string
  cluster?: string
  type?: CredentialType
  data?: {[key: string]: string}
}