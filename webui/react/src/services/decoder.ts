import dayjs from 'dayjs';

import {
  ioTypeAgents, ioTypeCommandAddress, ioTypeDeterminedInfo,
  ioTypeExperiments, ioTypeGenericCommand, ioTypeGenericCommands, ioTypeUsers,
} from 'ioTypes';
import {
  Agent, Command, CommandType, DeterminedInfo, Experiment, ResourceState, ResourceType, User,
} from 'types';
import { capitalize } from 'utils/string';

export const jsonToDeterminedInfo = (data: ioTypeDeterminedInfo): DeterminedInfo => {
  return {
    clusterId: data.cluster_id,
    masterId: data.master_id,
    telemetry: {
      enabled: data.telemetry.enabled,
      segmentKey: data.telemetry.segment_key,
    },
    version: data.version,
  };
};

export const jsonToUsers = (data: ioTypeUsers): User[] => {
  return data.map(user => ({
    id: user.id,
    isActive: user.active,
    isAdmin: user.admin,
    username: user.username,
  }));
};

export const jsonToAgents = (data: ioTypeAgents): Agent[] => {
  return Object.keys(data).map(agentId => {
    const agent = data[agentId];
    const resources = Object.keys(agent.slots).map(slotId => {
      const slot = agent.slots[slotId];

      return {
        container: slot.container ? {
          id: slot.container.id,
          state: ResourceState[
            capitalize(slot.container.state) as keyof typeof ResourceState
          ],
        } : undefined,
        enabled: slot.enabled,
        id: slot.id,
        name: slot.device.brand,
        type: ResourceType[slot.device.type.toUpperCase() as keyof typeof ResourceType],
        uuid: slot.device.uuid || undefined,
      };
    });

    return {
      id: agent.id,
      registeredTime: dayjs(agent.registered_time).unix(),
      resources,
    };
  });
};

const jsonToGenericCommands = (data: ioTypeGenericCommands, type: CommandType): Command[] => {
  return Object.keys(data).map(genericCommandId => {
    const command: ioTypeGenericCommand = data[genericCommandId];
    const addresses = command.addresses ?
      command.addresses.map((address: ioTypeCommandAddress) => ({
        containerIp: address.container_ip,
        containerPort: address.container_port,
        hostIp: address.host_ip,
        hostPort: address.host_port,
        protocol: address.protocol,
      })) : undefined;
    const misc = command.misc ? {
      experimentIds: command.misc.experiment_ids || undefined,
      privateKey: command.misc.privateKey || undefined,
      trialIds: command.misc.trial_ids || undefined,
    } : undefined;

    return {
      addresses,
      config: { ...command.config },
      exitStatus: command.exit_status || undefined,
      id: command.id,
      kind: type,
      misc,
      owner: {
        id: command.owner.id,
        username: command.owner.username,
      },
      registeredTime: command.registered_time,
      serviceAddress: command.service_address || undefined,
      state: command.state,
    };
  });
};

export const jsonToCommands = (data: ioTypeGenericCommands): Command[] => {
  return jsonToGenericCommands(data, CommandType.Command);
};

export const jsonToNotebooks = (data: ioTypeGenericCommands): Command[] => {
  return jsonToGenericCommands(data, CommandType.Notebook);
};

export const jsonToShells = (data: ioTypeGenericCommands): Command[] => {
  return jsonToGenericCommands(data, CommandType.Shell);
};

export const jsonToTensorboards = (data: ioTypeGenericCommands): Command[] => {
  return jsonToGenericCommands(data, CommandType.Tensorboard);
};

export const jsonToExperiments = (data: ioTypeExperiments): Experiment[] => {
  return (data.data.experiments || []).map(experiment => {
    return {
      archived: experiment.archived,
      config: experiment.config,
      endTime: experiment.end_time || undefined,
      id: experiment.id,
      ownerId: experiment.owner_id,
      progress: experiment.progress || undefined,
      startTime: experiment.start_time,
      state: experiment.state,
    };
  });
};
