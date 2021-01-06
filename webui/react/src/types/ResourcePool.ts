// To parse this data:
//
//   import { Convert } from "./file";
//
//   const resourcePool = Convert.toResourcePool(json);
//
// These functions will throw an error if the JSON doesn't
// match the expected interface, even if the JSON is valid.

/* eslint-disable */
export interface ResourcePool {
  name: string;
  description: string;
  type: string;
  numAgents: number;
  slotsAvailable: number;
  slotsUsed: number;
  cpuContainerCapacity: number;
  cpuContainersRunning: number;
  defaultCpuPool: boolean;
  defaultGpuPool?: boolean;
  spotOrPreemptible: boolean;
  minInstances: number;
  maxInstances: number;
  gpusPerAgent: number;
  cpuContainerCapacityPerAgent: number;
  schedulerType: string;
  schedulerFittingPolicy: string;
  location: string;
  imageId: string;
  instanceType: string;
  details: Details;
}

export interface Details {
  poolName: string;
  description: string;
  maxCpuContainersPerAgent: number;
  provisionerType: string;
  masterUrl: string;
  masterCertName: string;
  startupScript: string;
  containerStartupScript: string;
  agentDockerRuntime: string;
  agentDockerImage: string;
  agentFluentImage: string;
  maxIdleAgentPeriod: string;
  maxAgentStartingPeriod: string;
  minInstances: number;
  maxInstances: number;
  aws?: Aws;
  schedulerType: string;
  schedulerFittingPolicy: string;
  gcp?: Gcp;
  priorityScheduler?: PriorityScheduler;
}

export interface Aws {
  region: string;
  rootVolumeSize: number;
  imageId: string;
  tagKey: string;
  tagValue: string;
  instanceName: string;
  sshKeyName: string;
  publicIp: boolean;
  subnetId: string;
  securityGroupId: string;
  iamInstanceProfileArn: string;
  instanceType: string;
  logGroup: string;
  logStream: string;
  spotEnabled: boolean;
  spotMaxPrice: string;
  customTags: CustomTag[];
}

export interface CustomTag {
  key: string;
  value: string;
}

export interface Gcp {
  project: string;
  zone: string;
  bootDiskSize: number;
  bootDiskSourceImage: string;
  labelKey: string;
  labelValue: string;
  namePrefix: string;
  network: string;
  subnetwork: string;
  externalIp: boolean;
  networkTags: string[];
  serviceAccountEmail: string;
  serviceAccountScopes: string[];
  machineType: string;
  gpuType: string;
  gpuNum: number;
  preemptible: boolean;
  operationTimeoutPeriod: string;
}

export interface PriorityScheduler {
  preemption: boolean;
  defaultPriority: number;
}

// Converts JSON strings to/from your types
// and asserts the results of JSON.parse at runtime
export class Convert {
  public static toResourcePool(json: string): ResourcePool[] {
    return cast(JSON.parse(json), a(r('ResourcePool')));
  }

  public static resourcePoolToJson(value: ResourcePool[]): string {
    return JSON.stringify(uncast(value, a(r('ResourcePool'))), null, 2);
  }
}

function invalidValue(typ: any, val: any, key: any = ''): never {
  if (key) {
    throw Error(`Invalid value for key "${key}". Expected type ${JSON.stringify(typ)} but got ${JSON.stringify(val)}`);
  }
  throw Error(`Invalid value ${JSON.stringify(val)} for type ${JSON.stringify(typ)}` );
}

function jsonToJSProps(typ: any): any {
  if (typ.jsonToJS === undefined) {
    const map: any = {};
    typ.props.forEach((p: any) => map[p.json] = { key: p.js, typ: p.typ });
    typ.jsonToJS = map;
  }
  return typ.jsonToJS;
}

function jsToJSONProps(typ: any): any {
  if (typ.jsToJSON === undefined) {
    const map: any = {};
    typ.props.forEach((p: any) => map[p.js] = { key: p.json, typ: p.typ });
    typ.jsToJSON = map;
  }
  return typ.jsToJSON;
}

function transform(val: any, typ: any, getProps: any, key: any = ''): any {
  function transformPrimitive(typ: string, val: any): any {
    if (typeof typ === typeof val) return val;
    return invalidValue(typ, val, key);
  }

  function transformUnion(typs: any[], val: any): any {
    // val must validate against one typ in typs
    const l = typs.length;
    for (let i = 0; i < l; i++) {
      const typ = typs[i];
      try {
        return transform(val, typ, getProps);
      } catch (_) {}
    }
    return invalidValue(typs, val);
  }

  function transformEnum(cases: string[], val: any): any {
    if (cases.indexOf(val) !== -1) return val;
    return invalidValue(cases, val);
  }

  function transformArray(typ: any, val: any): any {
    // val must be an array with no invalid elements
    if (!Array.isArray(val)) return invalidValue('array', val);
    return val.map(el => transform(el, typ, getProps));
  }

  function transformDate(val: any): any {
    if (val === null) {
      return null;
    }
    const d = new Date(val);
    if (isNaN(d.valueOf())) {
      return invalidValue('Date', val);
    }
    return d;
  }

  function transformObject(props: { [k: string]: any }, additional: any, val: any): any {
    if (val === null || typeof val !== 'object' || Array.isArray(val)) {
      return invalidValue('object', val);
    }
    const result: any = {};
    Object.getOwnPropertyNames(props).forEach(key => {
      const prop = props[key];
      const v = Object.prototype.hasOwnProperty.call(val, key) ? val[key] : undefined;
      result[prop.key] = transform(v, prop.typ, getProps, prop.key);
    });
    Object.getOwnPropertyNames(val).forEach(key => {
      if (!Object.prototype.hasOwnProperty.call(props, key)) {
        result[key] = transform(val[key], additional, getProps, key);
      }
    });
    return result;
  }

  if (typ === 'any') return val;
  if (typ === null) {
    if (val === null) return val;
    return invalidValue(typ, val);
  }
  if (typ === false) return invalidValue(typ, val);
  while (typeof typ === 'object' && typ.ref !== undefined) {
    typ = typeMap[typ.ref];
  }
  if (Array.isArray(typ)) return transformEnum(typ, val);
  if (typeof typ === 'object') {
    return typ.hasOwnProperty('unionMembers') ? transformUnion(typ.unionMembers, val)
      : typ.hasOwnProperty('arrayItems') ? transformArray(typ.arrayItems, val)
        : typ.hasOwnProperty('props') ? transformObject(getProps(typ), typ.additional, val)
          : invalidValue(typ, val);
  }
  // Numbers can be parsed by Date but shouldn't be.
  if (typ === Date && typeof val !== 'number') return transformDate(val);
  return transformPrimitive(typ, val);
}

function cast<T>(val: any, typ: any): T {
  return transform(val, typ, jsonToJSProps);
}

function uncast<T>(val: T, typ: any): any {
  return transform(val, typ, jsToJSONProps);
}

function a(typ: any) {
  return { arrayItems: typ };
}

function u(...typs: any[]) {
  return { unionMembers: typs };
}

function o(props: any[], additional: any) {
  return { additional, props };
}

function m(additional: any) {
  return { additional, props: [] };
}

function r(name: string) {
  return { ref: name };
}

const typeMap: any = {
  Aws: o([
    { js: 'region', json: 'region', typ: '' },
    { js: 'rootVolumeSize', json: 'rootVolumeSize', typ: 0 },
    { js: 'imageId', json: 'imageId', typ: '' },
    { js: 'tagKey', json: 'tagKey', typ: '' },
    { js: 'tagValue', json: 'tagValue', typ: '' },
    { js: 'instanceName', json: 'instanceName', typ: '' },
    { js: 'sshKeyName', json: 'sshKeyName', typ: '' },
    { js: 'publicIp', json: 'publicIp', typ: true },
    { js: 'subnetId', json: 'subnetId', typ: '' },
    { js: 'securityGroupId', json: 'securityGroupId', typ: '' },
    { js: 'iamInstanceProfileArn', json: 'iamInstanceProfileArn', typ: '' },
    { js: 'instanceType', json: 'instanceType', typ: '' },
    { js: 'logGroup', json: 'logGroup', typ: '' },
    { js: 'logStream', json: 'logStream', typ: '' },
    { js: 'spotEnabled', json: 'spotEnabled', typ: true },
    { js: 'spotMaxPrice', json: 'spotMaxPrice', typ: '' },
    { js: 'customTags', json: 'customTags', typ: a(r('CustomTag')) },
  ], false),
  CustomTag: o([
    { js: 'key', json: 'key', typ: '' },
    { js: 'value', json: 'value', typ: '' },
  ], false),
  Details: o([
    { js: 'poolName', json: 'poolName', typ: '' },
    { js: 'description', json: 'description', typ: '' },
    { js: 'maxCpuContainersPerAgent', json: 'maxCpuContainersPerAgent', typ: 0 },
    { js: 'provisionerType', json: 'provisionerType', typ: '' },
    { js: 'masterUrl', json: 'masterUrl', typ: '' },
    { js: 'masterCertName', json: 'masterCertName', typ: '' },
    { js: 'startupScript', json: 'startupScript', typ: '' },
    { js: 'containerStartupScript', json: 'containerStartupScript', typ: '' },
    { js: 'agentDockerRuntime', json: 'agentDockerRuntime', typ: '' },
    { js: 'agentDockerImage', json: 'agentDockerImage', typ: '' },
    { js: 'agentFluentImage', json: 'agentFluentImage', typ: '' },
    { js: 'maxIdleAgentPeriod', json: 'maxIdleAgentPeriod', typ: '' },
    { js: 'maxAgentStartingPeriod', json: 'maxAgentStartingPeriod', typ: '' },
    { js: 'minInstances', json: 'minInstances', typ: 0 },
    { js: 'maxInstances', json: 'maxInstances', typ: 0 },
    { js: 'aws', json: 'aws', typ: u(undefined, r('Aws')) },
    { js: 'schedulerType', json: 'schedulerType', typ: '' },
    { js: 'schedulerFittingPolicy', json: 'schedulerFittingPolicy', typ: '' },
    { js: 'gcp', json: 'gcp', typ: u(undefined, r('Gcp')) },
    { js: 'priorityScheduler', json: 'priorityScheduler', typ: u(undefined, r('PriorityScheduler')) },
  ], false),
  Gcp: o([
    { js: 'project', json: 'project', typ: '' },
    { js: 'zone', json: 'zone', typ: '' },
    { js: 'bootDiskSize', json: 'bootDiskSize', typ: 0 },
    { js: 'bootDiskSourceImage', json: 'bootDiskSourceImage', typ: '' },
    { js: 'labelKey', json: 'labelKey', typ: '' },
    { js: 'labelValue', json: 'labelValue', typ: '' },
    { js: 'namePrefix', json: 'namePrefix', typ: '' },
    { js: 'network', json: 'network', typ: '' },
    { js: 'subnetwork', json: 'subnetwork', typ: '' },
    { js: 'externalIp', json: 'externalIp', typ: true },
    { js: 'networkTags', json: 'networkTags', typ: a('') },
    { js: 'serviceAccountEmail', json: 'serviceAccountEmail', typ: '' },
    { js: 'serviceAccountScopes', json: 'serviceAccountScopes', typ: a('') },
    { js: 'machineType', json: 'machineType', typ: '' },
    { js: 'gpuType', json: 'gpuType', typ: '' },
    { js: 'gpuNum', json: 'gpuNum', typ: 0 },
    { js: 'preemptible', json: 'preemptible', typ: true },
    { js: 'operationTimeoutPeriod', json: 'operationTimeoutPeriod', typ: '' },
  ], false),
  PriorityScheduler: o([
    { js: 'preemption', json: 'preemption', typ: true },
    { js: 'defaultPriority', json: 'defaultPriority', typ: 0 },
  ], false),
  ResourcePool: o([
    { js: 'name', json: 'name', typ: '' },
    { js: 'description', json: 'description', typ: '' },
    { js: 'type', json: 'type', typ: '' },
    { js: 'numAgents', json: 'numAgents', typ: 0 },
    { js: 'slotsAvailable', json: 'slotsAvailable', typ: 0 },
    { js: 'slotsUsed', json: 'slotsUsed', typ: 0 },
    { js: 'cpuContainerCapacity', json: 'cpuContainerCapacity', typ: 0 },
    { js: 'cpuContainersRunning', json: 'cpuContainersRunning', typ: 0 },
    { js: 'defaultCpuPool', json: 'defaultCpuPool', typ: true },
    { js: 'defaultGpuPool', json: 'defaultGpuPool', typ: u(undefined, true) },
    { js: 'spotOrPreemptible', json: 'spotOrPreemptible', typ: true },
    { js: 'minInstances', json: 'minInstances', typ: 0 },
    { js: 'maxInstances', json: 'maxInstances', typ: 0 },
    { js: 'gpusPerAgent', json: 'gpusPerAgent', typ: 0 },
    { js: 'cpuContainerCapacityPerAgent', json: 'cpuContainerCapacityPerAgent', typ: 0 },
    { js: 'schedulerType', json: 'schedulerType', typ: '' },
    { js: 'schedulerFittingPolicy', json: 'schedulerFittingPolicy', typ: '' },
    { js: 'location', json: 'location', typ: '' },
    { js: 'imageId', json: 'imageId', typ: '' },
    { js: 'instanceType', json: 'instanceType', typ: '' },
    { js: 'details', json: 'details', typ: r('Details') },
  ], false),
};

/* eslint-enable */
