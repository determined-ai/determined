import { resetServerAddress, setServerAddress } from 'dev';
import { userPreferencesStorage } from 'hooks/useStorage';
import { alertAction } from 'omnibar/tree-extension/trees/actions';
import { Children, TreeNode } from 'omnibar/tree-extension/types';
import { checkServerAlive, serverAddress } from 'routes/utils';

const dev: TreeNode[] = [
  {
    options: [
      {
        onAction: alertAction(`address: ${serverAddress()}`),
        title: 'show',
      },
      {
        label: 'set <URL>',
        onCustomInput: (inp: string): Children => {
          return [ {
            closeBar: true,
            label: inp || '<URL>',
            onAction: async () => {
              const isAlive = await checkServerAlive(inp);
              if (isAlive) {
                setServerAddress(inp);
              } else {
                alertAction(`Could not find a valid server at ${inp}`)();
              }
            },
            title: inp,
          } ];
        },
        title: 'set',
      },
      {
        onAction: () => resetServerAddress(),
        title: 'reset',
      },
    ],
    title: 'serverAddress',
  },
  {
    onAction: () => window.localStorage.clear(),
    title: 'resetLocalStorage',
  },
  {
    onAction: () => userPreferencesStorage.reset(),
    title: 'resetUserPreferences',
  },
];

export default dev;
