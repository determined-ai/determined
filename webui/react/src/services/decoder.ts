import dayjs from 'dayjs';

import {
  decode, ioCommandLogs, ioDeterminedInfo, ioExperiments, ioLogs, ioTypeAgents,
  ioTypeCommandAddress, ioTypeCommandLogs, ioTypeDeterminedInfo, ioTypeExperiments,
  ioTypeGenericCommand, ioTypeGenericCommands, ioTypeLogs, ioTypeUsers,
} from 'ioTypes';
import {
  Agent, Command, CommandState, CommandType, DeterminedInfo, Experiment,
  Log, LogLevel, ResourceState, ResourceType, RunState, User,
} from 'types';
import { capitalize } from 'utils/string';

export const jsonToUsers = (data: ioTypeUsers): User[] => {
  return data.map(user => ({
    id: user.id,
    isActive: user.active,
    isAdmin: user.admin,
    username: user.username,
  }));
};

export const jsonToDeterminedInfo = (data: unknown): DeterminedInfo => {
  const info = decode<ioTypeDeterminedInfo>(ioDeterminedInfo, data);
  return {
    clusterId: info.cluster_id,
    masterId: info.master_id,
    telemetry: {
      enabled: info.telemetry.enabled,
      segmentKey: info.telemetry.segment_key,
    },
    version: info.version,
  };
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
      state: command.state as CommandState,
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

export const jsonToExperiments = (data: unknown): Experiment[] => {
  const ioType = decode<ioTypeExperiments>(ioExperiments, data);
  return ioType.map(experiment => {
    return {
      archived: experiment.archived,
      config: experiment.config,
      endTime: experiment.end_time || undefined,
      id: experiment.id,
      ownerId: experiment.owner_id,
      progress: experiment.progress || undefined,
      startTime: experiment.start_time,
      state: experiment.state as RunState,
    };
  });
};

export const jsonToLogs = (data: unknown): Log[] => {
  const ioType = decode<ioTypeLogs>(ioLogs, data);
  return ioType.map(log => ({
    id: log.id,
    level: log.level ? LogLevel[capitalize(log.level) as keyof typeof LogLevel] : undefined,
    message: log.message,
    time: log.time,
  }));
};

export const jsonToTrialLogs = (data: unknown): Log[] => {
  const ioType = decode<ioTypeLogs>(ioLogs, data);
  return ioType.map(log => {
    const [ header, message ] = log.message.split(' || ', 2);
    const [ time, meta ] = header.split(' ', 2);
    return {
      id: log.id,
      message,
      meta,
      time,
    };
  });
};

export const jsonToCommandLogs = (data: unknown): Log[] => {
  const ioType = decode<ioTypeCommandLogs>(ioCommandLogs, data);
  return ioType.map(log => ({
    id: log.seq,
    message: log.snapshot.config.description,
    time: log.time,
  }));
};
