import { serverAddress } from 'routes/utils';
import { paths } from 'routes/utils';
import { windowOpenFeatures } from 'shared/utils/routes';
import { Command, CommandState, CommandTask, CommandType } from 'types';
import { isCommandTask } from 'utils/task';

// createWsUrl: Given an event url create the corresponding ws url.
export function createWsUrl(eventUrl: string): string {
  const isFullUrl = /^https?:\/\//i;

  if (isFullUrl.test(eventUrl)) {
    return eventUrl.replace(/^http/, 'ws');
  } else {
    // Remove the preceding slash if it is an absolute path.
    eventUrl = eventUrl.replace(/^\//, '');
    let url = window.location.protocol.replace(/^http/, 'ws');
    url += '//' + window.location.host + '/' + eventUrl;
    return url;
  }
}

const commandToEventUrl = (command: Command | CommandTask): string => {
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

export const waitPageUrl = (command: Command | CommandTask): string => {
  const url = commandToEventUrl(command);
  if (!url || !command.serviceAddress)
    throw new Error('command cannot be opened');
  const kind = isCommandTask(command) ? command.type : command.kind;

  const waitPath = `${process.env.PUBLIC_URL}/wait/${kind.toLowerCase()}/${command.id}`;
  const waitParams = `?eventUrl=${url}&serviceAddr=${command.serviceAddress}`;
  return waitPath + waitParams;
};

export const openCommand = (command: CommandTask): void => {
  window.open(
    process.env.PUBLIC_URL + paths.interactive(command),
    '_blank',
    windowOpenFeatures.join(','),
  );
};

export interface WaitStatus {
  isReady: boolean;
  state: CommandState
}
