/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/

import * as fm from "../../fetch.pb"
import * as GoogleProtobufEmpty from "../../google/api/empty.pb"
import * as KantaloupeDynamiaAiApiAcceleratorcardV1alpha1Acceleratorcard from "../acceleratorcard/v1alpha1/acceleratorcard.pb"
import * as KantaloupeDynamiaAiApiClustersV1alpha1Cluster from "../clusters/v1alpha1/cluster.pb"
import * as KantaloupeDynamiaAiApiCoreV1alpha1Configmap from "../core/v1alpha1/configmap.pb"
import * as KantaloupeDynamiaAiApiCoreV1alpha1Event from "../core/v1alpha1/event.pb"
import * as KantaloupeDynamiaAiApiCoreV1alpha1Gpu from "../core/v1alpha1/gpu.pb"
import * as KantaloupeDynamiaAiApiCoreV1alpha1Namespace from "../core/v1alpha1/namespace.pb"
import * as KantaloupeDynamiaAiApiCoreV1alpha1Node from "../core/v1alpha1/node.pb"
import * as KantaloupeDynamiaAiApiCoreV1alpha1Persistentvolume from "../core/v1alpha1/persistentvolume.pb"
import * as KantaloupeDynamiaAiApiCoreV1alpha1Secret from "../core/v1alpha1/secret.pb"
import * as KantaloupeDynamiaAiApiCredentialsV1alpha1Credential from "../credentials/v1alpha1/credential.pb"
import * as KantaloupeDynamiaAiApiKantaloupeflowV1alpha1Kantaloupeflow from "../kantaloupeflow/v1alpha1/kantaloupeflow.pb"
import * as KantaloupeDynamiaAiApiMonitoringV1alpha1Monitoring from "../monitoring/v1alpha1/monitoring.pb"
import * as KantaloupeDynamiaAiApiQuotasV1alpha1Quota from "../quotas/v1alpha1/quota.pb"
import * as KantaloupeDynamiaAiApiStorageV1alpha1Storage from "../storage/v1alpha1/storage.pb"
import * as KantaloupeDynamiaAiApiStorageV1alpha1Storageclass from "../storage/v1alpha1/storageclass.pb"
export class Cluster {
  static ListClusters(req: KantaloupeDynamiaAiApiClustersV1alpha1Cluster.ListClustersRequest, initReq?: fm.InitReq): Promise<KantaloupeDynamiaAiApiClustersV1alpha1Cluster.ListClustersResponse> {
    return fm.fetchReq<KantaloupeDynamiaAiApiClustersV1alpha1Cluster.ListClustersRequest, KantaloupeDynamiaAiApiClustersV1alpha1Cluster.ListClustersResponse>(`/apis/kantaloupe.dynamia.ai/v1/clusters?${fm.renderURLSearchParams(req, [])}`, {...initReq, method: "GET"})
  }
  static IntegrateCluster(req: KantaloupeDynamiaAiApiClustersV1alpha1Cluster.IntegrateClusterRequest, initReq?: fm.InitReq): Promise<KantaloupeDynamiaAiApiClustersV1alpha1Cluster.Cluster> {
    return fm.fetchReq<KantaloupeDynamiaAiApiClustersV1alpha1Cluster.IntegrateClusterRequest, KantaloupeDynamiaAiApiClustersV1alpha1Cluster.Cluster>(`/apis/kantaloupe.dynamia.ai/v1/clusters`, {...initReq, method: "POST", body: JSON.stringify(req, fm.replacer)})
  }
  static GetCluster(req: KantaloupeDynamiaAiApiClustersV1alpha1Cluster.GetClusterRequest, initReq?: fm.InitReq): Promise<KantaloupeDynamiaAiApiClustersV1alpha1Cluster.Cluster> {
    return fm.fetchReq<KantaloupeDynamiaAiApiClustersV1alpha1Cluster.GetClusterRequest, KantaloupeDynamiaAiApiClustersV1alpha1Cluster.Cluster>(`/apis/kantaloupe.dynamia.ai/v1/clusters/${req["name"]}?${fm.renderURLSearchParams(req, ["name"])}`, {...initReq, method: "GET"})
  }
  static UpdateCluster(req: KantaloupeDynamiaAiApiClustersV1alpha1Cluster.UpdateClusterRequest, initReq?: fm.InitReq): Promise<GoogleProtobufEmpty.Empty> {
    return fm.fetchReq<KantaloupeDynamiaAiApiClustersV1alpha1Cluster.UpdateClusterRequest, GoogleProtobufEmpty.Empty>(`/apis/kantaloupe.dynamia.ai/v1/clusters/${req["name"]}`, {...initReq, method: "PUT", body: JSON.stringify(req, fm.replacer)})
  }
  static DeleteCluster(req: KantaloupeDynamiaAiApiClustersV1alpha1Cluster.DeleteClusterRequest, initReq?: fm.InitReq): Promise<GoogleProtobufEmpty.Empty> {
    return fm.fetchReq<KantaloupeDynamiaAiApiClustersV1alpha1Cluster.DeleteClusterRequest, GoogleProtobufEmpty.Empty>(`/apis/kantaloupe.dynamia.ai/v1/clusters/${req["name"]}`, {...initReq, method: "DELETE", body: JSON.stringify(req, fm.replacer)})
  }
  static ValidateKubeconfig(req: KantaloupeDynamiaAiApiClustersV1alpha1Cluster.ValidateKubeconfigRequest, initReq?: fm.InitReq): Promise<KantaloupeDynamiaAiApiClustersV1alpha1Cluster.ValidateKubeconfigResponse> {
    return fm.fetchReq<KantaloupeDynamiaAiApiClustersV1alpha1Cluster.ValidateKubeconfigRequest, KantaloupeDynamiaAiApiClustersV1alpha1Cluster.ValidateKubeconfigResponse>(`/apis/kantaloupe.dynamia.ai/v1/clusters/kubeconfig:validate`, {...initReq, method: "POST", body: JSON.stringify(req, fm.replacer)})
  }
  static GetPlatformSummury(req: KantaloupeDynamiaAiApiClustersV1alpha1Cluster.GetPlatformSummuryRequest, initReq?: fm.InitReq): Promise<KantaloupeDynamiaAiApiClustersV1alpha1Cluster.PlatformSummury> {
    return fm.fetchReq<KantaloupeDynamiaAiApiClustersV1alpha1Cluster.GetPlatformSummuryRequest, KantaloupeDynamiaAiApiClustersV1alpha1Cluster.PlatformSummury>(`/apis/kantaloupe.dynamia.ai/v1/platform/summury?${fm.renderURLSearchParams(req, [])}`, {...initReq, method: "GET"})
  }
  static ListClusterVersions(req: GoogleProtobufEmpty.Empty, initReq?: fm.InitReq): Promise<KantaloupeDynamiaAiApiClustersV1alpha1Cluster.ListClusterVersionsResponse> {
    return fm.fetchReq<GoogleProtobufEmpty.Empty, KantaloupeDynamiaAiApiClustersV1alpha1Cluster.ListClusterVersionsResponse>(`/apis/kantaloupe.dynamia.ai/v1/clusters/versions?${fm.renderURLSearchParams(req, [])}`, {...initReq, method: "GET"})
  }
  static GetPlatformResourceTrend(req: GoogleProtobufEmpty.Empty, initReq?: fm.InitReq): Promise<KantaloupeDynamiaAiApiMonitoringV1alpha1Monitoring.ResourceTrendResponse> {
    return fm.fetchReq<GoogleProtobufEmpty.Empty, KantaloupeDynamiaAiApiMonitoringV1alpha1Monitoring.ResourceTrendResponse>(`/apis/kantaloupe.dynamia.ai/v1/platform/resource/trend?${fm.renderURLSearchParams(req, [])}`, {...initReq, method: "GET"})
  }
  static GetPlatformGPUTop(req: KantaloupeDynamiaAiApiClustersV1alpha1Cluster.GetPlatformGPUTopRequest, initReq?: fm.InitReq): Promise<KantaloupeDynamiaAiApiClustersV1alpha1Cluster.GetPlatformGPUTopResponse> {
    return fm.fetchReq<KantaloupeDynamiaAiApiClustersV1alpha1Cluster.GetPlatformGPUTopRequest, KantaloupeDynamiaAiApiClustersV1alpha1Cluster.GetPlatformGPUTopResponse>(`/apis/kantaloupe.dynamia.ai/v1/platform/gpu/top?${fm.renderURLSearchParams(req, [])}`, {...initReq, method: "GET"})
  }
  static GetClusterPlugins(req: KantaloupeDynamiaAiApiClustersV1alpha1Cluster.GetClusterPluginsRequest, initReq?: fm.InitReq): Promise<KantaloupeDynamiaAiApiClustersV1alpha1Cluster.GetClusterPluginsResponse> {
    return fm.fetchReq<KantaloupeDynamiaAiApiClustersV1alpha1Cluster.GetClusterPluginsRequest, KantaloupeDynamiaAiApiClustersV1alpha1Cluster.GetClusterPluginsResponse>(`/apis/kantaloupe.dynamia.ai/v1/clusters/${req["name"]}/plugins?${fm.renderURLSearchParams(req, ["name"])}`, {...initReq, method: "GET"})
  }
  static GetClusterCardRequestType(req: KantaloupeDynamiaAiApiClustersV1alpha1Cluster.GetClusterCardRequestTypeRequest, initReq?: fm.InitReq): Promise<KantaloupeDynamiaAiApiClustersV1alpha1Cluster.GetClusterCardRequestTypeResponse> {
    return fm.fetchReq<KantaloupeDynamiaAiApiClustersV1alpha1Cluster.GetClusterCardRequestTypeRequest, KantaloupeDynamiaAiApiClustersV1alpha1Cluster.GetClusterCardRequestTypeResponse>(`/apis/kantaloupe.dynamia.ai/v1/clusters/${req["name"]}/requesttype?${fm.renderURLSearchParams(req, ["name"])}`, {...initReq, method: "GET"})
  }
}
export class Core {
  static ListPersistentVolumes(req: KantaloupeDynamiaAiApiCoreV1alpha1Persistentvolume.ListPersistentVolumesRequest, initReq?: fm.InitReq): Promise<KantaloupeDynamiaAiApiCoreV1alpha1Persistentvolume.ListPersistentVolumesResponse> {
    return fm.fetchReq<KantaloupeDynamiaAiApiCoreV1alpha1Persistentvolume.ListPersistentVolumesRequest, KantaloupeDynamiaAiApiCoreV1alpha1Persistentvolume.ListPersistentVolumesResponse>(`/apis/kantaloupe.dynamia.ai/v1/clusters/${req["cluster"]}/persistentvolumes?${fm.renderURLSearchParams(req, ["cluster"])}`, {...initReq, method: "GET"})
  }
  static GetPersistentVolume(req: KantaloupeDynamiaAiApiCoreV1alpha1Persistentvolume.GetPersistentVolumeRequest, initReq?: fm.InitReq): Promise<KantaloupeDynamiaAiApiCoreV1alpha1Persistentvolume.GetPersistentVolumeResponse> {
    return fm.fetchReq<KantaloupeDynamiaAiApiCoreV1alpha1Persistentvolume.GetPersistentVolumeRequest, KantaloupeDynamiaAiApiCoreV1alpha1Persistentvolume.GetPersistentVolumeResponse>(`/apis/kantaloupe.dynamia.ai/v1/clusters/${req["cluster"]}/persistentvolumes/${req["name"]}?${fm.renderURLSearchParams(req, ["cluster", "name"])}`, {...initReq, method: "GET"})
  }
  static GetPersistentVolumeJSON(req: KantaloupeDynamiaAiApiCoreV1alpha1Persistentvolume.GetPersistentVolumeJSONRequest, initReq?: fm.InitReq): Promise<KantaloupeDynamiaAiApiCoreV1alpha1Persistentvolume.GetPersistentVolumeResponse> {
    return fm.fetchReq<KantaloupeDynamiaAiApiCoreV1alpha1Persistentvolume.GetPersistentVolumeJSONRequest, KantaloupeDynamiaAiApiCoreV1alpha1Persistentvolume.GetPersistentVolumeResponse>(`/apis/kantaloupe.dynamia.ai/v1/clusters/${req["cluster"]}/persistentvolumes/${req["name"]}/json?${fm.renderURLSearchParams(req, ["cluster", "name"])}`, {...initReq, method: "GET"})
  }
  static CreatePersistentVolume(req: KantaloupeDynamiaAiApiCoreV1alpha1Persistentvolume.CreatePersistentVolumeRequest, initReq?: fm.InitReq): Promise<KantaloupeDynamiaAiApiCoreV1alpha1Persistentvolume.CreatePersistentVolumeResponse> {
    return fm.fetchReq<KantaloupeDynamiaAiApiCoreV1alpha1Persistentvolume.CreatePersistentVolumeRequest, KantaloupeDynamiaAiApiCoreV1alpha1Persistentvolume.CreatePersistentVolumeResponse>(`/apis/kantaloupe.dynamia.ai/v1/clusters/${req["cluster"]}/persistentvolumes`, {...initReq, method: "POST", body: JSON.stringify(req, fm.replacer)})
  }
  static UpdatePersistentVolume(req: KantaloupeDynamiaAiApiCoreV1alpha1Persistentvolume.UpdatePersistentVolumeRequest, initReq?: fm.InitReq): Promise<KantaloupeDynamiaAiApiCoreV1alpha1Persistentvolume.UpdatePersistentVolumeResponse> {
    return fm.fetchReq<KantaloupeDynamiaAiApiCoreV1alpha1Persistentvolume.UpdatePersistentVolumeRequest, KantaloupeDynamiaAiApiCoreV1alpha1Persistentvolume.UpdatePersistentVolumeResponse>(`/apis/kantaloupe.dynamia.ai/v1/clusters/${req["cluster"]}/persistentvolumes/${req["name"]}`, {...initReq, method: "PUT", body: JSON.stringify(req, fm.replacer)})
  }
  static DeletePersistentVolume(req: KantaloupeDynamiaAiApiCoreV1alpha1Persistentvolume.DeletePersistentVolumeRequest, initReq?: fm.InitReq): Promise<GoogleProtobufEmpty.Empty> {
    return fm.fetchReq<KantaloupeDynamiaAiApiCoreV1alpha1Persistentvolume.DeletePersistentVolumeRequest, GoogleProtobufEmpty.Empty>(`/apis/kantaloupe.dynamia.ai/v1/clusters/${req["cluster"]}/persistentvolumes/${req["name"]}`, {...initReq, method: "DELETE"})
  }
  static DeleteSecret(req: KantaloupeDynamiaAiApiCoreV1alpha1Secret.DeleteSecretRequest, initReq?: fm.InitReq): Promise<GoogleProtobufEmpty.Empty> {
    return fm.fetchReq<KantaloupeDynamiaAiApiCoreV1alpha1Secret.DeleteSecretRequest, GoogleProtobufEmpty.Empty>(`/apis/kantaloupe.dynamia.ai/v1/clusters/${req["cluster"]}/namespaces/${req["namespace"]}/secrets/${req["name"]}`, {...initReq, method: "DELETE"})
  }
  static GetSecret(req: KantaloupeDynamiaAiApiCoreV1alpha1Secret.GetSecretRequest, initReq?: fm.InitReq): Promise<KantaloupeDynamiaAiApiCoreV1alpha1Secret.Secret> {
    return fm.fetchReq<KantaloupeDynamiaAiApiCoreV1alpha1Secret.GetSecretRequest, KantaloupeDynamiaAiApiCoreV1alpha1Secret.Secret>(`/apis/kantaloupe.dynamia.ai/v1/clusters/${req["cluster"]}/namespaces/${req["namespace"]}/secrets/${req["name"]}?${fm.renderURLSearchParams(req, ["cluster", "namespace", "name"])}`, {...initReq, method: "GET"})
  }
  static ListSecrets(req: KantaloupeDynamiaAiApiCoreV1alpha1Secret.ListSecretsRequest, initReq?: fm.InitReq): Promise<KantaloupeDynamiaAiApiCoreV1alpha1Secret.ListSecretsResponse> {
    return fm.fetchReq<KantaloupeDynamiaAiApiCoreV1alpha1Secret.ListSecretsRequest, KantaloupeDynamiaAiApiCoreV1alpha1Secret.ListSecretsResponse>(`/apis/kantaloupe.dynamia.ai/v1/clusters/${req["cluster"]}/secrets?${fm.renderURLSearchParams(req, ["cluster"])}`, {...initReq, method: "GET"})
  }
  static CreateSecret(req: KantaloupeDynamiaAiApiCoreV1alpha1Secret.CreateSecretRequest, initReq?: fm.InitReq): Promise<KantaloupeDynamiaAiApiCoreV1alpha1Secret.CreateSecretResponse> {
    return fm.fetchReq<KantaloupeDynamiaAiApiCoreV1alpha1Secret.CreateSecretRequest, KantaloupeDynamiaAiApiCoreV1alpha1Secret.CreateSecretResponse>(`/apis/kantaloupe.dynamia.ai/v1/clusters/${req["cluster"]}/namespaces/${req["namespace"]}/secrets`, {...initReq, method: "POST", body: JSON.stringify(req, fm.replacer)})
  }
  static ListClusterNamespaces(req: KantaloupeDynamiaAiApiCoreV1alpha1Namespace.ListClusterNamespacesRequest, initReq?: fm.InitReq): Promise<KantaloupeDynamiaAiApiCoreV1alpha1Namespace.ListClusterNamespacesResponse> {
    return fm.fetchReq<KantaloupeDynamiaAiApiCoreV1alpha1Namespace.ListClusterNamespacesRequest, KantaloupeDynamiaAiApiCoreV1alpha1Namespace.ListClusterNamespacesResponse>(`/apis/kantaloupe.dynamia.ai/v1/clusters/${req["cluster"]}/namespaces?${fm.renderURLSearchParams(req, ["cluster"])}`, {...initReq, method: "GET"})
  }
  static ListClusterGPUSummary(req: KantaloupeDynamiaAiApiCoreV1alpha1Gpu.ListClusterGPUSummaryRequest, initReq?: fm.InitReq): Promise<KantaloupeDynamiaAiApiCoreV1alpha1Gpu.ListClusterGPUSummaryResponse> {
    return fm.fetchReq<KantaloupeDynamiaAiApiCoreV1alpha1Gpu.ListClusterGPUSummaryRequest, KantaloupeDynamiaAiApiCoreV1alpha1Gpu.ListClusterGPUSummaryResponse>(`/apis/kantaloupe.dynamia.ai/v1/clusters/${req["cluster"]}/gpusummary?${fm.renderURLSearchParams(req, ["cluster"])}`, {...initReq, method: "GET"})
  }
  static ListClusterEvents(req: KantaloupeDynamiaAiApiCoreV1alpha1Event.ListClusterEventsRequest, initReq?: fm.InitReq): Promise<KantaloupeDynamiaAiApiCoreV1alpha1Event.ListClusterEventsResponse> {
    return fm.fetchReq<KantaloupeDynamiaAiApiCoreV1alpha1Event.ListClusterEventsRequest, KantaloupeDynamiaAiApiCoreV1alpha1Event.ListClusterEventsResponse>(`/apis/kantaloupe.dynamia.ai/v1/clusters/${req["cluster"]}/events?${fm.renderURLSearchParams(req, ["cluster"])}`, {...initReq, method: "GET"})
  }
  static ListEvents(req: KantaloupeDynamiaAiApiCoreV1alpha1Event.ListEventsRequest, initReq?: fm.InitReq): Promise<KantaloupeDynamiaAiApiCoreV1alpha1Event.ListEventsResponse> {
    return fm.fetchReq<KantaloupeDynamiaAiApiCoreV1alpha1Event.ListEventsRequest, KantaloupeDynamiaAiApiCoreV1alpha1Event.ListEventsResponse>(`/apis/kantaloupe.dynamia.ai/v1/clusters/${req["cluster"]}/namespaces/${req["namespace"]}/events?${fm.renderURLSearchParams(req, ["cluster", "namespace"])}`, {...initReq, method: "GET"})
  }
  static ListNodes(req: KantaloupeDynamiaAiApiCoreV1alpha1Node.ListNodesRequest, initReq?: fm.InitReq): Promise<KantaloupeDynamiaAiApiCoreV1alpha1Node.ListNodesResponse> {
    return fm.fetchReq<KantaloupeDynamiaAiApiCoreV1alpha1Node.ListNodesRequest, KantaloupeDynamiaAiApiCoreV1alpha1Node.ListNodesResponse>(`/apis/kantaloupe.dynamia.ai/v1/clusters/${req["cluster"]}/nodes?${fm.renderURLSearchParams(req, ["cluster"])}`, {...initReq, method: "GET"})
  }
  static GetNode(req: KantaloupeDynamiaAiApiCoreV1alpha1Node.GetNodeRequest, initReq?: fm.InitReq): Promise<KantaloupeDynamiaAiApiCoreV1alpha1Node.Node> {
    return fm.fetchReq<KantaloupeDynamiaAiApiCoreV1alpha1Node.GetNodeRequest, KantaloupeDynamiaAiApiCoreV1alpha1Node.Node>(`/apis/kantaloupe.dynamia.ai/v1/clusters/${req["cluster"]}/nodes/${req["name"]}?${fm.renderURLSearchParams(req, ["cluster", "name"])}`, {...initReq, method: "GET"})
  }
  static PutNodeLabels(req: KantaloupeDynamiaAiApiCoreV1alpha1Node.PutNodeLabelsRequest, initReq?: fm.InitReq): Promise<KantaloupeDynamiaAiApiCoreV1alpha1Node.PutNodeLabelsResponse> {
    return fm.fetchReq<KantaloupeDynamiaAiApiCoreV1alpha1Node.PutNodeLabelsRequest, KantaloupeDynamiaAiApiCoreV1alpha1Node.PutNodeLabelsResponse>(`/apis/kantaloupe.dynamia.ai/v1/clusters/${req["cluster"]}/nodes/${req["node"]}/labels`, {...initReq, method: "PUT", body: JSON.stringify(req, fm.replacer)})
  }
  static PutNodeTaints(req: KantaloupeDynamiaAiApiCoreV1alpha1Node.PutNodeTaintsRequest, initReq?: fm.InitReq): Promise<KantaloupeDynamiaAiApiCoreV1alpha1Node.PutNodeTaintsResponse> {
    return fm.fetchReq<KantaloupeDynamiaAiApiCoreV1alpha1Node.PutNodeTaintsRequest, KantaloupeDynamiaAiApiCoreV1alpha1Node.PutNodeTaintsResponse>(`/apis/kantaloupe.dynamia.ai/v1/clusters/${req["cluster"]}/nodes/${req["node"]}/taints`, {...initReq, method: "PUT", body: JSON.stringify(req, fm.replacer)})
  }
  static UpdateNodeAnnotations(req: KantaloupeDynamiaAiApiCoreV1alpha1Node.UpdateNodeAnnotationsRequest, initReq?: fm.InitReq): Promise<KantaloupeDynamiaAiApiCoreV1alpha1Node.UpdateNodeAnnotationsResponse> {
    return fm.fetchReq<KantaloupeDynamiaAiApiCoreV1alpha1Node.UpdateNodeAnnotationsRequest, KantaloupeDynamiaAiApiCoreV1alpha1Node.UpdateNodeAnnotationsResponse>(`/apis/kantaloupe.dynamia.ai/v1/clusters/${req["cluster"]}/nodes/${req["node"]}/annotations`, {...initReq, method: "PUT", body: JSON.stringify(req, fm.replacer)})
  }
  static UnScheduleNode(req: KantaloupeDynamiaAiApiCoreV1alpha1Node.ScheduleNodeRequest, initReq?: fm.InitReq): Promise<KantaloupeDynamiaAiApiCoreV1alpha1Node.Node> {
    return fm.fetchReq<KantaloupeDynamiaAiApiCoreV1alpha1Node.ScheduleNodeRequest, KantaloupeDynamiaAiApiCoreV1alpha1Node.Node>(`/apis/kantaloupe.dynamia.ai/v1/clusters/${req["cluster"]}/nodes/${req["node"]}/unschedule`, {...initReq, method: "POST"})
  }
  static ScheduleNode(req: KantaloupeDynamiaAiApiCoreV1alpha1Node.ScheduleNodeRequest, initReq?: fm.InitReq): Promise<KantaloupeDynamiaAiApiCoreV1alpha1Node.Node> {
    return fm.fetchReq<KantaloupeDynamiaAiApiCoreV1alpha1Node.ScheduleNodeRequest, KantaloupeDynamiaAiApiCoreV1alpha1Node.Node>(`/apis/kantaloupe.dynamia.ai/v1/clusters/${req["cluster"]}/nodes/${req["node"]}/schedule`, {...initReq, method: "POST"})
  }
  static GetConfigMap(req: KantaloupeDynamiaAiApiCoreV1alpha1Configmap.GetConfigMapRequest, initReq?: fm.InitReq): Promise<KantaloupeDynamiaAiApiCoreV1alpha1Configmap.ConfigMap> {
    return fm.fetchReq<KantaloupeDynamiaAiApiCoreV1alpha1Configmap.GetConfigMapRequest, KantaloupeDynamiaAiApiCoreV1alpha1Configmap.ConfigMap>(`/apis/kantaloupe.dynamia.ai/v1/clusters/${req["cluster"]}/namespaces/${req["namespace"]}/configmaps/${req["name"]}?${fm.renderURLSearchParams(req, ["cluster", "namespace", "name"])}`, {...initReq, method: "GET"})
  }
  static GetConfigMapJSON(req: KantaloupeDynamiaAiApiCoreV1alpha1Configmap.GetConfigMapJSONRequest, initReq?: fm.InitReq): Promise<KantaloupeDynamiaAiApiCoreV1alpha1Configmap.GetConfigMapJSONResponse> {
    return fm.fetchReq<KantaloupeDynamiaAiApiCoreV1alpha1Configmap.GetConfigMapJSONRequest, KantaloupeDynamiaAiApiCoreV1alpha1Configmap.GetConfigMapJSONResponse>(`/apis/kantaloupe.dynamia.ai/v1/clusters/${req["cluster"]}/namespaces/${req["namespace"]}/configmaps/${req["name"]}/json?${fm.renderURLSearchParams(req, ["cluster", "namespace", "name"])}`, {...initReq, method: "GET"})
  }
  static UpdateConfigMap(req: KantaloupeDynamiaAiApiCoreV1alpha1Configmap.UpdateConfigMapRequest, initReq?: fm.InitReq): Promise<KantaloupeDynamiaAiApiCoreV1alpha1Configmap.UpdateConfigMapResponse> {
    return fm.fetchReq<KantaloupeDynamiaAiApiCoreV1alpha1Configmap.UpdateConfigMapRequest, KantaloupeDynamiaAiApiCoreV1alpha1Configmap.UpdateConfigMapResponse>(`/apis/kantaloupe.dynamia.ai/v1/clusters/${req["cluster"]}/namespaces/${req["namespace"]}/configmaps/${req["name"]}`, {...initReq, method: "PUT", body: JSON.stringify(req, fm.replacer)})
  }
}
export class Monitoring {
  static ListAllPodsGPUUtilization(req: KantaloupeDynamiaAiApiMonitoringV1alpha1Monitoring.ListMonitoringsRequest, initReq?: fm.InitReq): Promise<KantaloupeDynamiaAiApiMonitoringV1alpha1Monitoring.ListMonitoringsResponse> {
    return fm.fetchReq<KantaloupeDynamiaAiApiMonitoringV1alpha1Monitoring.ListMonitoringsRequest, KantaloupeDynamiaAiApiMonitoringV1alpha1Monitoring.ListMonitoringsResponse>(`/apis/kantaloupe.dynamia.ai/v1/clusters/${req["cluster"]}/monitoring/gpuUtilization?${fm.renderURLSearchParams(req, ["cluster"])}`, {...initReq, method: "GET"})
  }
  static GetResourceTrend(req: KantaloupeDynamiaAiApiMonitoringV1alpha1Monitoring.ResourceTrendRequest, initReq?: fm.InitReq): Promise<KantaloupeDynamiaAiApiMonitoringV1alpha1Monitoring.ResourceTrendResponse> {
    return fm.fetchReq<KantaloupeDynamiaAiApiMonitoringV1alpha1Monitoring.ResourceTrendRequest, KantaloupeDynamiaAiApiMonitoringV1alpha1Monitoring.ResourceTrendResponse>(`/apis/kantaloupe.dynamia.ai/v1/clusters/${req["cluster"]}/resource/trend?${fm.renderURLSearchParams(req, ["cluster"])}`, {...initReq, method: "GET"})
  }
  static GetNodeResourceTrend(req: KantaloupeDynamiaAiApiMonitoringV1alpha1Monitoring.NodeResourceTrendRequest, initReq?: fm.InitReq): Promise<KantaloupeDynamiaAiApiMonitoringV1alpha1Monitoring.ResourceTrendResponse> {
    return fm.fetchReq<KantaloupeDynamiaAiApiMonitoringV1alpha1Monitoring.NodeResourceTrendRequest, KantaloupeDynamiaAiApiMonitoringV1alpha1Monitoring.ResourceTrendResponse>(`/apis/kantaloupe.dynamia.ai/v1/clusters/${req["cluster"]}/nodes/${req["node"]}/resource/trend?${fm.renderURLSearchParams(req, ["cluster", "node"])}`, {...initReq, method: "GET"})
  }
  static GetGpuResourceTrend(req: KantaloupeDynamiaAiApiMonitoringV1alpha1Monitoring.GpuResourceTrendRequest, initReq?: fm.InitReq): Promise<KantaloupeDynamiaAiApiMonitoringV1alpha1Monitoring.ResourceTrendResponse> {
    return fm.fetchReq<KantaloupeDynamiaAiApiMonitoringV1alpha1Monitoring.GpuResourceTrendRequest, KantaloupeDynamiaAiApiMonitoringV1alpha1Monitoring.ResourceTrendResponse>(`/apis/kantaloupe.dynamia.ai/v1/clusters/${req["cluster"]}/nodes/${req["node"]}/gpus/${req["uuid"]}/resource/trend?${fm.renderURLSearchParams(req, ["cluster", "node", "uuid"])}`, {...initReq, method: "GET"})
  }
  static GetKantaloupeflowResourceTrend(req: KantaloupeDynamiaAiApiMonitoringV1alpha1Monitoring.KantaloupeflowResourceTrendRequest, initReq?: fm.InitReq): Promise<KantaloupeDynamiaAiApiMonitoringV1alpha1Monitoring.ResourceTrendResponse> {
    return fm.fetchReq<KantaloupeDynamiaAiApiMonitoringV1alpha1Monitoring.KantaloupeflowResourceTrendRequest, KantaloupeDynamiaAiApiMonitoringV1alpha1Monitoring.ResourceTrendResponse>(`/apis/kantaloupe.dynamia.ai/v1/clusters/${req["cluster"]}/namespaces/${req["namespace"]}/kantaloupeflows/${req["name"]}/resource/trend?${fm.renderURLSearchParams(req, ["cluster", "namespace", "name"])}`, {...initReq, method: "GET"})
  }
  static GetNodeWorkloadDistribution(req: KantaloupeDynamiaAiApiMonitoringV1alpha1Monitoring.NodeWorkloadDistributionRequest, initReq?: fm.InitReq): Promise<KantaloupeDynamiaAiApiMonitoringV1alpha1Monitoring.WorkloadDistributionResponse> {
    return fm.fetchReq<KantaloupeDynamiaAiApiMonitoringV1alpha1Monitoring.NodeWorkloadDistributionRequest, KantaloupeDynamiaAiApiMonitoringV1alpha1Monitoring.WorkloadDistributionResponse>(`/apis/kantaloupe.dynamia.ai/v1/clusters/${req["cluster"]}/nodes/${req["node"]}/workloads/distribution?${fm.renderURLSearchParams(req, ["cluster", "node"])}`, {...initReq, method: "GET"})
  }
  static GetClusterWorkloadDistribution(req: KantaloupeDynamiaAiApiMonitoringV1alpha1Monitoring.ClusterWorkloadDistributionRequest, initReq?: fm.InitReq): Promise<KantaloupeDynamiaAiApiMonitoringV1alpha1Monitoring.WorkloadDistributionResponse> {
    return fm.fetchReq<KantaloupeDynamiaAiApiMonitoringV1alpha1Monitoring.ClusterWorkloadDistributionRequest, KantaloupeDynamiaAiApiMonitoringV1alpha1Monitoring.WorkloadDistributionResponse>(`/apis/kantaloupe.dynamia.ai/v1/clusters/${req["cluster"]}/workloads/distribution?${fm.renderURLSearchParams(req, ["cluster"])}`, {...initReq, method: "GET"})
  }
  static GetTopNodes(req: KantaloupeDynamiaAiApiMonitoringV1alpha1Monitoring.TopNodeRequest, initReq?: fm.InitReq): Promise<KantaloupeDynamiaAiApiMonitoringV1alpha1Monitoring.TopNodeResponse> {
    return fm.fetchReq<KantaloupeDynamiaAiApiMonitoringV1alpha1Monitoring.TopNodeRequest, KantaloupeDynamiaAiApiMonitoringV1alpha1Monitoring.TopNodeResponse>(`/apis/kantaloupe.dynamia.ai/v1/clusters/${req["cluster"]}/nodes/top?${fm.renderURLSearchParams(req, ["cluster"])}`, {...initReq, method: "GET"})
  }
  static GetTopNodeWorkloads(req: KantaloupeDynamiaAiApiMonitoringV1alpha1Monitoring.TopNodeWorkloadRequest, initReq?: fm.InitReq): Promise<KantaloupeDynamiaAiApiMonitoringV1alpha1Monitoring.TopNodeResponse> {
    return fm.fetchReq<KantaloupeDynamiaAiApiMonitoringV1alpha1Monitoring.TopNodeWorkloadRequest, KantaloupeDynamiaAiApiMonitoringV1alpha1Monitoring.TopNodeResponse>(`/apis/kantaloupe.dynamia.ai/v1/clusters/${req["cluster"]}/nodeWorkloads/top?${fm.renderURLSearchParams(req, ["cluster"])}`, {...initReq, method: "GET"})
  }
  static GetKantaloupeflowMemoryDistribution(req: KantaloupeDynamiaAiApiMonitoringV1alpha1Monitoring.MemoryDistributionRequest, initReq?: fm.InitReq): Promise<KantaloupeDynamiaAiApiMonitoringV1alpha1Monitoring.MemoryDistributionResponse> {
    return fm.fetchReq<KantaloupeDynamiaAiApiMonitoringV1alpha1Monitoring.MemoryDistributionRequest, KantaloupeDynamiaAiApiMonitoringV1alpha1Monitoring.MemoryDistributionResponse>(`/apis/kantaloupe.dynamia.ai/v1/clusters/${req["cluster"]}/namespace/${req["namespace"]}/kantaloupeflow/${req["name"]}/memory/distribution?${fm.renderURLSearchParams(req, ["cluster", "namespace", "name"])}`, {...initReq, method: "GET"})
  }
  static GetCardTopWorkloads(req: KantaloupeDynamiaAiApiMonitoringV1alpha1Monitoring.CardTopWorkloadsRequest, initReq?: fm.InitReq): Promise<KantaloupeDynamiaAiApiMonitoringV1alpha1Monitoring.CardTopWorkloadsResponse> {
    return fm.fetchReq<KantaloupeDynamiaAiApiMonitoringV1alpha1Monitoring.CardTopWorkloadsRequest, KantaloupeDynamiaAiApiMonitoringV1alpha1Monitoring.CardTopWorkloadsResponse>(`/apis/kantaloupe.dynamia.ai/v1/clusters/${req["cluster"]}/nodes/${req["node"]}/uuids/${req["uuid"]}/top?${fm.renderURLSearchParams(req, ["cluster", "node", "uuid"])}`, {...initReq, method: "GET"})
  }
  static GetClusterWorkloadsTop(req: KantaloupeDynamiaAiApiMonitoringV1alpha1Monitoring.GetClusterWorkloadsTopRequest, initReq?: fm.InitReq): Promise<KantaloupeDynamiaAiApiMonitoringV1alpha1Monitoring.GetClusterWorkloadsTopResponse> {
    return fm.fetchReq<KantaloupeDynamiaAiApiMonitoringV1alpha1Monitoring.GetClusterWorkloadsTopRequest, KantaloupeDynamiaAiApiMonitoringV1alpha1Monitoring.GetClusterWorkloadsTopResponse>(`/apis/kantaloupe.dynamia.ai/v1/clusters/${req["cluster"]}/workloads/top?${fm.renderURLSearchParams(req, ["cluster"])}`, {...initReq, method: "GET"})
  }
}
export class Kantaloupeflow {
  static CreateKantaloupeflow(req: KantaloupeDynamiaAiApiKantaloupeflowV1alpha1Kantaloupeflow.CreateKantaloupeflowRequest, initReq?: fm.InitReq): Promise<KantaloupeDynamiaAiApiKantaloupeflowV1alpha1Kantaloupeflow.Kantaloupeflow> {
    return fm.fetchReq<KantaloupeDynamiaAiApiKantaloupeflowV1alpha1Kantaloupeflow.CreateKantaloupeflowRequest, KantaloupeDynamiaAiApiKantaloupeflowV1alpha1Kantaloupeflow.Kantaloupeflow>(`/apis/kantaloupe.dynamia.ai/v1/clusters/${req["cluster"]}/kantaloupeflows`, {...initReq, method: "POST", body: JSON.stringify(req, fm.replacer)})
  }
  static GetKantaloupeflow(req: KantaloupeDynamiaAiApiKantaloupeflowV1alpha1Kantaloupeflow.GetKantaloupeflowRequest, initReq?: fm.InitReq): Promise<KantaloupeDynamiaAiApiKantaloupeflowV1alpha1Kantaloupeflow.GetKantaloupeflowResponse> {
    return fm.fetchReq<KantaloupeDynamiaAiApiKantaloupeflowV1alpha1Kantaloupeflow.GetKantaloupeflowRequest, KantaloupeDynamiaAiApiKantaloupeflowV1alpha1Kantaloupeflow.GetKantaloupeflowResponse>(`/apis/kantaloupe.dynamia.ai/v1/clusters/${req["cluster"]}/namespaces/${req["namespace"]}/kantaloupeflows/${req["name"]}?${fm.renderURLSearchParams(req, ["cluster", "namespace", "name"])}`, {...initReq, method: "GET"})
  }
  static DeleteKantaloupeflow(req: KantaloupeDynamiaAiApiKantaloupeflowV1alpha1Kantaloupeflow.DeleteKantaloupeflowRequest, initReq?: fm.InitReq): Promise<GoogleProtobufEmpty.Empty> {
    return fm.fetchReq<KantaloupeDynamiaAiApiKantaloupeflowV1alpha1Kantaloupeflow.DeleteKantaloupeflowRequest, GoogleProtobufEmpty.Empty>(`/apis/kantaloupe.dynamia.ai/v1/clusters/${req["cluster"]}/namespaces/${req["namespace"]}/kantaloupeflows/${req["name"]}`, {...initReq, method: "DELETE"})
  }
  static ListKantaloupeflows(req: KantaloupeDynamiaAiApiKantaloupeflowV1alpha1Kantaloupeflow.ListKantaloupeflowsRequest, initReq?: fm.InitReq): Promise<KantaloupeDynamiaAiApiKantaloupeflowV1alpha1Kantaloupeflow.ListKantaloupeflowsResponse> {
    return fm.fetchReq<KantaloupeDynamiaAiApiKantaloupeflowV1alpha1Kantaloupeflow.ListKantaloupeflowsRequest, KantaloupeDynamiaAiApiKantaloupeflowV1alpha1Kantaloupeflow.ListKantaloupeflowsResponse>(`/apis/kantaloupe.dynamia.ai/v1/clusters/${req["cluster"]}/namespaces/${req["namespace"]}/kantaloupeflows?${fm.renderURLSearchParams(req, ["cluster", "namespace"])}`, {...initReq, method: "GET"})
  }
  static GetKantaloupeTree(req: GoogleProtobufEmpty.Empty, initReq?: fm.InitReq): Promise<KantaloupeDynamiaAiApiKantaloupeflowV1alpha1Kantaloupeflow.KantaloupeTree> {
    return fm.fetchReq<GoogleProtobufEmpty.Empty, KantaloupeDynamiaAiApiKantaloupeflowV1alpha1Kantaloupeflow.KantaloupeTree>(`/apis/kantaloupe.dynamia.ai/v1/platform/kantaloupeflows/tree?${fm.renderURLSearchParams(req, [])}`, {...initReq, method: "GET"})
  }
  static UpdateKantaloupeflowGPUMemory(req: KantaloupeDynamiaAiApiKantaloupeflowV1alpha1Kantaloupeflow.UpdateKantaloupeflowGPUMemoryRequest, initReq?: fm.InitReq): Promise<GoogleProtobufEmpty.Empty> {
    return fm.fetchReq<KantaloupeDynamiaAiApiKantaloupeflowV1alpha1Kantaloupeflow.UpdateKantaloupeflowGPUMemoryRequest, GoogleProtobufEmpty.Empty>(`/apis/kantaloupe.dynamia.ai/v1/clusters/${req["cluster"]}/namespaces/${req["namespace"]}/kantaloupeflows/${req["name"]}/gpumemory`, {...initReq, method: "POST", body: JSON.stringify(req, fm.replacer)})
  }
  static GetKantaloupeflowConditions(req: KantaloupeDynamiaAiApiKantaloupeflowV1alpha1Kantaloupeflow.GetKantaloupeflowConditionsRequest, initReq?: fm.InitReq): Promise<KantaloupeDynamiaAiApiKantaloupeflowV1alpha1Kantaloupeflow.GetKantaloupeflowConditionsResponse> {
    return fm.fetchReq<KantaloupeDynamiaAiApiKantaloupeflowV1alpha1Kantaloupeflow.GetKantaloupeflowConditionsRequest, KantaloupeDynamiaAiApiKantaloupeflowV1alpha1Kantaloupeflow.GetKantaloupeflowConditionsResponse>(`/apis/kantaloupe.dynamia.ai/v1/clusters/${req["cluster"]}/namespaces/${req["namespace"]}/kantaloupeflows/${req["name"]}/conditions?${fm.renderURLSearchParams(req, ["cluster", "namespace", "name"])}`, {...initReq, method: "GET"})
  }
}
export class Credential {
  static ListCredentials(req: KantaloupeDynamiaAiApiCredentialsV1alpha1Credential.ListCredentialsRequest, initReq?: fm.InitReq): Promise<KantaloupeDynamiaAiApiCredentialsV1alpha1Credential.ListCredentialsResponse> {
    return fm.fetchReq<KantaloupeDynamiaAiApiCredentialsV1alpha1Credential.ListCredentialsRequest, KantaloupeDynamiaAiApiCredentialsV1alpha1Credential.ListCredentialsResponse>(`/apis/kantaloupe.dynamia.ai/v1/clusters/${req["cluster"]}/credentials?${fm.renderURLSearchParams(req, ["cluster"])}`, {...initReq, method: "GET"})
  }
  static DeleteCredential(req: KantaloupeDynamiaAiApiCredentialsV1alpha1Credential.DeleteCredentialRequest, initReq?: fm.InitReq): Promise<GoogleProtobufEmpty.Empty> {
    return fm.fetchReq<KantaloupeDynamiaAiApiCredentialsV1alpha1Credential.DeleteCredentialRequest, GoogleProtobufEmpty.Empty>(`/apis/kantaloupe.dynamia.ai/v1/clusters/${req["cluster"]}/credentials/${req["name"]}`, {...initReq, method: "DELETE"})
  }
  static CreateCredential(req: KantaloupeDynamiaAiApiCredentialsV1alpha1Credential.CreateCredentialRequest, initReq?: fm.InitReq): Promise<KantaloupeDynamiaAiApiCredentialsV1alpha1Credential.CredentialResponse> {
    return fm.fetchReq<KantaloupeDynamiaAiApiCredentialsV1alpha1Credential.CreateCredentialRequest, KantaloupeDynamiaAiApiCredentialsV1alpha1Credential.CredentialResponse>(`/apis/kantaloupe.dynamia.ai/v1/clusters/${req["cluster"]}/credentials`, {...initReq, method: "POST", body: JSON.stringify(req, fm.replacer)})
  }
  static UpdateCredential(req: KantaloupeDynamiaAiApiCredentialsV1alpha1Credential.UpdateCredentialRequest, initReq?: fm.InitReq): Promise<KantaloupeDynamiaAiApiCredentialsV1alpha1Credential.CredentialResponse> {
    return fm.fetchReq<KantaloupeDynamiaAiApiCredentialsV1alpha1Credential.UpdateCredentialRequest, KantaloupeDynamiaAiApiCredentialsV1alpha1Credential.CredentialResponse>(`/apis/kantaloupe.dynamia.ai/v1/clusters/${req["cluster"]}/credentials/${req["name"]}`, {...initReq, method: "PUT", body: JSON.stringify(req, fm.replacer)})
  }
}
export class Quota {
  static ListQuotas(req: KantaloupeDynamiaAiApiQuotasV1alpha1Quota.ListQuotasRequest, initReq?: fm.InitReq): Promise<KantaloupeDynamiaAiApiQuotasV1alpha1Quota.ListQuotasResponse> {
    return fm.fetchReq<KantaloupeDynamiaAiApiQuotasV1alpha1Quota.ListQuotasRequest, KantaloupeDynamiaAiApiQuotasV1alpha1Quota.ListQuotasResponse>(`/apis/kantaloupe.dynamia.ai/v1/clusters/${req["cluster"]}/quotas?${fm.renderURLSearchParams(req, ["cluster"])}`, {...initReq, method: "GET"})
  }
  static DeleteQuota(req: KantaloupeDynamiaAiApiQuotasV1alpha1Quota.DeleteQuotaRequest, initReq?: fm.InitReq): Promise<GoogleProtobufEmpty.Empty> {
    return fm.fetchReq<KantaloupeDynamiaAiApiQuotasV1alpha1Quota.DeleteQuotaRequest, GoogleProtobufEmpty.Empty>(`/apis/kantaloupe.dynamia.ai/v1/clusters/${req["cluster"]}/namespaces/${req["namespace"]}/quotas/${req["name"]}`, {...initReq, method: "DELETE"})
  }
  static CreateQuota(req: KantaloupeDynamiaAiApiQuotasV1alpha1Quota.CreateQuotaRequest, initReq?: fm.InitReq): Promise<KantaloupeDynamiaAiApiQuotasV1alpha1Quota.QuotaResponse> {
    return fm.fetchReq<KantaloupeDynamiaAiApiQuotasV1alpha1Quota.CreateQuotaRequest, KantaloupeDynamiaAiApiQuotasV1alpha1Quota.QuotaResponse>(`/apis/kantaloupe.dynamia.ai/v1/clusters/${req["cluster"]}/namespaces/${req["namespace"]}/quotas`, {...initReq, method: "POST", body: JSON.stringify(req, fm.replacer)})
  }
  static UpdateQuota(req: KantaloupeDynamiaAiApiQuotasV1alpha1Quota.UpdateQuotaRequest, initReq?: fm.InitReq): Promise<KantaloupeDynamiaAiApiQuotasV1alpha1Quota.QuotaResponse> {
    return fm.fetchReq<KantaloupeDynamiaAiApiQuotasV1alpha1Quota.UpdateQuotaRequest, KantaloupeDynamiaAiApiQuotasV1alpha1Quota.QuotaResponse>(`/apis/kantaloupe.dynamia.ai/v1/clusters/${req["cluster"]}/namespaces/${req["namespace"]}/quotas/${req["name"]}`, {...initReq, method: "PUT", body: JSON.stringify(req, fm.replacer)})
  }
  static GetQuota(req: KantaloupeDynamiaAiApiQuotasV1alpha1Quota.GetQuotaRequest, initReq?: fm.InitReq): Promise<KantaloupeDynamiaAiApiQuotasV1alpha1Quota.QuotaResponse> {
    return fm.fetchReq<KantaloupeDynamiaAiApiQuotasV1alpha1Quota.GetQuotaRequest, KantaloupeDynamiaAiApiQuotasV1alpha1Quota.QuotaResponse>(`/apis/kantaloupe.dynamia.ai/v1/clusters/${req["cluster"]}/namespaces/${req["namespace"]}/quotas/${req["name"]}?${fm.renderURLSearchParams(req, ["cluster", "namespace", "name"])}`, {...initReq, method: "GET"})
  }
}
export class Storage {
  static ListStorageClasses(req: KantaloupeDynamiaAiApiStorageV1alpha1Storageclass.ListStorageClassesRequest, initReq?: fm.InitReq): Promise<KantaloupeDynamiaAiApiStorageV1alpha1Storageclass.ListStorageClassesResponse> {
    return fm.fetchReq<KantaloupeDynamiaAiApiStorageV1alpha1Storageclass.ListStorageClassesRequest, KantaloupeDynamiaAiApiStorageV1alpha1Storageclass.ListStorageClassesResponse>(`/apis/kantaloupe.dynamia.ai/v1/clusters/${req["cluster"]}/storageclasses?${fm.renderURLSearchParams(req, ["cluster"])}`, {...initReq, method: "GET"})
  }
  static CreateStorage(req: KantaloupeDynamiaAiApiStorageV1alpha1Storage.CreateStorageRequest, initReq?: fm.InitReq): Promise<KantaloupeDynamiaAiApiStorageV1alpha1Storage.Storage> {
    return fm.fetchReq<KantaloupeDynamiaAiApiStorageV1alpha1Storage.CreateStorageRequest, KantaloupeDynamiaAiApiStorageV1alpha1Storage.Storage>(`/apis/kantaloupe.dynamia.ai/v1/clusters/${req["cluster"]}/namespace/${req["namespace"]}/storage`, {...initReq, method: "POST", body: JSON.stringify(req, fm.replacer)})
  }
  static DeleteStorage(req: KantaloupeDynamiaAiApiStorageV1alpha1Storage.DeleteStorageRequest, initReq?: fm.InitReq): Promise<GoogleProtobufEmpty.Empty> {
    return fm.fetchReq<KantaloupeDynamiaAiApiStorageV1alpha1Storage.DeleteStorageRequest, GoogleProtobufEmpty.Empty>(`/apis/kantaloupe.dynamia.ai/v1/clusters/${req["cluster"]}/namespace/${req["namespace"]}/name/${req["name"]}/storage`, {...initReq, method: "DELETE"})
  }
  static ListStorages(req: KantaloupeDynamiaAiApiStorageV1alpha1Storage.ListStoragesRequest, initReq?: fm.InitReq): Promise<KantaloupeDynamiaAiApiStorageV1alpha1Storage.ListStoragesResponse> {
    return fm.fetchReq<KantaloupeDynamiaAiApiStorageV1alpha1Storage.ListStoragesRequest, KantaloupeDynamiaAiApiStorageV1alpha1Storage.ListStoragesResponse>(`/apis/kantaloupe.dynamia.ai/v1/clusters/${req["cluster"]}/storage?${fm.renderURLSearchParams(req, ["cluster"])}`, {...initReq, method: "GET"})
  }
}
export class AcceleratorCard {
  static ListAcceleratorCard(req: KantaloupeDynamiaAiApiAcceleratorcardV1alpha1Acceleratorcard.ListAcceleratorCardsRequest, initReq?: fm.InitReq): Promise<KantaloupeDynamiaAiApiAcceleratorcardV1alpha1Acceleratorcard.ListAcceleratorCardsResponse> {
    return fm.fetchReq<KantaloupeDynamiaAiApiAcceleratorcardV1alpha1Acceleratorcard.ListAcceleratorCardsRequest, KantaloupeDynamiaAiApiAcceleratorcardV1alpha1Acceleratorcard.ListAcceleratorCardsResponse>(`/apis/kantaloupe.dynamia.ai/v1/clusters/${req["cluster"]}/acceleratorcards?${fm.renderURLSearchParams(req, ["cluster"])}`, {...initReq, method: "GET"})
  }
  static GetAcceleratorCard(req: KantaloupeDynamiaAiApiAcceleratorcardV1alpha1Acceleratorcard.GetAcceleratorCardRequest, initReq?: fm.InitReq): Promise<KantaloupeDynamiaAiApiAcceleratorcardV1alpha1Acceleratorcard.AcceleratorCard> {
    return fm.fetchReq<KantaloupeDynamiaAiApiAcceleratorcardV1alpha1Acceleratorcard.GetAcceleratorCardRequest, KantaloupeDynamiaAiApiAcceleratorcardV1alpha1Acceleratorcard.AcceleratorCard>(`/apis/kantaloupe.dynamia.ai/v1/clusters/${req["cluster"]}/acceleratorcards/${req["uuid"]}?${fm.renderURLSearchParams(req, ["cluster", "uuid"])}`, {...initReq, method: "GET"})
  }
  static ListModelNames(req: KantaloupeDynamiaAiApiAcceleratorcardV1alpha1Acceleratorcard.ListModelNamesRequest, initReq?: fm.InitReq): Promise<KantaloupeDynamiaAiApiAcceleratorcardV1alpha1Acceleratorcard.ListModelNamesResponse> {
    return fm.fetchReq<KantaloupeDynamiaAiApiAcceleratorcardV1alpha1Acceleratorcard.ListModelNamesRequest, KantaloupeDynamiaAiApiAcceleratorcardV1alpha1Acceleratorcard.ListModelNamesResponse>(`/apis/kantaloupe.dynamia.ai/v1/clusters/${req["cluster"]}/acceleratorcards/modelnames?${fm.renderURLSearchParams(req, ["cluster"])}`, {...initReq, method: "GET"})
  }
}