import { notification } from 'antd';
import { w3cwebsocket as W3CWebSocket } from 'websocket';

import { openBlank, serverAddress } from 'routes/utils';
import { Command, CommandState, CommandTask, CommandType } from 'types';
import { capitalize } from 'utils/string';
import { isCommandTask, terminalCommandStates } from 'utils/types';

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
    case CommandType.Notebook:
      path = `/notebooks/${command.id}/events`;
      break;
    case CommandType.Tensorboard:
      path = `/tensorboard/${command.id}/events?tail=1`;
      break;
  }
  if (path) path = serverAddress() + path;
  return path;
};

const waitPageUrl = (command: Command | CommandTask): string => {
  const url = commandToEventUrl(command);
  if (!url || !command.serviceAddress)
    throw new Error('command cannot be opened');
  const kind = isCommandTask(command) ? command.type : command.kind;

  const waitPath = `${process.env.PUBLIC_URL}/wait/${kind.toLowerCase()}/${command.id}`;
  const waitParams = `?eventUrl=${url}&serviceAddr=${command.serviceAddress}`;
  return waitPath + waitParams;
};

export const openCommand = (command: Command | CommandTask): void => {
  openBlank(waitPageUrl(command));
};

export interface WaitStatus {
  isReady: boolean;
  state: CommandState
}

// export const waitCommandReady = (eventUrl: string): Promise<WaitStatus> => {
//   return new Promise((resolve, reject) => {
//     const url = createWsUrl(eventUrl);
//     const client = new W3CWebSocket(url);

//     client.onmessage = (messageEvent) => {
//       if (typeof messageEvent.data !== 'string') return;
//       const msg = JSON.parse(messageEvent.data);
//       if (msg.snapshot) {
//         const state = msg.snapshot.state;
//         if (state === 'RUNNING' && msg.snapshot.is_ready) {
//           resolve({ isReady: true, state: CommandState.Running });
//           client.close();
//         } else if (terminalCommandStates.has(state)) {
//           resolve({ isReady: false, state });
//           client.close();
//         }
//       }
//     };

//     client.onclose = reject;
//     client.onerror = reject;
//   });
// };

// const readyOpenHandler = (command: Command | CommandTask) => {
//   return () => {
//     notification.close(command.id);
//     openBlank(command.serviceAddress as string);
//   };
// };

// export const openCommandNoWaitPage = async (command: Command | CommandTask): Promise<void> => {
//   const url = commandToEventUrl(command);
//   if (!url || !command.serviceAddress)
//     throw new Error('command cannot be opened');
//   const kind = isCommandTask(command) ? command.type : command.kind;
//   const name = isCommandTask(command) ? command.name : command.config.description;
//   const cmdKind = capitalize(kind);

//   notification.open({
//     description: `Waiting for ${name}`,
//     duration: 0,
//     key: command.id,
//     message: `Loading ${cmdKind}`,
//     placement: 'topRight',
//   });

//   const waitPath = `${process.env.PUBLIC_URL}/wait/${kind.toLowerCase()}/${command.id}`;
//   const waitParams = `?eventUrl=${url}&serviceAddr=${command.serviceAddress}`;
//   openBlank(waitPath + waitParams);

//   const waitStatus = await waitCommandReady(url);
//   notification.close(command.id);
//   if (waitStatus.isReady) {
//     notification.open({
//       description: `Click to open ${name}`,
//       duration: 0,
//       key: command.id,
//       message: `${cmdKind} Ready`,
//       onClick: readyOpenHandler(command),
//       placement: 'topRight',
//     });
//   }
//   if (terminalCommandStates.has(waitStatus.state)) {
//     notification.open({
//       description: `${name} is terminated.`,
//       duration: 0,
//       key: command.id,
//       message: `${cmdKind} Terminated`,
//       placement: 'topRight',
//     });

//   }
// };
