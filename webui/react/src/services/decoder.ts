import dayjs from 'dayjs';

import {
  decode, ioCommandLogs, ioDeterminedInfo, ioExperimentConfig, ioExperimentDetails, ioExperiments,
  ioGenericCommand, ioLogs, ioTrialDetails, ioTypeAgents,
  ioTypeCheckpoint, ioTypeCommandAddress, ioTypeCommandLogs, ioTypeDeterminedInfo,
  ioTypeExperimentConfig,
  ioTypeExperimentDetails,
  ioTypeExperiments,
  ioTypeGenericCommand,
  ioTypeGenericCommands,
  ioTypeLogs,
  ioTypeTrialDetails,
  ioTypeTrialSummary,
  ioTypeUsers,
} from 'ioTypes';
import {
  Agent, Checkpoint, CheckpointState, Command, CommandState, CommandType,
  DeterminedInfo, Experiment, ExperimentConfig, ExperimentDetails, Log, LogLevel, ResourceState,
  ResourceType, RunState, TrialDetails, TrialSummary, User,
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

export const jsonToGenericCommand = (data: unknown, type: CommandType): Command => {
  const ioType = decode<ioTypeGenericCommand>(ioGenericCommand, data);
  const addresses = ioType.addresses ?
    ioType.addresses.map((address: ioTypeCommandAddress) => ({
      containerIp: address.container_ip,
      containerPort: address.container_port,
      hostIp: address.host_ip,
      hostPort: address.host_port,
      protocol: address.protocol,
    })) : undefined;
  const misc = ioType.misc ? {
    experimentIds: ioType.misc.experiment_ids || undefined,
    privateKey: ioType.misc.privateKey || undefined,
    trialIds: ioType.misc.trial_ids || undefined,
  } : undefined;

  return {
    addresses,
    config: { ...ioType.config },
    exitStatus: ioType.exit_status || undefined,
    id: ioType.id,
    kind: type,
    misc,
    owner: {
      id: ioType.owner.id,
      username: ioType.owner.username,
    },
    registeredTime: ioType.registered_time,
    serviceAddress: ioType.service_address || undefined,
    state: ioType.state as CommandState,
  };
};

const jsonToGenericCommands = (data: ioTypeGenericCommands, type: CommandType): Command[] => {
  return Object.keys(data).map(genericCommandId => {
    return jsonToGenericCommand(data[genericCommandId], type);
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

export const jsonToTensorboard = (data: unknown): Command => {
  return jsonToGenericCommand(data, CommandType.Tensorboard);
};

export const jsonToTensorboards = (data: ioTypeGenericCommands): Command[] => {
  return jsonToGenericCommands(data, CommandType.Tensorboard);
};

const jsonToExperimentConfig = (data: unknown): ExperimentConfig => {
  const io = decode<ioTypeExperimentConfig>(ioExperimentConfig, data);
  return {
    description: io.description,
    resources: {
      maxSlots: io.resources.max_slots,
    },
    searcher: {
      ...io.searcher,
      smallerIsBetter: io.searcher.smaller_is_better,
    },
  };

};

export const jsonToExperiments = (data: unknown): Experiment[] => {
  const ioType = decode<ioTypeExperiments>(ioExperiments, data);
  return ioType.map(experiment => {
    return {
      archived: experiment.archived,
      config: jsonToExperimentConfig(experiment.config),
      endTime: experiment.end_time || undefined,
      id: experiment.id,
      ownerId: experiment.owner_id,
      progress: experiment.progress !== null ? experiment.progress : undefined,
      startTime: experiment.start_time,
      state: experiment.state as RunState,
    };
  });
};

const ioCheckpoinToCheckpoint = (io: ioTypeCheckpoint): Checkpoint => {
  return { ...io,
    endTime: io.end_time || undefined,
    id: io.id,
    startTime: io.start_time,
    state: io.state as CheckpointState,
    stepId: io.step_id,
    trialId: io.trial_id,
    uuid: io.uuid || undefined,
    validationMetric: io.valiation_metric !== null ? io.valiation_metric : undefined,
  };
};

const ioTrialToTrial = (io: ioTypeTrialSummary): TrialSummary => {
  return { ...io,
    bestAvailableCheckpoint: io.best_available_checkpoint
      ? ioCheckpoinToCheckpoint(io.best_available_checkpoint) : undefined,
    numBatches: io.num_batches,
    numSteps: io.num_steps,
    state: io.state as RunState,// TODO add checkpoint decoder
  };
};

export const jsonToTrialDetails = (data: unknown): TrialDetails => {
  const io = decode<ioTypeTrialDetails>(ioTrialDetails, data);
  return {
    experimentId: io.experiment_id,
    id: io.id,
    state: io.state as RunState,
  };
};

export const jsonToExperimentDetails = (data: unknown): ExperimentDetails => {
  const ioType = decode<ioTypeExperimentDetails>(ioExperimentDetails, data);
  return {
    archived: ioType.archived,
    config: jsonToExperimentConfig(ioType.config),
    endTime: ioType.end_time || undefined,
    id: ioType.id,
    ownerId: ioType.owner.id,
    progress: ioType.progress !== null ? ioType.progress : undefined,
    startTime: ioType.start_time,
    state: ioType.state as RunState,
    trials: ioType.trials.map(ioTrialToTrial),
    username: ioType.owner.username,
    validationHistory: ioType.validation_history.map(vh => ({
      endTime: vh.end_time,
      id: vh.trial_id,
      validationError: vh.validation_error || undefined,
    })),
  };
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
    const matches = log.message.match(/\[([^\]]+)\] (.*)/);
    const time = matches && matches[1] ? matches[1] : undefined;
    const message = matches && matches[2] ? matches[2] : '';
    return { id: log.id, message, time };
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
