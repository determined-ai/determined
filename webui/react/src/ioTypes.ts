/* eslint-disable @typescript-eslint/camelcase */
import { isLeft } from 'fp-ts/lib/Either';
import * as io from 'io-ts';

import { CommandState, RunState } from 'types';

/* eslint-disable-next-line @typescript-eslint/no-explicit-any */
export const decode = <T>(type: io.Mixed, data: any): T => {
  const result = type.decode(data);
  if (isLeft(result)) throw result.left;
  return result.right;
};

/* User */

export const ioUser = io.type({
  active: io.boolean,
  admin: io.boolean,
  id: io.number,
  username: io.string,
});

export const ioUsers = io.array(ioUser);

export type ioTypeUser = io.TypeOf<typeof ioUser>;
export type ioTypeUsers = io.TypeOf<typeof ioUsers>;

/* Info */

export const ioDeterminedInfo = io.type({
  cluster_id: io.string,
  master_id: io.string,
  telemetry: io.type({
    enabled: io.boolean,
    segment_key: io.union([ io.string, io.undefined ]),
  }),
  version: io.string,
});

export type ioTypeDeterminedInfo = io.TypeOf<typeof ioDeterminedInfo>;

/* Slot */

export const ioSlotDevice = io.type({
  brand: io.string,
  id: io.number,
  type: io.string,
  uuid: io.union([ io.string, io.null ]),
});

export const ioSlotContainer = io.type({
  devices: io.array(ioSlotDevice),
  id: io.string,
  state: io.string,
});

export const ioSlot = io.type({
  container: io.union([ ioSlotContainer, io.null ]),
  device: ioSlotDevice,
  enabled: io.boolean,
  id: io.string,
});

export const ioSlots = io.record(io.string, ioSlot);

/* Agent */

export const ioAgent = io.type({
  id: io.string,
  registered_time: io.string,
  slots: ioSlots,
});

export const ioAgents = io.record(io.string, ioAgent);

export type ioTypeAgent = io.TypeOf<typeof ioAgent>;
export type ioTypeAgents = io.TypeOf<typeof ioAgents>;

/* Generic Command */

const ioOwner = io.type({
  id: io.number,
  username: io.string,
});

const ioCommandAddress = io.type({
  container_ip: io.string,
  container_port: io.number,
  host_ip: io.string,
  host_port: io.number,
  protocol: io.string,
});

const ioCommandMisc = io.partial({
  experiment_ids: io.union([ io.array(io.number), io.null ]),
  privateKey: io.string,
  trial_ids: io.union([ io.array(io.number), io.null ]),
});

const ioCommandConfig = io.exact(io.type({
  description: io.string,
}));

const commandStates: Record<string, null> = Object.values(CommandState)
  .reduce((acc, val) => ({ ...acc, [val]: null }), {});
const commandStatesIoType = io.keyof(commandStates);

export const ioGenericCommand = io.type({
  addresses: io.union([ io.array(ioCommandAddress), io.null ]),
  config: ioCommandConfig,
  exit_status: io.union([ io.string, io.null ]),
  id: io.string,
  misc: io.union([ ioCommandMisc, io.null ]),
  owner: ioOwner,
  registered_time: io.string,
  service_address: io.union([ io.string, io.null ]),
  state: commandStatesIoType,
});

export const ioGenericCommands = io.record(io.string, ioGenericCommand);

export type ioTypeCommandAddress = io.TypeOf<typeof ioCommandAddress>;
export type ioTypeGenericCommand = io.TypeOf<typeof ioGenericCommand>;
export type ioTypeGenericCommands = io.TypeOf<typeof ioGenericCommands>;

/* Experiments */

const ioExperimentConfig = io.type({
  description: io.string,
});

const runStates: Record<string, null> = Object.values(RunState)
  .reduce((acc, val) => ({ ...acc, [val]: null }), {});
const runStatesIoType = io.keyof(runStates);

export const ioExperiment = io.type({
  archived: io.boolean,
  config: ioExperimentConfig,
  end_time: io.union([ io.string, io.null ]),
  id: io.number,
  kind: io.union([ io.undefined, io.number ]),
  owner_id: io.number,
  progress: io.union([ io.number, io.null ]),
  start_time: io.string,
  state: runStatesIoType,
});

export const ioExperiments = io.type({
  data: io.type({
    experiments: io.array(ioExperiment),
  }),
});

export type ioTypeExperiment = io.TypeOf<typeof ioExperiment>;
export type ioTypeExperiments = io.TypeOf<typeof ioExperiments>;
