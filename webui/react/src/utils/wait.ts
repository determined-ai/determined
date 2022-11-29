import { serverAddress } from 'routes/utils';
import { paths } from 'routes/utils';
import { V1LaunchWarning } from 'services/api-ts-sdk';
import { openBlank } from 'shared/utils/routes';
import { Command, CommandResponse, CommandState, CommandTask, CommandType } from 'types';
import { isCommandTask } from 'utils/task';

export interface WaitStatus {
  isReady: boolean;
  state: CommandState;
}

export const commandToEventUrl = (command: Command | CommandTask): string => {
  const kind = isCommandTask(command) ? command.type : command.kind;
  let path = '';
  switch (kind) {
    case CommandType.JupyterLab:
      path = `/notebooks/${command.id}/events`;
      break;
    case CommandType.TensorBoard:
      path = `/tensorboard/${command.id}/events?tail=1`;
      break;
  }
  if (path) path = serverAddress() + path;
  return path;
};

export const openCommand = (command: CommandTask): void => {
  openBlank(`${process.env.PUBLIC_URL}${paths.interactive(command, false)}`);
};

export const openCommandResponse = (commandResponse: CommandResponse): void => {
  const warnings = commandResponse?.warnings ? commandResponse.warnings : [];
  const maxSlotsExceeded = warnings.includes(V1LaunchWarning.CURRENTSLOTSEXCEEDED);
  openBlank(
    `${process.env.PUBLIC_URL}${paths.interactive(commandResponse.command, maxSlotsExceeded)}`,
  );
};

export const CANNOT_OPEN_COMMAND_ERROR = 'Command cannot be opened.';

export const waitPageUrl = (command: Command | CommandTask): string => {
  const url = commandToEventUrl(command);
  if (!url || !command.serviceAddress) throw new Error(CANNOT_OPEN_COMMAND_ERROR);

  const kind = isCommandTask(command) ? command.type : command.kind;
  const waitPath = `${process.env.PUBLIC_URL}/wait/${kind.toLowerCase()}/${command.id}`;
  const waitParams = `?eventUrl=${url}&serviceAddr=${command.serviceAddress}`;
  return waitPath + waitParams;
};
